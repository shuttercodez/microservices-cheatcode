package product

import (
	"errors"

	"github.com/gofiber/fiber/v2"
)

// Handler จัดการ HTTP Request / Response ของ Fiber
type Handler struct {
	service Service
}

// NewHandler สร้าง Fiber Handler สำหรับ Product
func NewHandler(service Service) *Handler {
	return &Handler{service: service}
}

// RegisterRoutes ลงทะเบียน RESTful API Endpoints เข้ากับ Fiber Router
func (h *Handler) RegisterRoutes(router fiber.Router) {
	v1 := router.Group("/api/v1/products")

	v1.Post("/", h.Create)
	v1.Get("/", h.GetAll)
	v1.Get("/:id", h.GetByID)
	v1.Put("/:id", h.Update)
	v1.Patch("/:id/stock", h.PatchStock)
	v1.Delete("/:id", h.Delete)
}

// Create POST: สร้างสินค้าใหม่
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

// GetAll GET (All): ดึงรายการสินค้าทั้งหมด
func (h *Handler) GetAll(c *fiber.Ctx) error {
	products, err := h.service.GetAll(c.UserContext())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(fiber.StatusOK).JSON(products)
}

// GetByID GET (By ID): ดูสินค้าตาม ID
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

// Update PUT: แก้ไขข้อมูลสินค้าทั้งก้อน (Full Replace)
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

// PatchStock PATCH: อัปเดตเฉพาะจำนวนสต็อก (Partial Update + Optimistic Locking)
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

// Delete DELETE: ลบสินค้า (Soft Delete ด้วย GORM)
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
