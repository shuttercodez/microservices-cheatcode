package order

import (
	"errors"

	"go-microservice-starter/internal/product"

	"github.com/gofiber/fiber/v2"
)

// Handler manages HTTP requests and responses for order endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new Fiber HTTP handler for orders.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers RESTful order routes under /api/v1/orders.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	v1 := router.Group("/api/v1/orders")

	v1.Post("/", h.Create)
	v1.Get("/", h.GetAll)
	v1.Get("/:id", h.GetByID)
}

// Create handles POST requests to place a new order and automatically deduct product stock.
func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	order, err := h.service.CreateOrder(c.UserContext(), req)
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidOrderInput):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, product.ErrProductNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, ErrInsufficientStock), errors.Is(err, product.ErrVersionConflict):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(fiber.StatusCreated).JSON(order)
}

// GetAll handles GET requests to retrieve a paginated list of orders (?page=1&limit=10).
func (h *Handler) GetAll(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	orders, err := h.service.GetAllOrders(c.UserContext(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(orders)
}

// GetByID handles GET requests to retrieve a single order by its ID.
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	order, err := h.service.GetOrderByID(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, ErrOrderNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(order)
}
