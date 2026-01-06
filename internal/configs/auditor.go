package configs

import (
	"sistema-pasajes/internal/appcontext"

	"gorm.io/gorm"
)

func RegisterAuditCallbacks(db *gorm.DB) {
	callback := db.Callback()

	callback.Create().Before("gorm:create").Register("audit:before_create", func(tx *gorm.DB) {
		setAuditField(tx, "CreatedBy")
	})

	callback.Update().Before("gorm:update").Register("audit:before_update", func(tx *gorm.DB) {
		setAuditField(tx, "UpdatedBy")
	})

	callback.Delete().Before("gorm:delete").Register("audit:before_delete", func(tx *gorm.DB) {
		setAuditField(tx, "DeletedBy")
	})
}

func setAuditField(tx *gorm.DB, fieldName string) {
	if tx.Statement.Schema == nil {
		return
	}

	userID := appcontext.AuthID()
	if userID == nil {
		return
	}
	field := tx.Statement.Schema.LookUpField(fieldName)
	if field != nil {
		field.Set(tx.Statement.Context, tx.Statement.ReflectValue, *userID)
	}
}
