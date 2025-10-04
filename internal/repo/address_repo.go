package repo

import (
	"backend/app"
	"backend/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type AddressRepo struct {
	db *gorm.DB
}

func NewAddressRepo() *AddressRepo {
	return &AddressRepo{
		db: app.DB,
	}
}

// GetAddressesByPhone lấy tất cả địa chỉ theo số điện thoại, sắp xếp default lên đầu
func (r *AddressRepo) GetAddressesByPhone(phone string) ([]model.Address, error) {
	var addresses []model.Address
	err := r.db.Where("phone = ?", phone).
		Order("is_default DESC, created_at ASC").
		Find(&addresses).Error
	return addresses, err
}

// BulkCreateAddresses xóa tất cả địa chỉ cũ của user, tạo mới từ danh sách (phần tử đầu làm default)
func (r *AddressRepo) BulkCreateAddresses(userID uuid.UUID, addresses []model.Address) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// Xóa tất cả địa chỉ cũ của user
		if err := tx.Where("user_id = ?", userID).Delete(&model.Address{}).Error; err != nil {
			return err
		}
		
		// Tạo địa chỉ mới
		for i, addr := range addresses {
			addr.UserID = userID
			addr.IsDefault = (i == 0) // Phần tử đầu tiên làm default
			if err := tx.Create(&addr).Error; err != nil {
				return err
			}
		}
		
		return nil
	})
}

// SaveAddressFromOrder lưu địa chỉ từ đơn hàng nếu số điện thoại chưa tồn tại
func (r *AddressRepo) SaveAddressFromOrder(userID *uuid.UUID, name, phone, addressLine1 string) error {
	if addressLine1 == "" || phone == "" {
		return nil // Không có đầy đủ thông tin để lưu
	}

	// Kiểm tra xem số điện thoại này đã có địa chỉ nào chưa
	var existingCount int64
	err := r.db.Model(&model.Address{}).Where("phone = ?", phone).Count(&existingCount).Error
	if err != nil {
		return err
	}
	
	// Nếu số điện thoại đã có địa chỉ thì không thêm mới
	if existingCount > 0 {
		return nil
	}

	// Tạo địa chỉ mới cho số điện thoại này
	newAddress := model.Address{
		Name:         name,
		Phone:        phone,
		AddressLine1: addressLine1,
		AddressLine2: "",
		City:         "N/A",
		State:        "N/A",
		PostalCode:   "000000",
		Country:      "Vietnam",
		IsDefault:    true, // Địa chỉ đầu tiên của số điện thoại luôn là default
	}
	
	// Set UserID nếu có
	if userID != nil {
		newAddress.UserID = *userID
	}
	
	return r.db.Create(&newAddress).Error
}