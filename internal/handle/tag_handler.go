package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TagHandler struct {
	tagRepo *repo.TagRepo
}

func NewTagHandler() *TagHandler {
	return &TagHandler{
		tagRepo: repo.NewTagRepo(),
	}
}

// CreateTag tạo tag mới
func (h *TagHandler) CreateTag(c *gin.Context) {
	var input model.TagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra slug đã tồn tại chưa
	if exists, err := h.tagRepo.CheckSlugExists(input.Slug, uuid.Nil); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	} else if exists {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Slug đã tồn tại", nil)
		return
	}

	tag := &model.Tag{
		Name:        input.Name,
		Slug:        input.Slug,
		Color:       input.Color,
		Description: input.Description,
		IsActive:    input.IsActive,
	}

	// Set default color if not provided
	if tag.Color == "" {
		tag.Color = "#007bff"
	}

	if err := h.tagRepo.Create(tag); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo tag", err)
		return
	}

	helpers.SuccessResponse(c, "Tạo tag thành công", tag.ToResponse())
}

// GetTags lấy danh sách tags
func (h *TagHandler) GetTags(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	activeOnly := c.Query("active_only") == "true"

	tags, total, err := h.tagRepo.GetAll(page, limit, activeOnly)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách tags", err)
		return
	}

	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse())
	}

	response := map[string]interface{}{
		"tags": responses,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	helpers.SuccessResponse(c, "Lấy danh sách tags thành công", response)
}

// GetTagByID lấy tag theo ID
func (h *TagHandler) GetTagByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", err)
		return
	}

	tag, err := h.tagRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thông tin tag thành công", tag.ToResponse())
}

// GetTagBySlug lấy tag theo slug
func (h *TagHandler) GetTagBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tag, err := h.tagRepo.GetBySlug(slug)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thông tin tag thành công", tag.ToResponse())
}

// UpdateTag cập nhật tag
func (h *TagHandler) UpdateTag(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", err)
		return
	}

	tag, err := h.tagRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy tag", err)
		return
	}

	var input model.TagInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra slug đã tồn tại chưa (trừ chính nó)
	if input.Slug != tag.Slug {
		if exists, err := h.tagRepo.CheckSlugExists(input.Slug, id); err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		} else if exists {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Slug đã tồn tại", nil)
			return
		}
	}

	// Cập nhật thông tin
	tag.Name = input.Name
	tag.Slug = input.Slug
	tag.Description = input.Description
	tag.IsActive = input.IsActive
	if input.Color != "" {
		tag.Color = input.Color
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
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID tag không hợp lệ", err)
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
		helpers.ErrorResponse(c, http.StatusBadRequest, "Từ khóa tìm kiếm không được để trống", nil)
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	tags, err := h.tagRepo.SearchTags(keyword, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tìm kiếm tags", err)
		return
	}

	var responses []model.TagResponse
	for _, tag := range tags {
		responses = append(responses, tag.ToResponse())
	}

	helpers.SuccessResponse(c, "Tìm kiếm tags thành công", responses)
}

