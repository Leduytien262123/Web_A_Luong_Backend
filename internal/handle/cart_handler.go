package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type CartHandler struct {
	cartRepo *repo.CartRepo
}

func NewCartHandler() *CartHandler {
	return &CartHandler{
		cartRepo: repo.NewCartRepo(),
	}
}

// GetCart retrieves user's cart
func (h *CartHandler) GetCart(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	cart, err := h.cartRepo.GetOrCreateCart(userID.(uuid.UUID))
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve cart", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cart retrieved successfully",
		Data:    cart.ToResponse(),
	})
}

// AddToCart adds item to cart
func (h *CartHandler) AddToCart(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	var input model.CartItemInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	if err := h.cartRepo.AddItem(userID.(uuid.UUID), input.ProductID, input.Quantity); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to add item to cart", err)
		return
	}

	// Get updated cart
	cart, err := h.cartRepo.GetOrCreateCart(userID.(uuid.UUID))
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve updated cart", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Item added to cart successfully",
		Data:    cart.ToResponse(),
	})
}

// UpdateCartItem updates cart item quantity
func (h *CartHandler) UpdateCartItem(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	var input struct {
		Quantity int `json:"quantity" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	if err := h.cartRepo.UpdateItemQuantity(userID.(uuid.UUID), productID, input.Quantity); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to update cart item", err)
		return
	}

	// Get updated cart
	cart, err := h.cartRepo.GetOrCreateCart(userID.(uuid.UUID))
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve updated cart", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cart item updated successfully",
		Data:    cart.ToResponse(),
	})
}

// RemoveFromCart removes item from cart
func (h *CartHandler) RemoveFromCart(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	productIDStr := c.Param("product_id")
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid product ID", err)
		return
	}

	if err := h.cartRepo.RemoveItem(userID.(uuid.UUID), productID); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to remove item from cart", err)
		return
	}

	// Get updated cart
	cart, err := h.cartRepo.GetOrCreateCart(userID.(uuid.UUID))
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve updated cart", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Item removed from cart successfully",
		Data:    cart.ToResponse(),
	})
}

// ClearCart removes all items from cart
func (h *CartHandler) ClearCart(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	if err := h.cartRepo.ClearCart(userID.(uuid.UUID)); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to clear cart", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cart cleared successfully",
		Data:    nil,
	})
}