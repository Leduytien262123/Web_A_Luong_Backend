package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type ArticleHandler struct {
	articleRepo  *repo.ArticleRepo
	categoryRepo *repo.CategoryRepo
	tagRepo      *repo.TagRepo
}

func normalizeArticleStatus(status *string) (string, error) {
	if status == nil || strings.TrimSpace(*status) == "" {
		return "draft", nil
	}

	val := strings.ToLower(strings.TrimSpace(*status))
	switch val {
	case "draft":
		return "draft", nil
	case "post":
		return "post", nil
	default:
		return "", errors.New("status phải là 'draft' hoặc 'post'")
	}
}

func NewArticleHandler() *ArticleHandler {
	return &ArticleHandler{
		articleRepo:  repo.NewArticleRepo(),
		categoryRepo: repo.NewCategoryRepo(),
		tagRepo:      repo.NewTagRepo(),
	}
}

// attachTagNamesToResponses thêm trường TagNames (tên của tags) vào các ArticleResponse
func (h *ArticleHandler) attachTagNamesToResponses(responses []model.ArticleResponse) {
	// collect unique tag ids
	idSet := make(map[string]struct{})
	var ids []uuid.UUID
	for _, r := range responses {
		for _, id := range r.TagIDs {
			if _, ok := idSet[id.String()]; !ok {
				idSet[id.String()] = struct{}{}
				ids = append(ids, id)
			}
		}
	}
	if len(ids) == 0 {
		return
	}

	tags, err := h.tagRepo.GetByIDs(ids)
	if err != nil {
		return
	}

	nameMap := make(map[string]string)
	for _, t := range tags {
		nameMap[t.ID.String()] = t.Name
	}

	for i := range responses {
		var names []string
		for _, id := range responses[i].TagIDs {
			if n, ok := nameMap[id.String()]; ok {
				names = append(names, n)
			}
		}
		responses[i].TagNames = names
	}
}

// attachTagNamesToResponse thêm TagNames cho một ArticleResponse đơn lẻ
func (h *ArticleHandler) attachTagNamesToResponse(resp *model.ArticleResponse) {
	if resp == nil || len(resp.TagIDs) == 0 {
		return
	}
	tags, err := h.tagRepo.GetByIDs(resp.TagIDs)
	if err != nil {
		return
	}
	var names []string
	for _, t := range tags {
		names = append(names, t.Name)
	}
	resp.TagNames = names
}

// CreateArticle tạo bài viết mới
func (h *ArticleHandler) CreateArticle(c *gin.Context) {
	var input model.ArticleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Lấy author ID từ context (user đang đăng nhập)
	authorIDInterface, exists := c.Get("userID")
	if !exists {
		helpers.ErrorResponse(c, http.StatusUnauthorized, "Không xác định được người dùng", errors.New("userID not found in context"))
		return
	}
	authorID := authorIDInterface.(uuid.UUID)

	// Chuẩn hóa slug
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))

	// Kiểm tra slug
	exists, err := h.articleRepo.CheckSlugExists(input.Slug, uuid.Nil)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug đã tồn tại", errors.New("bài viết với slug này đã tồn tại"))
		return
	}

	// Kiểm tra danh mục nếu có
	if input.CategoryID != nil {
		_, err := h.categoryRepo.GetByID(*input.CategoryID)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh mục không hợp lệ", errors.New("không tìm thấy danh mục"))
			return
		}
	}

	// Validate và set status
	status, err := normalizeArticleStatus(input.Status)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Status không hợp lệ", err)
		return
	}

	isActive := true
	if input.IsActive != nil {
		isActive = *input.IsActive
	}

	isHot := false
	if input.IsHot != nil {
		isHot = *input.IsHot
	}

	article := model.Article{
		Title:       input.Title,
		Description: input.Description,
		Slug:        input.Slug,
		CategoryID:  input.CategoryID,
		IsActive:    isActive,
		IsHot:       isHot,
		Status:      status,
		PublishedAt: input.PublishedAt,
		AuthorID:    authorID,
	}

	// Sử dụng method SetTagIDs của Article model
	if err := article.SetTagIDs(input.TagIDs); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Danh sách tag không hợp lệ", err)
		return
	}

	// Parse metadata
	if input.Metadata != nil {
		metadataJSON, _ := json.Marshal(input.Metadata)
		article.Metadata = datatypes.JSON(metadataJSON)
	}

	// Parse content
	if input.Content != nil {
		contentJSON, _ := json.Marshal(input.Content)
		article.Content = datatypes.JSON(contentJSON)
	}

	if err := h.articleRepo.Create(&article); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo bài viết", err)
		return
	}

	// Load lại với quan hệ
	createdArticle, err := h.articleRepo.GetByID(article.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tải bài viết đã tạo", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "Tạo bài viết thành công",
		Data:    createdArticle.ToResponse(),
	})
}

