package auth

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"go-microservice-starter/internal/config"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrInvalidToken       = errors.New("invalid or expired JWT token")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserAlreadyExists  = errors.New("username already exists")
	ErrInvalidInput       = errors.New("username and password (min 6 chars) are required")
)

// User represents a registered user stored in PostgreSQL with a Bcrypt password hash.
type User struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey" json:"id"`
	Username  string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"username"`
	Role      string    `gorm:"type:varchar(32);not null;default:'user'" json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// RegisterRequest represents the payload for registering a new user account (POST /oauth/register).
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

// Claims represents standard RFC 7519 JWT payload claims.
type Claims struct {
	Sub   string `json:"sub"`
	Role  string `json:"role,omitempty"`
	Scope string `json:"scope"`
	Iss   string `json:"iss"`
	Iat   int64  `json:"iat"`
	Exp   int64  `json:"exp"`
}

// TokenRequest represents an OAuth 2.0 token request body (supports JSON and form-urlencoded).
type TokenRequest struct {
	GrantType    string `json:"grant_type" form:"grant_type"`
	Username     string `json:"username" form:"username"`
	Password     string `json:"password" form:"password"`
	ClientID     string `json:"client_id" form:"client_id"`
	ClientSecret string `json:"client_secret" form:"client_secret"`
	Scope        string `json:"scope" form:"scope"`
}

// TokenResponse represents a standard RFC 6749 OAuth 2.0 Access Token response.
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// Repository defines database operations for OAuth 2.0 users and M2M clients using PostgreSQL pgcrypto Bcrypt.
type Repository interface {
	CreateUser(ctx context.Context, username, password, role string) (*User, error)
	AuthenticateUser(ctx context.Context, username, password string) (*User, error)
	AuthenticateClient(ctx context.Context, clientID, clientSecret string) (string, error)
}

type gormRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) Repository {
	return &gormRepository{db: db}
}

func (r *gormRepository) CreateUser(ctx context.Context, username, password, role string) (*User, error) {
	if role == "" {
		role = "user"
	}
	var user User
	query := `
		INSERT INTO users (username, password_hash, role)
		VALUES (?, crypt(?, gen_salt('bf', 10)), ?)
		RETURNING id, username, role, created_at, updated_at
	`
	if err := r.db.WithContext(ctx).Raw(query, username, password, role).Scan(&user).Error; err != nil || user.ID == uuid.Nil {
		return nil, ErrUserAlreadyExists
	}
	return &user, nil
}

func (r *gormRepository) AuthenticateUser(ctx context.Context, username, password string) (*User, error) {
	var user User
	query := `
		SELECT id, username, role, created_at, updated_at
		FROM users
		WHERE username = ? AND password_hash = crypt(?, password_hash)
	`
	if err := r.db.WithContext(ctx).Raw(query, username, password).Scan(&user).Error; err != nil || user.ID == uuid.Nil {
		return nil, ErrInvalidCredentials
	}
	return &user, nil
}

func (r *gormRepository) AuthenticateClient(ctx context.Context, clientID, clientSecret string) (string, error) {
	var result struct {
		ClientID string
		Scope    string
	}
	query := `
		SELECT client_id, scope
		FROM oauth_clients
		WHERE client_id = ? AND client_secret_hash = crypt(?, client_secret_hash)
	`
	if err := r.db.WithContext(ctx).Raw(query, clientID, clientSecret).Scan(&result).Error; err != nil || result.ClientID == "" {
		return "", ErrInvalidCredentials
	}
	return result.Scope, nil
}

// GenerateJWT creates an HS256 signed JWT string.
func GenerateJWT(secret, subject, role, scope string, ttl time.Duration) (string, error) {
	headerJSON := []byte(`{"alg":"HS256","typ":"JWT"}`)
	now := time.Now().UTC()
	claims := Claims{
		Sub:   subject,
		Role:  role,
		Scope: scope,
		Iss:   "inventory-order-service",
		Iat:   now.Unix(),
		Exp:   now.Add(ttl).Unix(),
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)
	unsigned := headerB64 + "." + claimsB64

	sig := signHS256(unsigned, secret)
	return unsigned + "." + sig, nil
}

