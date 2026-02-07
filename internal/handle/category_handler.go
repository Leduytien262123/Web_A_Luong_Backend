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

type CategoryHandler struct {
	categoryRepo *repo.CategoryRepo
}

func NewCategoryHandler() *CategoryHandler {
	return &CategoryHandler{
		categoryRepo: repo.NewCategoryRepo(),
	}
}

// CreateCategory tạo danh mục mới
func (h *CategoryHandler) CreateCategory(c *gin.Context) {
	var input model.CategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Chuẩn hóa slug
	input.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input.Slug), " ", "-"))

	// Kiểm tra xem slug đã tồn tại chưa
	exists, err := h.categoryRepo.CheckSlugExists(input.Slug, uuid.Nil)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug đã tồn tại", errors.New("danh mục với slug này đã tồn tại"))
		return
	}

	// Đảm bảo metadata luôn có giá trị và meta_image luôn là mảng (không null)
	if input.Metadata == nil {
		input.Metadata = &model.CategoryMetadata{
			MetaTitle:       "",
			MetaDescription: "",
			MetaImage:       []model.MetaImageCategory{},
			MetaKeywords:    "",
		}
	} else {
		if input.Metadata.MetaImage == nil {
			input.Metadata.MetaImage = []model.MetaImageCategory{}
		}
	}
	metadataJSON, _ := json.Marshal(input.Metadata)

	category := model.Category{
		Name:         input.Name,
		Description:  input.Description,
		Slug:         input.Slug,
		DisplayOrder: 0,
		IsActive:     false,
		ShowOnMenu:   false,
		ShowOnHome:   false,
		ShowOnFooter: false,
		Metadata:     datatypes.JSON(metadataJSON),
	}

	// Handle parent category if provided
	if input.ParentCategory != nil && strings.TrimSpace(*input.ParentCategory) != "" {
		if parentID, err := uuid.Parse(strings.TrimSpace(*input.ParentCategory)); err == nil {
			category.ParentID = &parentID
		} else {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Parent category ID không hợp lệ", err)
			return
		}
	}

	if input.IsActive != nil {
		category.IsActive = *input.IsActive
	}
	if input.DisplayOrder != nil {
		category.DisplayOrder = *input.DisplayOrder
	}
	if input.ShowOnMenu != nil {
		category.ShowOnMenu = *input.ShowOnMenu
	}
	if input.ShowOnHome != nil {
		category.ShowOnHome = *input.ShowOnHome
	}
	if input.ShowOnFooter != nil {
		category.ShowOnFooter = *input.ShowOnFooter
	}
	if input.PositionMenu != nil {
		category.PositionMenu = *input.PositionMenu
	}
	if input.PositionFooter != nil {
		category.PositionFooter = *input.PositionFooter
	}
	if input.PositionHome != nil {
		category.PositionHome = *input.PositionHome
	}
	// Handle parent category update
	if input.ParentCategory != nil {
		if strings.TrimSpace(*input.ParentCategory) == "" {
			// clear parent
			category.ParentID = nil
		} else {
			if parentID, err := uuid.Parse(strings.TrimSpace(*input.ParentCategory)); err == nil {
				category.ParentID = &parentID
			} else {
				helpers.ErrorResponse(c, http.StatusBadRequest, "Parent category ID không hợp lệ", err)
				return
			}
		}
	}
	// Metadata đã được chuẩn hóa phía trên
	category.Metadata = datatypes.JSON(metadataJSON)

	if err := h.categoryRepo.Create(&category); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo danh mục", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "Tạo danh mục thành công",
		Data:    category.ToResponse(),
	})
}

// GetCategories lấy tất cả danh mục dạng tree
func (h *CategoryHandler) GetCategories(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limitStr := c.Query("limit")
	if limitStr == "" {
		limitStr = c.Query("length")
	}
	if limitStr == "" {
		limitStr = "100"
	}
	limit, _ := strconv.Atoi(limitStr)
	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 100
	}
	withArticles := c.Query("with_articles") == "true"
	treeView := c.DefaultQuery("tree", "true") == "true" // Mặc định là tree view

	var categories []model.Category
	var total int64
	var err error

	search := strings.TrimSpace(c.Query("search"))
	if search != "" {
		categories, total, err = h.categoryRepo.GetByNameWithPagination(search, withArticles, page, limit)
	} else {
		if withArticles {
			categories, total, err = h.categoryRepo.GetAllWithArticlesAndPagination(page, limit)
		} else {
			categories, total, err = h.categoryRepo.GetAllWithPagination(page, limit)
		}
	}

	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách danh mục", err)
		return
	}

	var response interface{}
	if treeView {
		// Trả về dạng cây
		response = model.BuildCategoryTree(categories)
	} else {
		// Trả về dạng list phẳng
		flatResponse := make([]model.CategoryResponse, 0, len(categories))
		for _, category := range categories {
			flatResponse = append(flatResponse, category.ToResponse())
		}
		response = flatResponse
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	result := map[string]interface{}{
		"categories": response,
		"pagination": map[string]interface{}{
			"page":        page,
			"limit":       limit,
			"total":       total,
			"total_pages": totalPages,
		},
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách danh mục thành công",
		Data:    result,
	})
}

// GetPublicCategories trả về danh mục hoạt động (public)
func (h *CategoryHandler) GetPublicCategories(c *gin.Context) {
	treeView := c.DefaultQuery("tree", "true") == "true"

	categories, err := h.categoryRepo.GetActive()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh mục", err)
		return
	}

	var response interface{}
	if treeView {
		response = model.BuildCategoryTree(categories)
	} else {
		var flatResponse []model.CategoryResponse
		for _, category := range categories {
			flatResponse = append(flatResponse, category.ToResponse())
		}
		response = flatResponse
	}

	helpers.SuccessResponse(c, "Lấy danh mục thành công", map[string]interface{}{
		"categories": response,
	})
}

