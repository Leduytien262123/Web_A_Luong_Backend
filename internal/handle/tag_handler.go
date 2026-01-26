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

type TagHandler struct {
	tagRepo     *repo.TagRepo
	articleRepo *repo.ArticleRepo
}

func NewTagHandler() *TagHandler {
	return &TagHandler{
		tagRepo:     repo.NewTagRepo(),
		articleRepo: repo.NewArticleRepo(),
	}
}

// CreateTag tạo tag mới
func (h *TagHandler) CreateTag(c *gin.Context) {
	var input model.TagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Chuẩn hóa slug
	input.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input.Slug), " ", "-"))

	// Kiểm tra xem slug đã tồn tại chưa
	exists, err := h.tagRepo.CheckSlugExists(input.Slug, uuid.Nil)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug đã tồn tại", errors.New("tag với slug này đã tồn tại"))
		return
	}

	// Tạo model tag từ input
	tag := model.Tag{
		Name:         input.Name,
		Slug:         input.Slug,
		Description:  input.Description,
		DisplayOrder: 0,
		IsActive:     true,
		UsageCount:   0,
	}

	if input.DisplayOrder != nil {
		tag.DisplayOrder = *input.DisplayOrder
	}
	if input.IsActive != nil {
		tag.IsActive = *input.IsActive
	}

	// Marshal metadata và content sang JSON
	if input.Metadata != nil {
		metadataJSON, _ := json.Marshal(input.Metadata)
		tag.Metadata = datatypes.JSON(metadataJSON)
	}
	if input.Content != nil {
		contentJSON, _ := json.Marshal(input.Content)
		tag.Content = datatypes.JSON(contentJSON)
	}

	if err := h.tagRepo.Create(&tag); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo tag", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "Tạo tag thành công",
		Data:    tag.ToResponse(),
	})
}

// GetTags lấy danh sách tags với phân trang
func (h *TagHandler) GetTags(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	activeOnly := c.Query("active_only") == "true"
	search := c.Query("search")

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 10
	}

	var tags []model.Tag
	var total int64
	var err error

	if search != "" {
		// Tìm kiếm tags
		tags, err = h.tagRepo.SearchTags(search, limit)
		total = int64(len(tags))
	} else {
		// Lấy tất cả tags với phân trang
		tags, total, err = h.tagRepo.GetAll(page, limit, activeOnly)
	}

	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách tags", err)
		return
	}

	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	response := map[string]interface{}{
		"tags": responses,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}

	helpers.SuccessResponse(c, "Lấy danh sách tags thành công", response)
}

// GetPublicTags trả về danh sách tags đang hoạt động (public)
func (h *TagHandler) GetPublicTags(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 50 {
		limit = 20
	}

	tags, total, err := h.tagRepo.GetAll(page, limit, true)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách tags", err)
		return
	}

	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	helpers.SuccessResponse(c, "Lấy danh sách tags thành công", map[string]interface{}{
		"tags": responses,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}

// GetTagByID lấy tag theo ID
func (h *TagHandler) GetTagByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", errors.New("ID tag phải là UUID hợp lệ"))
		return
	}

	tag, err := h.tagRepo.GetByID(id)
	if err != nil {
		if err.Error() == "tag not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thông tin tag thành công", tag.ToResponse())
}

// GetTagBySlug lấy tag theo slug
func (h *TagHandler) GetTagBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Slug không hợp lệ", errors.New("slug không được để trống"))
		return
	}

	tag, err := h.tagRepo.GetBySlug(slug)
	if err != nil {
		if err.Error() == "tag not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thông tin tag thành công", tag.ToResponse())
}

// UpdateTag cập nhật tag
func (h *TagHandler) UpdateTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", errors.New("ID tag phải là UUID hợp lệ"))
		return
	}

	var input model.TagUpdateInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Lấy tag hiện tại
	tag, err := h.tagRepo.GetByID(id)
	if err != nil {
		if err.Error() == "tag not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	// Chuẩn hóa slug
	input.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input.Slug), " ", "-"))

	// Kiểm tra xem slug đã tồn tại chưa (loại trừ tag hiện tại)
	if input.Slug != tag.Slug {
		exists, err := h.tagRepo.CheckSlugExists(input.Slug, id)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		}
		if exists {
			helpers.ErrorResponse(c, http.StatusConflict, "Slug đã tồn tại", errors.New("tag khác với slug này đã tồn tại"))
			return
		}
	}

	// Cập nhật tag
	tag.Name = input.Name
	tag.Slug = input.Slug
	tag.Description = input.Description

	if input.DisplayOrder != nil {
		tag.DisplayOrder = *input.DisplayOrder
	}
	if input.IsActive != nil {
		tag.IsActive = *input.IsActive
	}

	// Marshal metadata và content sang JSON
	if input.Metadata != nil {
		metadataJSON, _ := json.Marshal(input.Metadata)
		tag.Metadata = datatypes.JSON(metadataJSON)
	}
	if input.Content != nil {
		contentJSON, _ := json.Marshal(input.Content)
		tag.Content = datatypes.JSON(contentJSON)
	}

	if err := h.tagRepo.Update(tag); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật tag", err)
		return
	}

	helpers.SuccessResponse(c, "Cập nhật tag thành công", tag.ToResponse())
}

// DeleteTag xóa tag
func (h *TagHandler) DeleteTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", errors.New("ID tag phải là UUID hợp lệ"))
		return
	}

	// Kiểm tra xem tag có tồn tại không
	_, err = h.tagRepo.GetByID(id)
	if err != nil {
		if err.Error() == "tag not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	if err := h.tagRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa tag", err)
		return
	}

	helpers.SuccessResponse(c, "Xóa tag thành công", nil)
}

// GetPopularTags lấy tags phổ biến
func (h *TagHandler) GetPopularTags(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	tags, err := h.tagRepo.GetPopularTags(limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy tags phổ biến", err)
		return
	}

	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse())
	}

	helpers.SuccessResponse(c, "Lấy tags phổ biến thành công", responses)
}

// SearchTags tìm kiếm tags
func (h *TagHandler) SearchTags(c *gin.Context) {
	keyword := c.Query("q")
	if keyword == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Từ khóa tìm kiếm là bắt buộc", nil)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit < 1 || limit > 50 {
		limit = 10
	}

	tags, err := h.tagRepo.SearchTags(keyword, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tìm kiếm tags", err)
		return
	}

	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse())
	}

	response := map[string]interface{}{
		"tags":    responses,
		"keyword": keyword,
		"total":   len(responses),
	}

	helpers.SuccessResponse(c, "Tìm kiếm tags thành công", response)
}

// GetArticlesByTagSlug trả về bài viết đã đăng theo slug tag (public)
func (h *TagHandler) GetArticlesByTagSlug(c *gin.Context) {
	slug := c.Param("slug")
	if strings.TrimSpace(slug) == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Slug không hợp lệ", errors.New("slug không được để trống"))
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 10
	}

	tag, err := h.tagRepo.GetBySlug(slug)
	if err != nil {
		if err.Error() == "tag not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy tag", err)
		return
	}

	articles, total, err := h.articleRepo.GetPublishedByTagID(tag.ID, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy bài viết theo tag", err)
		return
	}

	var responses []model.ArticleResponse
	for _, article := range articles {
		responses = append(responses, article.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	helpers.SuccessResponse(c, "Lấy bài viết theo tag thành công", map[string]interface{}{
		"tag":      tag.ToResponse(),
		"articles": responses,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	})
}