// VerifyJWT validates an HS256 JWT signature and expiration time.
func VerifyJWT(tokenStr, secret string) (*Claims, error) {
	parts := strings.Split(tokenStr, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	unsigned := parts[0] + "." + parts[1]
	expectedSig := signHS256(unsigned, secret)
	if !hmac.Equal([]byte(parts[2]), []byte(expectedSig)) {
		return nil, ErrInvalidToken
	}

	claimsBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(claimsBytes, &claims); err != nil {
		return nil, ErrInvalidToken
	}

	if time.Now().UTC().Unix() > claims.Exp {
		return nil, ErrInvalidToken
	}

	return &claims, nil
}

func signHS256(unsigned, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(unsigned))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Handler manages OAuth 2.0 user registration, token issuance, and JWT verification middleware.
type Handler struct {
	cfg  config.AuthConfig
	repo Repository
}

func NewHandler(cfg config.AuthConfig, repo Repository) *Handler {
	return &Handler{cfg: cfg, repo: repo}
}

func (h *Handler) RegisterRoutes(router fiber.Router) {
	router.Get("/oauth/config", h.GetConfig)
	router.Post("/oauth/register", h.Register)
	router.Post("/oauth/token", h.IssueToken)
}

// GetConfig returns public OAuth configuration flags (such as whether user registration is enabled).
func (h *Handler) GetConfig(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"enable_register": h.cfg.EnableRegister,
	})
}

// Register creates a new user account in PostgreSQL with a Bcrypt password hash (when enable_register is true).
func (h *Handler) Register(c *fiber.Ctx) error {
	if !h.cfg.EnableRegister {
		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"error": "user registration is disabled",
		})
	}

	var req RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	req.Username = strings.TrimSpace(req.Username)
	if req.Username == "" || len(req.Password) < 6 {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": ErrInvalidInput.Error()})
	}

	user, err := h.repo.CreateUser(c.UserContext(), req.Username, req.Password, req.Role)
	if err != nil {
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

// IssueToken implements the OAuth 2.0 Token Endpoint verifying credentials against PostgreSQL.
func (h *Handler) IssueToken(c *fiber.Ctx) error {
	var req TokenRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "invalid_request",
			"error_description": "cannot parse OAuth 2.0 token request",
		})
	}

	// Support HTTP Basic Auth header for client_id / client_secret
	if authHeader := c.Get("Authorization"); strings.HasPrefix(authHeader, "Basic ") {
		raw, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(authHeader, "Basic "))
		if err == nil {
			if pair := strings.SplitN(string(raw), ":", 2); len(pair) == 2 {
				req.ClientID = pair[0]
				req.ClientSecret = pair[1]
			}
		}
	}

	scope := req.Scope
	if scope == "" {
		scope = "inventory:read inventory:write orders:read orders:write"
	}

	var subject, role string
	switch req.GrantType {
	case "password":
		user, err := h.repo.AuthenticateUser(c.UserContext(), req.Username, req.Password)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":             "invalid_grant",
				"error_description": "invalid username or password",
			})
		}
		subject = user.Username
		role = user.Role
	case "client_credentials":
		clientScope, err := h.repo.AuthenticateClient(c.UserContext(), req.ClientID, req.ClientSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error":             "invalid_client",
				"error_description": "invalid client_id or client_secret",
			})
		}
		subject = req.ClientID
		role = "service"
		if req.Scope == "" && clientScope != "" {
			scope = clientScope
		}
	default:
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":             "unsupported_grant_type",
			"error_description": "supported grant_types are 'password' and 'client_credentials'",
		})
	}

	ttl := h.cfg.TokenTTL()
	token, err := GenerateJWT(h.cfg.JWTSecret, subject, role, scope, ttl)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "server_error",
		})
	}

	return c.Status(fiber.StatusOK).JSON(TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   int(ttl.Seconds()),
		Scope:       scope,
	})
}

// Middleware validates the Bearer JWT on protected API routes.
func (h *Handler) Middleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing or invalid Authorization Bearer token",
			})
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := VerifyJWT(tokenStr, h.cfg.JWTSecret)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": err.Error(),
			})
		}

		c.Locals("sub", claims.Sub)
		c.Locals("role", claims.Role)
		c.Locals("scope", claims.Scope)
		return c.Next()
	}
}
