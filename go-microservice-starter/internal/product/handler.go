package product

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// Handler manages HTTP requests and responses for product endpoints.
type Handler struct {
	service Service
}

// NewHandler creates a new Fiber HTTP handler for products.
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes registers RESTful product routes on the Fiber router.
func (h *Handler) RegisterRoutes(router fiber.Router) {
	v1 := router.Group("/api/v1/products")

	v1.Post("/", h.Create)
	v1.Get("/", h.GetAll)
	v1.Get("/:id", h.GetByID)
	v1.Put("/:id", h.Update)
	v1.Patch("/:id/stock", h.PatchStock)
	v1.Delete("/:id", h.Delete)
}

// Create handles POST requests to create a new product.
func (h *Handler) Create(c *fiber.Ctx) error {
	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	product, err := h.service.Create(c.UserContext(), req)
	if err != nil {
		if errors.Is(err, ErrInvalidPayload) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(product)
}

// GetAll handles GET requests to retrieve a paginated list of products (?page=1&limit=10).
func (h *Handler) GetAll(c *fiber.Ctx) error {
	page := c.QueryInt("page", 1)
	limit := c.QueryInt("limit", 10)

	products, err := h.service.GetAll(c.UserContext(), page, limit)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(products)
}

// GetByID handles GET requests to retrieve a product by its ID.
func (h *Handler) GetByID(c *fiber.Ctx) error {
	id := c.Params("id")
	product, err := h.service.GetByID(c.UserContext(), id)
	if err != nil {
		if errors.Is(err, ErrProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(product)
}

// Update handles PUT requests for a full replacement of product details.
func (h *Handler) Update(c *fiber.Ctx) error {
	id := c.Params("id")

	var req CreateProductRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	product, err := h.service.Update(c.UserContext(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrProductNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, ErrInvalidPayload):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(fiber.StatusOK).JSON(product)
}

// PatchStock handles PATCH requests for partial stock updates using optimistic locking.
func (h *Handler) PatchStock(c *fiber.Ctx) error {
	id := c.Params("id")

	var req PatchStockRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "cannot parse JSON"})
	}

	product, err := h.service.PatchStock(c.UserContext(), id, req)
	if err != nil {
		switch {
		case errors.Is(err, ErrNegativeStock):
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, ErrProductNotFound):
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		case errors.Is(err, ErrVersionConflict):
			return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
		default:
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}
	}

	return c.Status(fiber.StatusOK).JSON(product)
}

// Delete handles DELETE requests to soft-delete a product via GORM.
func (h *Handler) Delete(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.service.Delete(c.UserContext(), id); err != nil {
		if errors.Is(err, ErrProductNotFound) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