// GetArticles lấy danh sách bài viết
func (h *ArticleHandler) GetArticles(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	categoryIDStr := c.Query("category_id")
	tagIDStr := c.Query("tag_id")
	publishedStr := c.Query("published")
	search := c.Query("search")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Build filter
	filter := repo.ArticleFilter{
		Search: strings.TrimSpace(search),
	}

	// Parse category_id
	if categoryIDStr != "" {
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", err)
			return
		}
		filter.CategoryID = &categoryID
	}

	// Parse tag_id
	if tagIDStr != "" {
		tagID, err := uuid.Parse(tagIDStr)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", err)
			return
		}
		filter.TagID = &tagID
	}

	// Parse published status
	if publishedStr == "true" {
		t := true
		filter.Published = &t
	} else if publishedStr == "false" {
		f := false
		filter.Published = &f
	}

	// Search with filters
	articles, total, err := h.articleRepo.SearchWithFilters(filter, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách bài viết", err)
		return
	}

	response := make([]model.ArticleResponse, 0, len(articles))
	for _, article := range articles {
		response = append(response, article.ToResponse())
	}

	// Attach tag names for user-facing responses
	h.attachTagNamesToResponses(response)

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách bài viết thành công",
		Data: map[string]interface{}{
			"articles":    response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// GetPublicArticles lấy danh sách bài viết đã đăng (public)
func (h *ArticleHandler) GetPublicArticles(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	articles, total, err := h.articleRepo.GetPublished(page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách bài viết", err)
		return
	}

	var response []model.ArticleResponse
	for _, article := range articles {
		response = append(response, article.ToResponse())
	}
	// Attach tag names for user-facing responses
	h.attachTagNamesToResponses(response)
	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách bài viết thành công",
		Data: map[string]interface{}{
			"articles":    response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
		},
	})
}

// GetArticleByID lấy bài viết theo ID
func (h *ArticleHandler) GetArticleByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID bài viết không hợp lệ", err)
		return
	}

	article, err := h.articleRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy bài viết", err)
		return
	}

	// Tăng view count
	_ = h.articleRepo.IncrementViewCount(id)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin bài viết thành công",
		Data:    article.ToResponse(),
	})
}

// GetArticleBySlug lấy bài viết theo slug
func (h *ArticleHandler) GetArticleBySlug(c *gin.Context) {
	slug := c.Param("slug")

	article, err := h.articleRepo.GetBySlug(slug)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy bài viết", err)
		return
	}

	// Tăng view count
	_ = h.articleRepo.IncrementViewCount(article.ID)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin bài viết thành công",
		Data:    article.ToResponse(),
	})
}

// GetArticleBySlugPublic lấy bài viết public theo slug
func (h *ArticleHandler) GetArticleBySlugPublic(c *gin.Context) {
	slug := c.Param("slug")

	article, err := h.articleRepo.GetPublishedBySlug(slug)
	if err != nil {
		if err.Error() == "article not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy bài viết", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy bài viết", err)
		return
	}

	_ = h.articleRepo.IncrementViewCount(article.ID)

	resp := article.ToResponse()
	h.attachTagNamesToResponse(&resp)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin bài viết thành công",
		Data:    resp,
	})
}

// GetArticlesByCategorySlug lấy bài viết public theo slug danh mục
func (h *ArticleHandler) GetArticlesByCategorySlug(c *gin.Context) {
	slug := c.Param("slug")
	if strings.TrimSpace(slug) == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Slug không hợp lệ", errors.New("slug không được để trống"))
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	category, err := h.categoryRepo.GetActiveBySlugWithArticles(slug)
	if err != nil {
		if err.Error() == "category not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh mục", err)
		return
	}

	articles, total, err := h.articleRepo.GetPublishedByCategorySlug(slug, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy bài viết theo danh mục", err)
		return
	}

	var responses []model.ArticleResponse
	for _, article := range articles {
		responses = append(responses, article.ToResponse())
	}
	// Attach tag names for user-facing responses
	h.attachTagNamesToResponses(responses)
	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy bài viết theo danh mục thành công",
		Data: map[string]interface{}{
			"category": category.ToResponse(),
			"articles": responses,
			"pagination": map[string]interface{}{
				"page":        page,
				"limit":       limit,
				"total":       total,
				"total_pages": totalPages,
			},
		},
	})
}

