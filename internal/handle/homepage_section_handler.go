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
	"gorm.io/datatypes"
)

type HomepageSectionHandler struct {
	sectionRepo *repo.HomepageSectionRepo
}

func NewHomepageSectionHandler() *HomepageSectionHandler {
	return &HomepageSectionHandler{
		sectionRepo: repo.NewHomepageSectionRepo(),
	}
}

// CreateSection tạo section mới
func (h *HomepageSectionHandler) CreateSection(c *gin.Context) {
	var input model.HomepageSectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Chuẩn hóa type_key
	input.TypeKey = strings.ToUpper(strings.TrimSpace(input.TypeKey))

	// Kiểm tra type_key đã tồn tại chưa
	exists, err := h.sectionRepo.CheckTypeKeyExists(input.TypeKey, uuid.Nil)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
		return
	}
	if exists {
		helpers.ErrorResponse(c, http.StatusConflict, "TypeKey đã tồn tại", errors.New("section với type_key này đã tồn tại"))
		return
	}

	// Set giá trị mặc định
	showHome := true
	if input.ShowHome != nil {
		showHome = *input.ShowHome
	}

	position := 0
	if input.Position != nil {
		position = *input.Position
	}

	section := &model.HomepageSection{
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		TypeKey:     input.TypeKey,
		Metadata:    datatypes.JSON(input.Metadata),
		Position:    position,
		ShowHome:    showHome,
	}

	if err := h.sectionRepo.Create(section); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể tạo section", err)
		return
	}

	helpers.SuccessResponse(c, "Tạo section thành công", section.ToResponse())
}

// GetSections lấy danh sách sections cho CMS (có phân trang)
func (h *HomepageSectionHandler) GetSections(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	search := strings.TrimSpace(c.Query("search"))

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}

	var sections []model.HomepageSection
	var total int64
	var err error

	// Use search function that supports both search and pagination
	sections, total, err = h.sectionRepo.SearchByTitle(search, page, limit)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách sections", err)
		return
	}

	responses := make([]model.HomepageSectionResponse, 0, len(sections))
	for _, section := range sections {
		responses = append(responses, section.ToResponse())
	}

	totalPages := (total + int64(limit) - 1) / int64(limit)

	helpers.SuccessResponse(c, "Lấy danh sách sections thành công", gin.H{
		"data": responses,
		"pagination": gin.H{
			"total":        total,
			"page":         page,
			"limit":        limit,
			"total_pages":  totalPages,
			"has_next":     int64(page*limit) < total,
			"has_previous": page > 1,
		},
	})
}

// GetSectionByID lấy section theo ID
func (h *HomepageSectionHandler) GetSectionByID(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID không hợp lệ", err)
		return
	}

	section, err := h.sectionRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy section", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy section thành công", section.ToResponse())
}

// GetSectionByTypeKey lấy section theo TypeKey
func (h *HomepageSectionHandler) GetSectionByTypeKey(c *gin.Context) {
	typeKey := c.Param("type_key")
	typeKey = strings.ToUpper(strings.TrimSpace(typeKey))

	section, err := h.sectionRepo.GetByTypeKey(typeKey)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy section", err)
		return
	}

	helpers.SuccessResponse(c, "Lấy section thành công", section.ToResponse())
}

// UpdateSection cập nhật section
func (h *HomepageSectionHandler) UpdateSection(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID không hợp lệ", err)
		return
	}

	section, err := h.sectionRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy section", err)
		return
	}

	var input model.HomepageSectionInput
	if err := c.ShouldBindJSON(&input); err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Dữ liệu đầu vào không hợp lệ", err)
		return
	}

	// Chuẩn hóa type_key
	input.TypeKey = strings.ToUpper(strings.TrimSpace(input.TypeKey))

	// Kiểm tra type_key đã tồn tại ở section khác chưa
	if input.TypeKey != section.TypeKey {
		exists, err := h.sectionRepo.CheckTypeKeyExists(input.TypeKey, id)
		if err != nil {
			helpers.ErrorResponse(c, http.StatusInternalServerError, "Lỗi cơ sở dữ liệu", err)
			return
		}
		if exists {
			helpers.ErrorResponse(c, http.StatusConflict, "TypeKey đã tồn tại", errors.New("section với type_key này đã tồn tại"))
			return
		}
	}

	// Cập nhật các trường
	section.Title = strings.TrimSpace(input.Title)
	section.Description = strings.TrimSpace(input.Description)
	section.TypeKey = input.TypeKey

	if input.Metadata != nil {
		section.Metadata = datatypes.JSON(input.Metadata)
	}

	if input.Position != nil {
		section.Position = *input.Position
	}

	if input.ShowHome != nil {
		section.ShowHome = *input.ShowHome
	}

	if err := h.sectionRepo.Update(section); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể cập nhật section", err)
		return
	}

	helpers.SuccessResponse(c, "Cập nhật section thành công", section.ToResponse())
}

// DeleteSection xóa section
func (h *HomepageSectionHandler) DeleteSection(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusBadRequest, "ID không hợp lệ", err)
		return
	}

	// Kiểm tra section có tồn tại không
	_, err = h.sectionRepo.GetByID(id)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy section", err)
		return
	}

	if err := h.sectionRepo.Delete(id); err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể xóa section", err)
		return
	}

	helpers.SuccessResponse(c, "Xóa section thành công", nil)
}

// GetPublicSections lấy danh sách sections công khai (cho người dùng)
func (h *HomepageSectionHandler) GetPublicSections(c *gin.Context) {
	sections, err := h.sectionRepo.GetPublic()
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách sections", err)
		return
	}

	var responses []model.HomepageSectionPublicResponse
	for _, section := range sections {
		responses = append(responses, section.ToPublicResponse())
	}

	helpers.SuccessResponse(c, "Lấy danh sách sections thành công", responses)
}

// GetPublicSectionByTypeKey lấy section công khai theo TypeKey
func (h *HomepageSectionHandler) GetPublicSectionByTypeKey(c *gin.Context) {
	typeKey := c.Param("type_key")
	typeKey = strings.ToUpper(strings.TrimSpace(typeKey))

	section, err := h.sectionRepo.GetByTypeKey(typeKey)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusNotFound, "Không tìm thấy section", err)
		return
	}

	// Chỉ trả về nếu show_home = true
	if !section.ShowHome {
		helpers.ErrorResponse(c, http.StatusNotFound, "Section không được hiển thị", errors.New("section is hidden"))
		return
	}

	helpers.SuccessResponse(c, "Lấy section thành công", section.ToPublicResponse())
}
