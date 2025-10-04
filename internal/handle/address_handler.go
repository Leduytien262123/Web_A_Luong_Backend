package handle

import (
	"backend/internal/helpers"
	"backend/internal/model"
	"backend/internal/repo"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AddressHandler struct {
	addressRepo *repo.AddressRepo
}

func NewAddressHandler() *AddressHandler {
	return &AddressHandler{
		addressRepo: repo.NewAddressRepo(),
	}
}

// GetAddressesByPhone lấy danh sách địa chỉ theo số điện thoại (API công khai)
func (h *AddressHandler) GetAddressesByPhone(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		helpers.ErrorResponse(c, http.StatusBadRequest, "Số điện thoại là bắt buộc", nil)
		return
	}

	addresses, err := h.addressRepo.GetAddressesByPhone(phone)
	if err != nil {
		helpers.ErrorResponse(c, http.StatusInternalServerError, "Không thể lấy danh sách địa chỉ", err)
		return
	}

	var response []model.AddressResponse
	for _, addr := range addresses {
		response = append(response, addr.ToResponse())
	}

	c.JSON(http.StatusOK, helpers.Response{
		Success: true,
		Message: "Lấy danh sách địa chỉ thành công",
		Data:    response,
	})
}