// GetPublicHomeCategories trả về các danh mục có show_on_home = true kèm bài viết
// Query params: per_articles (số bài viết mỗi danh mục, mặc định 6)
func (h *CategoryHandler) GetPublicHomeCategories(c *gin.Context) {
	perArticles, _ := strconv.Atoi(c.DefaultQuery("per_articles", "6"))
	if perArticles < 0 {
		perArticles = 6
	}

	categories, err := h.categoryRepo.GetHomeCategoriesWithArticles(perArticles)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách danh mục cho trang chủ", err)
		return
	}

	var resp []model.CategoryResponse
	for _, cat := range categories {
		resp = append(resp, cat.ToResponse())
	}

	helpers.SuccessResponse(c, "Lấy danh mục trang chủ thành công", map[string]interface{}{
		"categories": resp,
	})
}

// GetCategoryTree lấy cây danh mục
func (h *CategoryHandler) GetCategoryTree(c *gin.Context) {
	withArticles := c.Query("with_articles") == "true"

	var categories []model.Category
	var err error

	if withArticles {
		categories, err = h.categoryRepo.GetAllWithArticles()
	} else {
		categories, err = h.categoryRepo.GetAll()
	}

	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy cây danh mục", err)
		return
	}

	// Xây dựng cấu trúc cây
	treeResponse := model.BuildCategoryTree(categories)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy cây danh mục thành công",
		Data:    treeResponse,
	})
}

// GetCategoryByID lấy danh mục theo ID
func (h *CategoryHandler) GetCategoryByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", errors.New("ID danh mục phải là UUID hợp lệ"))
		return
	}

	withArticles := c.Query("with_articles") == "true"

	var category *model.Category
	if withArticles {
		category, err = h.categoryRepo.GetByIDWithArticles(id)
	} else {
		category, err = h.categoryRepo.GetByID(id)
	}
	if err != nil {
		if err.Error() == "category not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin danh mục thành công",
		Data:    category.ToResponse(),
	})
}

// GetCategoryBySlug lấy danh mục theo slug
func (h *CategoryHandler) GetCategoryBySlug(c *gin.Context) {
	slug := c.Param("slug")
	if slug == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Slug không hợp lệ", errors.New("slug không được để trống"))
		return
	}

	withArticles := c.Query("with_articles") == "true"

	var err error
	var category *model.Category
	if withArticles {
		category, err = h.categoryRepo.GetBySlugWithArticles(slug)
	} else {
		category, err = h.categoryRepo.GetBySlug(slug)
	}
	if err != nil {
		if err.Error() == "category not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin danh mục thành công",
		Data:    category.ToResponse(),
	})
}

// UpdateCategory cập nhật danh mục
func (h *CategoryHandler) UpdateCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", errors.New("ID danh mục phải là UUID hợp lệ"))
		return
	}

	var input model.CategoryInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Lấy danh mục hiện tại
	category, err := h.categoryRepo.GetByID(id)
	if err != nil {
		if err.Error() == "category not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	// Chuẩn hóa slug
	input.Slug = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(input.Slug), " ", "-"))

	// Kiểm tra xem slug đã tồn tại chưa (loại trừ danh mục hiện tại)
	exists, err := h.categoryRepo.CheckSlugExists(input.Slug, id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "Slug đã tồn tại", errors.New("danh mục khác với slug này đã tồn tại"))
		return
	}

	// Cập nhật danh mục
	category.Name = input.Name
	category.Description = input.Description
	category.Slug = input.Slug
	if input.DisplayOrder != nil {
		category.DisplayOrder = *input.DisplayOrder
	}
	if input.IsActive != nil {
		category.IsActive = *input.IsActive
	}
	if input.ShowOnMenu != nil {
		category.ShowOnMenu = *input.ShowOnMenu
	}
	if input.ShowOnHome != nil {
		category.ShowOnHome = *input.ShowOnHome
	}
	if input.ShowOnFooter != nil {
		category.ShowOnFooter = *input.ShowOnFooter
	}
	if input.PositionMenu != nil {
		category.PositionMenu = *input.PositionMenu
	}
	if input.PositionFooter != nil {
		category.PositionFooter = *input.PositionFooter
	}
	if input.PositionHome != nil {
		category.PositionHome = *input.PositionHome
	}
	if input.Metadata != nil {
		if input.Metadata.MetaImage == nil {
			input.Metadata.MetaImage = []model.MetaImageCategory{}
		}
		metadataJSON, _ := json.Marshal(input.Metadata)
		category.Metadata = datatypes.JSON(metadataJSON)
	}

	// Handle parent category update
	if input.ParentCategory != nil {
		if strings.TrimSpace(*input.ParentCategory) == "" {
			// Clear parent
			category.ParentID = nil
		} else {
			if parentID, err := uuid.Parse(strings.TrimSpace(*input.ParentCategory)); err == nil {
				category.ParentID = &parentID
			} else {
				helpers.ErrorResponse(c, http.StatusBadRequest, "Parent category ID không hợp lệ", err)
				return
			}
		}
	}

	if err := h.categoryRepo.Update(category); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật danh mục", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật danh mục thành công",
		Data:    category.ToResponse(),
	})
}

// DeleteCategory xóa danh mục
func (h *CategoryHandler) DeleteCategory(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", errors.New("ID danh mục phải là UUID hợp lệ"))
		return
	}

	// Kiểm tra xem danh mục có tồn tại không
	_, err = h.categoryRepo.GetByID(id)
	if err != nil {
		if err.Error() == "category not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy danh mục", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	if err := h.categoryRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa danh mục", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Xóa danh mục thành công",
		Data:    nil,
	})
}
