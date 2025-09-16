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

type NewsHandler struct {
	newsRepo         *repo.NewsRepo
	newsCategoryRepo *repo.NewsCategoryRepo
	tagRepo          *repo.TagRepo
}

func NewNewsHandler() *NewsHandler {
	return &NewsHandler{
		newsRepo:         repo.NewNewsRepo(),
		newsCategoryRepo: repo.NewNewsCategoryRepo(),
		tagRepo:          repo.NewTagRepo(),
	}
}

// CreateNews creates a new news article with tags and categories
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

	news := &model.News{
		Title:       input.Title,
		Slug:        input.Slug,
		Summary:     input.Summary,
		Content:     input.Content,
		ImageURL:    input.ImageURL,
		AuthorID:    userID.(uuid.UUID),
		CategoryID:  input.CategoryID,
		IsPublished: input.IsPublished,
		IsFeatured:  input.IsFeatured,
		MetaTitle:   input.MetaTitle,
		MetaDesc:    input.MetaDesc,
	}

	if input.IsPublished {
		now := time.Now()
		news.PublishedAt = &now
	}

	// Create news with associations
	if err := h.newsRepo.CreateWithAssociations(news, input.TagIDs, input.CategoryIDs); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to create news", err)
		return
	}

	// Load created news with relationships
	createdNews, err := h.newsRepo.GetByID(news.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to load created news", err)
		return
	}

	helpers.SuccessResponse(c, "News created successfully", createdNews.ToResponse())
}

// GetNews retrieves news with pagination and filters
func (h *NewsHandler) GetNews(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	publishedOnly := c.Query("published") == "true"
	categoryID := c.Query("category_id")
	tagID := c.Query("tag_id")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var news []model.News
	var total int64
	var err error

	// Filter by category
	if categoryID != "" {
		if catID, parseErr := uuid.Parse(categoryID); parseErr == nil {
			news, total, err = h.newsRepo.GetByCategory(catID, page, limit)
		} else {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid category ID", parseErr)
			return
		}
	} else if tagID != "" {
		// Filter by tag
		if tID, parseErr := uuid.Parse(tagID); parseErr == nil {
			news, total, err = h.newsRepo.GetByTag(tID, page, limit)
		} else {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Invalid tag ID", parseErr)
			return
		}
	} else {
		// Get all news
		news, total, err = h.newsRepo.GetAll(page, limit, publishedOnly)
	}

	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to retrieve news", err)
		return
	}

	var responses []model.NewsResponse
	for _, article := range news {
		responses = append(responses, article.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	response := map[string]interface{}{
		"news":        responses,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"has_next":    page < int(totalPages),
		"has_prev":    page > 1,
	}

	helpers.SuccessResponse(c, "News retrieved successfully", response)
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

	helpers.SuccessResponse(c, "News retrieved successfully", news.ToResponse())
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

	helpers.SuccessResponse(c, "News retrieved successfully", news.ToResponse())
}

// UpdateNews updates a news article with tags and categories
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
	if input.Slug != news.Slug {
		exists, err := h.newsRepo.CheckSlugExists(input.Slug, id)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Database error", err)
			return
		}
		if exists {
			helpers.ErrorResponse(c, http.StatusConflict, "Slug already exists", nil)
			return
		}
	}

	// Update news fields
	news.Title = input.Title
	news.Slug = input.Slug
	news.Summary = input.Summary
	news.Content = input.Content
	news.ImageURL = input.ImageURL
	news.CategoryID = input.CategoryID
	news.IsPublished = input.IsPublished
	news.IsFeatured = input.IsFeatured
	news.MetaTitle = input.MetaTitle
	news.MetaDesc = input.MetaDesc

	if input.IsPublished && news.PublishedAt == nil {
		now := time.Now()
		news.PublishedAt = &now
	}

	// Update with associations
	if err := h.newsRepo.UpdateWithAssociations(news, input.TagIDs, input.CategoryIDs); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to update news", err)
		return
	}

	// Load updated news with relationships
	updatedNews, err := h.newsRepo.GetByID(news.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to load updated news", err)
		return
	}

	helpers.SuccessResponse(c, "News updated successfully", updatedNews.ToResponse())
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

	helpers.SuccessResponse(c, "News deleted successfully", nil)
}

// GetFeaturedNews lấy tin tức nổi bật
func (h *NewsHandler) GetFeaturedNews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	news, err := h.newsRepo.GetFeaturedNews(limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get featured news", err)
		return
	}

	var responses []model.NewsResponse
	for _, article := range news {
		responses = append(responses, article.ToResponse())
	}

	helpers.SuccessResponse(c, "Featured news retrieved successfully", responses)
}

// GetLatestNews lấy tin tức mới nhất
func (h *NewsHandler) GetLatestNews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	news, err := h.newsRepo.GetLatestNews(limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get latest news", err)
		return
	}

	var responses []model.NewsResponse
	for _, article := range news {
		responses = append(responses, article.ToResponse())
	}

	helpers.SuccessResponse(c, "Latest news retrieved successfully", responses)
}

// GetPopularNews lấy tin tức phổ biến
func (h *NewsHandler) GetPopularNews(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	news, err := h.newsRepo.GetPopularNews(limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to get popular news", err)
		return
	}

	var responses []model.NewsResponse
	for _, article := range news {
		responses = append(responses, article.ToResponse())
	}

	helpers.SuccessResponse(c, "Popular news retrieved successfully", responses)
}

// SearchNews tìm kiếm tin tức
func (h *NewsHandler) SearchNews(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Search keyword is required", nil)
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	news, total, err := h.newsRepo.SearchNews(keyword, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Failed to search news", err)
		return
	}

	var responses []model.NewsResponse
	for _, article := range news {
		responses = append(responses, article.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	response := map[string]interface{}{
		"news":        responses,
		"keyword":     keyword,
		"total":       total,
		"page":        page,
		"limit":       limit,
		"total_pages": totalPages,
		"has_next":    page < int(totalPages),
		"has_prev":    page > 1,
	}

	helpers.SuccessResponse(c, "News search completed successfully", response)
}