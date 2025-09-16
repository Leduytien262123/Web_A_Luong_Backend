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

type NewsCategoryHandler struct {
	newsCategoryRepo *repo.NewsCategoryRepo
}

func NewNewsCategoryHandler() *NewsCategoryHandler {
	return &NewsCategoryHandler{
		newsCategoryRepo: repo.NewNewsCategoryRepo(),
	}
}

// CreateNewsCategory tạo danh mục tin tức mới
func (h *NewsCategoryHandler) CreateNewsCategory(c *gin.Context) {
	var input model.NewsCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra slug đã tồn tại chưa
	if exists, err := h.newsCategoryRepo.CheckSlugExists(input.Slug, uuid.Nil); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	} else if exists {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Slug đã tồn tại", nil)
		return
	}

	// Kiểm tra parent category nếu có
	if input.ParentID != nil {
		if _, err := h.newsCategoryRepo.GetByID(*input.ParentID); err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh mục cha không tồn tại", err)
			return
		}
	}

	category := &model.NewsCategory{
		Name:        input.Name,
		Slug:        input.Slug,
		Description: input.Description,
		ImageURL:    input.ImageURL,
		ParentID:    input.ParentID,
		SortOrder:   input.SortOrder,
		IsActive:    input.IsActive,
		MetaTitle:   input.MetaTitle,
		MetaDesc:    input.MetaDesc,
	}

	if err := h.newsCategoryRepo.Create(category); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo danh mục", err)
		return
	}

	helpers.SuccessResponse(c, "Tạo danh mục tin tức thành công", category.ToResponse())
}

// GetNewsCategories lấy danh sách danh mục tin tức
func (h *NewsCategoryHandler) GetNewsCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	activeOnly := c.Query("active_only") == "true"

	categories, total, err := h.newsCategoryRepo.GetAll(page, limit, activeOnly)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách danh mục", err)
		return
	}

	var responses []model.NewsCategoryResponse
	for _, category := range categories {
		responses = append(responses, category.ToResponse())
	}

	response := map[string]interface{}{
		"categories": responses,
		"pagination": map[string]interface{}{
			"page":  page,
			"limit": limit,
			"total": total,
		},
	}

	helpers.SuccessResponse(c, "Lấy danh sách danh mục thành công", response)
}

// GetNewsCategoryByID lấy danh mục tin tức theo ID
func (h *NewsCategoryHandler) GetNewsCategoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", err)
		return
	}

	category, err := h.newsCategoryRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thông tin danh mục thành công", category.ToResponse())
}

// GetNewsCategoryBySlug lấy danh mục tin tức theo slug
func (h *NewsCategoryHandler) GetNewsCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")

	category, err := h.newsCategoryRepo.GetBySlug(slug)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy thông tin danh mục thành công", category.ToResponse())
}

// UpdateNewsCategory cập nhật danh mục tin tức
func (h *NewsCategoryHandler) UpdateNewsCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", err)
		return
	}

	category, err := h.newsCategoryRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
		return
	}

	var input model.NewsCategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra slug đã tồn tại chưa (trừ chính nó)
	if input.Slug != category.Slug {
		if exists, err := h.newsCategoryRepo.CheckSlugExists(input.Slug, id); err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		} else if exists {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Slug đã tồn tại", nil)
			return
		}
	}

	// Kiểm tra parent category nếu có
	if input.ParentID != nil && *input.ParentID != id {
		if _, err := h.newsCategoryRepo.GetByID(*input.ParentID); err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh mục cha không tồn tại", err)
			return
		}
	}

	// Cập nhật thông tin
	category.Name = input.Name
	category.Slug = input.Slug
	category.Description = input.Description
	category.ImageURL = input.ImageURL
	category.ParentID = input.ParentID
	category.SortOrder = input.SortOrder
	category.IsActive = input.IsActive
	category.MetaTitle = input.MetaTitle
	category.MetaDesc = input.MetaDesc

	if err := h.newsCategoryRepo.Update(category); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật danh mục", err)
		return
	}

	helpers.SuccessResponse(c, "Cập nhật danh mục thành công", category.ToResponse())
}

// DeleteNewsCategory xóa danh mục tin tức
func (h *NewsCategoryHandler) DeleteNewsCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", err)
		return
	}

	if err := h.newsCategoryRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa danh mục", err)
		return
	}

	helpers.SuccessResponse(c, "Xóa danh mục thành công", nil)
}

// GetNewsCategoryTree lấy cấu trúc cây danh mục
func (h *NewsCategoryHandler) GetNewsCategoryTree(c *gin.Context) {
	categories, err := h.newsCategoryRepo.GetTreeStructure()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy cấu trúc danh mục", err)
		return
	}

	var responses []model.NewsCategoryResponse
	for _, category := range categories {
		responses = append(responses, category.ToResponse())
	}

	helpers.SuccessResponse(c, "Lấy cấu trúc danh mục thành công", responses)
}