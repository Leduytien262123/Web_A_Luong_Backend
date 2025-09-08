package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"
	"strconv"
	"time"

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

type NewsHandler struct {
	newsRepo *repo.NewsRepo
}

func NewNewsHandler() *NewsHandler {
	return &NewsHandler{
		newsRepo: repo.NewNewsRepo(),
	}
}

// CreateNews creates a new news article
func (h *NewsHandler) CreateNews(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		helpers.UnauthorizedResponse(c, "Unauthorized")
		return
	}

	var input model.NewsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Check if slug already exists
	exists, err := h.newsRepo.CheckSlugExists(input.Slug, uuid.Nil)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug already exists", nil)
		return
	}

	news := model.News{
		Title:       input.Title,
		Slug:        input.Slug,
		Summary:     input.Summary,
		Content:     input.Content,
		ImageURL:    input.ImageURL,
		AuthorID:    userID.(uuid.UUID),
		IsPublished: input.IsPublished,
		Tags:        input.Tags,
		MetaTitle:   input.MetaTitle,
		MetaDesc:    input.MetaDesc,
	}

	if input.IsPublished {
		now := time.Now()
		news.PublishedAt = &now
	}

	if err := h.newsRepo.Create(&news); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to create news", err)
		return
	}

	// Load created news with author
	createdNews, err := h.newsRepo.GetByID(news.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to load created news", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "News created successfully",
		Data:    createdNews.ToResponse(),
	})
}

// GetNews retrieves news with pagination
func (h *NewsHandler) GetNews(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	publishedOnly := c.Query("published") == "true"

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	news, total, err := h.newsRepo.GetAll(page, limit, publishedOnly)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve news", err)
		return
	}

	var response []model.NewsResponse
	for _, article := range news {
		response = append(response, article.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "News retrieved successfully",
		Data: map[string]interface{}{
			"news":        response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"has_next":    page < int(totalPages),
			"has_prev":    page > 1,
		},
	})
}

// GetNewsByID retrieves a news article by ID
func (h *NewsHandler) GetNewsByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid news ID", err)
		return
	}

	news, err := h.newsRepo.GetByID(id)
	if err != nil {
		if err.Error() == "news not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "News not found", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	}

	// Increment view count
	h.newsRepo.IncrementViewCount(id)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "News retrieved successfully",
		Data:    news.ToResponse(),
	})
}

// GetNewsBySlug retrieves a news article by slug
func (h *NewsHandler) GetNewsBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid slug", nil)
		return
	}

	news, err := h.newsRepo.GetBySlug(slug)
	if err != nil {
		if err.Error() == "news not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "News not found", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	}

	// Increment view count
	h.newsRepo.IncrementViewCount(news.ID)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "News retrieved successfully",
		Data:    news.ToResponse(),
	})
}

// UpdateNews updates a news article
func (h *NewsHandler) UpdateNews(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid news ID", err)
		return
	}

	var input model.NewsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid input", err)
		return
	}

	// Get existing news
	news, err := h.newsRepo.GetByID(id)
	if err != nil {
		if err.Error() == "news not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "News not found", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	}

	// Check if slug already exists (excluding current news)
	exists, err := h.newsRepo.CheckSlugExists(input.Slug, id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug already exists", nil)
		return
	}

	// Update news
	news.Title = input.Title
	news.Slug = input.Slug
	news.Summary = input.Summary
	news.Content = input.Content
	news.ImageURL = input.ImageURL
	news.IsPublished = input.IsPublished
	news.Tags = input.Tags
	news.MetaTitle = input.MetaTitle
	news.MetaDesc = input.MetaDesc

	if input.IsPublished && news.PublishedAt == nil {
		now := time.Now()
		news.PublishedAt = &now
	}

	if err := h.newsRepo.Update(news); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to update news", err)
		return
	}

	// Load updated news with author
	updatedNews, err := h.newsRepo.GetByID(news.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to load updated news", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "News updated successfully",
		Data:    updatedNews.ToResponse(),
	})
}

// DeleteNews deletes a news article
func (h *NewsHandler) DeleteNews(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid news ID", err)
		return
	}

	// Check if news exists
	_, err = h.newsRepo.GetByID(id)
	if err != nil {
		if err.Error() == "news not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "News not found", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
		return
	}

	if err := h.newsRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to delete news", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "News deleted successfully",
		Data:    nil,
	})
}