// GetFeaturedArticles lấy bài viết nổi bật
func (h *ArticleHandler) GetFeaturedArticles(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "5")
	limit, _ := strconv.Atoi(limitStr)

	if limit < 1 || limit > 20 {
		limit = 5
	}

	articles, err := h.articleRepo.GetFeatured(limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy bài viết nổi bật", err)
		return
	}

	var response []model.ArticleResponse
	for _, article := range articles {
		response = append(response, article.ToResponse())
	}
	// Attach tag names for user-facing responses
	h.attachTagNamesToResponses(response)
	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy bài viết nổi bật thành công",
		Data:    response,
	})
}

// SearchArticles tìm kiếm bài viết
func (h *ArticleHandler) SearchArticles(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Từ khóa tìm kiếm không được để trống", nil)
		return
	}

	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")

	page, _ := strconv.Atoi(pageStr)
	limit, _ := strconv.Atoi(limitStr)

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	articles, total, err := h.articleRepo.Search(keyword, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tìm kiếm bài viết", err)
		return
	}

	var response []model.ArticleResponse
	for _, article := range articles {
		response = append(response, article.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Tìm kiếm bài viết thành công",
		Data: map[string]interface{}{
			"articles":    response,
			"total":       total,
			"page":        page,
			"limit":       limit,
			"total_pages": totalPages,
			"keyword":     keyword,
		},
	})
}

// UpdateArticle cập nhật bài viết
func (h *ArticleHandler) UpdateArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID bài viết không hợp lệ", err)
		return
	}

	var input model.ArticleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	article, err := h.articleRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy bài viết", err)
		return
	}

	// Chuẩn hóa slug
	input.Slug = strings.ToLower(strings.TrimSpace(input.Slug))

	// Kiểm tra slug
	exists, err := h.articleRepo.CheckSlugExists(input.Slug, id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug đã tồn tại", errors.New("bài viết khác với slug này đã tồn tại"))
		return
	}

	// Kiểm tra danh mục nếu có
	if input.CategoryID != nil {
		_, err := h.categoryRepo.GetByID(*input.CategoryID)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh mục không hợp lệ", errors.New("không tìm thấy danh mục"))
			return
		}
	}

	// Validate status nếu có
	if input.Status != nil && *input.Status != "" {
		if _, err := normalizeArticleStatus(input.Status); err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Status không hợp lệ", err)
			return
		}
	}

	article.Title = input.Title
	article.Description = input.Description
	article.Slug = input.Slug
	article.CategoryID = input.CategoryID
	article.Category = nil

	if input.IsActive != nil {
		article.IsActive = *input.IsActive
	}

	if input.IsHot != nil {
		article.IsHot = *input.IsHot
	}

	// Sử dụng method SetTagIDs của Article model
	if input.TagIDs != nil {
		if err := article.SetTagIDs(input.TagIDs); err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh sách tag không hợp lệ", err)
			return
		}
	}

	if input.Status != nil && *input.Status != "" {
		article.Status, _ = normalizeArticleStatus(input.Status)
	}
	if input.PublishedAt != nil {
		article.PublishedAt = input.PublishedAt
	}

	if input.Metadata != nil {
		metadataJSON, _ := json.Marshal(input.Metadata)
		article.Metadata = datatypes.JSON(metadataJSON)
	}

	if input.Content != nil {
		contentJSON, _ := json.Marshal(input.Content)
		article.Content = datatypes.JSON(contentJSON)
	}

	if err := h.articleRepo.Update(article); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật bài viết", err)
		return
	}

	updatedArticle, err := h.articleRepo.GetByID(article.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tải bài viết đã cập nhật", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật bài viết thành công",
		Data:    updatedArticle.ToResponse(),
	})
}

// DeleteArticle xóa bài viết
func (h *ArticleHandler) DeleteArticle(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID bài viết không hợp lệ", err)
		return
	}

	_, err = h.articleRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy bài viết", err)
		return
	}

	if err := h.articleRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa bài viết", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Xóa bài viết thành công",
		Data:    nil,
	})
}
