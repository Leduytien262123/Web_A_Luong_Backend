package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProductHandler struct {
	productRepo  *repo.ProductRepo
	categoryRepo *repo.CategoryRepo
}

func NewProductHandler() *ProductHandler {
	return &ProductHandler{
		productRepo:  repo.NewProductRepo(),
		categoryRepo: repo.NewCategoryRepo(),
	}
}

// CreateProduct tạo sản phẩm mới
func (h *ProductHandler) CreateProduct(c *gin.Context) {
	var input model.ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Chuẩn hóa SKU
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))

	// Kiểm tra xem SKU đã tồn tại chưa
	exists, err := h.productRepo.CheckSKUExists(input.SKU, uuid.Nil)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "SKU đã tồn tại", errors.New("sản phẩm với SKU này đã tồn tại"))
		return
	}

	// Kiểm tra xem danh mục có tồn tại không (nếu được cung cấp)
	if input.CategoryID != nil {
		_, err := h.categoryRepo.GetByID(*input.CategoryID)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh mục không hợp lệ", errors.New("không tìm thấy danh mục"))
			return
		}
	}

	product := model.Product{
		Name:        input.Name,
		Description: input.Description,
		Price:       input.Price,
		SKU:         input.SKU,
		Stock:       input.Stock,
		CategoryID:  input.CategoryID,
		IsActive:    true,
	}

	if err := h.productRepo.Create(&product); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo sản phẩm", err)
		return
	}

	// Tải sản phẩm với danh mục để phản hồi
	createdProduct, err := h.productRepo.GetByID(product.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tải sản phẩm đã tạo", err)
		return
	}

	c.JSON(http.StatusCreated, helpers.Response{
		Success: true,
		Message: "Tạo sản phẩm thành công",
		Data:    createdProduct.ToResponse(),
	})
}

// GetProducts lấy tất cả sản phẩm với phân trang
func (h *ProductHandler) GetProducts(c *gin.Context) {
	// Phân tích các tham số phân trang
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	categoryIDStr := c.Query("category_id")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	var products []model.Product
	var total int64

	if categoryIDStr != "" {
		// Lấy sản phẩm theo danh mục
		categoryID, err := uuid.Parse(categoryIDStr)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "ID danh mục không hợp lệ", errors.New("ID danh mục phải là UUID hợp lệ"))
			return
		}

		products, total, err = h.productRepo.GetByCategoryID(categoryID, page, limit)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách sản phẩm", err)
			return
		}
	} else {
		// Lấy tất cả sản phẩm
		products, total, err = h.productRepo.GetAll(page, limit)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách sản phẩm", err)
			return
		}
	}

	var response []model.ProductResponse
	for _, product := range products {
		response = append(response, product.ToResponse())
	}

	// Tính toán thông tin phân trang
	totalPages := (total + int64(limit) - 1) / int64(limit)

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách sản phẩm thành công",
		Data: map[string]interface{}{
			"products":     response,
			"total":        total,
			"page":         page,
			"limit":        limit,
			"total_pages":  totalPages,
			"has_next":     page < int(totalPages),
			"has_prev":     page > 1,
		},
	})
}

// GetProductByID lấy sản phẩm theo ID
func (h *ProductHandler) GetProductByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID sản phẩm không hợp lệ", errors.New("ID sản phẩm phải là UUID hợp lệ"))
		return
	}

	product, err := h.productRepo.GetByID(id)
	if err != nil {
		if err.Error() == "product not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy sản phẩm", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin sản phẩm thành công",
		Data:    product.ToResponse(),
	})
}

// GetProductBySKU lấy sản phẩm theo SKU
func (h *ProductHandler) GetProductBySKU(c *gin.Context) {
	sku := c.Param("sku")
	if sku == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "SKU không hợp lệ", errors.New("SKU không được để trống"))
		return
	}

	product, err := h.productRepo.GetBySKU(sku)
	if err != nil {
		if err.Error() == "product not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy sản phẩm", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy thông tin sản phẩm thành công",
		Data:    product.ToResponse(),
	})
}

// UpdateProduct cập nhật sản phẩm
func (h *ProductHandler) UpdateProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID sản phẩm không hợp lệ", errors.New("ID sản phẩm phải là UUID hợp lệ"))
		return
	}

	var input model.ProductInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Lấy sản phẩm hiện tại
	product, err := h.productRepo.GetByID(id)
	if err != nil {
		if err.Error() == "product not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy sản phẩm", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	// Chuẩn hóa SKU
	input.SKU = strings.ToUpper(strings.TrimSpace(input.SKU))

	// Kiểm tra xem SKU đã tồn tại chưa (loại trừ sản phẩm hiện tại)
	exists, err := h.productRepo.CheckSKUExists(input.SKU, id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "SKU đã tồn tại", errors.New("sản phẩm khác với SKU này đã tồn tại"))
		return
	}

	// Kiểm tra xem danh mục có tồn tại không (nếu được cung cấp)
	if input.CategoryID != nil {
		_, err := h.categoryRepo.GetByID(*input.CategoryID)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusBadRequest, "Danh mục không hợp lệ", errors.New("không tìm thấy danh mục"))
			return
		}
	}

	// Cập nhật sản phẩm
	product.Name = input.Name
	product.Description = input.Description
	product.Price = input.Price
	product.SKU = input.SKU
	product.Stock = input.Stock
	product.CategoryID = input.CategoryID

	if err := h.productRepo.Update(product); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật sản phẩm", err)
		return
	}

	// Tải sản phẩm đã cập nhật với danh mục
	updatedProduct, err := h.productRepo.GetByID(product.ID)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tải sản phẩm đã cập nhật", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật sản phẩm thành công",
		Data:    updatedProduct.ToResponse(),
	})
}

// DeleteProduct xóa sản phẩm
func (h *ProductHandler) DeleteProduct(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID sản phẩm không hợp lệ", errors.New("ID sản phẩm phải là UUID hợp lệ"))
		return
	}

	// Kiểm tra xem sản phẩm có tồn tại không
	_, err = h.productRepo.GetByID(id)
	if err != nil {
		if err.Error() == "product not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy sản phẩm", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	if err := h.productRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa sản phẩm", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Xóa sản phẩm thành công",
		Data:    nil,
	})
}

// UpdateProductStock cập nhật số lượng tồn kho sản phẩm
func (h *ProductHandler) UpdateProductStock(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID sản phẩm không hợp lệ", errors.New("ID sản phẩm phải là UUID hợp lệ"))
		return
	}

	var input struct {
		Stock int `json:"stock" binding:"required,gte=0"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Kiểm tra xem sản phẩm có tồn tại không
	product, err := h.productRepo.GetByID(id)
	if err != nil {
		if err.Error() == "product not found" {
			helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy sản phẩm", err)
			return
		}
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}

	if err := h.productRepo.UpdateStock(id, input.Stock); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật số lượng tồn kho", err)
		return
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Cập nhật số lượng tồn kho thành công",
		Data: map[string]interface{}{
			"product_id": product.ID,
			"old_stock":  product.Stock,
			"new_stock":  input.Stock,
		},
	})
}
