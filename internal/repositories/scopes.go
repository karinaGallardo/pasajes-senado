package repositories

import (
	"gorm.io/gorm"
)

func Paginate(page, pageSize int) func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if page <= 0 {
			page = 1
		}
		switch {
		case pageSize > 100:
			pageSize = 100
		case pageSize <= 0:
			pageSize = 10
		}

		offset := (page - 1) * pageSize
		return db.Offset(offset).Limit(pageSize)
	}
}

func OrderByCreatedDesc(db *gorm.DB) *gorm.DB {
	return db.Order("created_at DESC")
}

func IsActive(db *gorm.DB) *gorm.DB {
	return db.Where("estado = ?", "ACTIVO")
}
