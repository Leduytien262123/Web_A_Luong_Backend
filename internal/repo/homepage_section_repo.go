package repo

import (
	"backend/app"
	"backend/internal/model"
	"errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type HomepageSectionRepo struct {
	db *gorm.DB
}

func NewHomepageSectionRepo() *HomepageSectionRepo {
	return &HomepageSectionRepo{
		db: app.GetDB(),
	}
}

// Create tạo section mới
func (r *HomepageSectionRepo) Create(section *model.HomepageSection) error {
	return r.db.Create(section).Error
}

// GetByID lấy section theo ID
func (r *HomepageSectionRepo) GetByID(id uuid.UUID) (*model.HomepageSection, error) {
	var section model.HomepageSection
	err := r.db.Where("id = ?", id).First(&section).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("homepage section not found")
		}
		return nil, err
	}
	return &section, nil
}

// GetByTypeKey lấy section theo TypeKey
func (r *HomepageSectionRepo) GetByTypeKey(typeKey string) (*model.HomepageSection, error) {
	var section model.HomepageSection
	err := r.db.Where("type_key = ?", typeKey).First(&section).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("homepage section not found")
		}
		return nil, err
	}
	return &section, nil
}

// CheckTypeKeyExists kiểm tra TypeKey đã tồn tại chưa (trừ excludeID)
func (r *HomepageSectionRepo) CheckTypeKeyExists(typeKey string, excludeID uuid.UUID) (bool, error) {
	var count int64
	query := r.db.Model(&model.HomepageSection{}).Where("type_key = ?", typeKey)
	if excludeID != uuid.Nil {
		query = query.Where("id != ?", excludeID)
	}
	if err := query.Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetAll lấy tất cả sections với phân trang và sắp xếp theo position
func (r *HomepageSectionRepo) GetAll(page, limit int) ([]model.HomepageSection, int64, error) {
	var sections []model.HomepageSection
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.HomepageSection{})

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("position ASC, created_at DESC").
		Limit(limit).Offset(offset).Find(&sections).Error
	if err != nil {
		return nil, 0, err
	}

	return sections, total, nil
}

// SearchByTitle tìm kiếm sections theo title với phân trang
func (r *HomepageSectionRepo) SearchByTitle(keyword string, page, limit int) ([]model.HomepageSection, int64, error) {
	var sections []model.HomepageSection
	var total int64

	offset := (page - 1) * limit

	query := r.db.Model(&model.HomepageSection{})

	// Apply search filter if keyword is provided
	if keyword != "" {
		searchPattern := "%" + keyword + "%"
		query = query.Where("title LIKE ? OR type_key LIKE ?", searchPattern, searchPattern)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := query.Order("position ASC, created_at DESC").
		Limit(limit).Offset(offset).Find(&sections).Error
	if err != nil {
		return nil, 0, err
	}

	return sections, total, nil
}

// GetPublic lấy tất cả sections công khai (show_home = true) sắp xếp theo position
func (r *HomepageSectionRepo) GetPublic() ([]model.HomepageSection, error) {
	var sections []model.HomepageSection

	err := r.db.Where("show_home = ?", true).
		Order("position ASC, created_at DESC").
		Find(&sections).Error
	if err != nil {
		return nil, err
	}

	return sections, nil
}

// Update cập nhật section
func (r *HomepageSectionRepo) Update(section *model.HomepageSection) error {
	return r.db.Save(section).Error
}

// Delete xóa section (soft delete)
func (r *HomepageSectionRepo) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.HomepageSection{}, "id = ?", id).Error
}

// HardDelete xóa vĩnh viễn section
func (r *HomepageSectionRepo) HardDelete(id uuid.UUID) error {
	return r.db.Unscoped().Delete(&model.HomepageSection{}, "id = ?", id).Error
}
