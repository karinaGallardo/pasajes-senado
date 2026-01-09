package configs

import (
	"reflect"
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

	if tx.Statement.Context == nil {
		return
	}

	userID := appcontext.GetUserIDFromContext(tx.Statement.Context)
	if userID == nil {
		return
	}

	field := tx.Statement.Schema.LookUpField(fieldName)
	if field == nil {
		return
	}

	switch tx.Statement.ReflectValue.Kind() {
	case reflect.Slice, reflect.Array:
		for i := 0; i < tx.Statement.ReflectValue.Len(); i++ {
			elem := tx.Statement.ReflectValue.Index(i)
			if elem.Kind() == reflect.Ptr {
				elem = elem.Elem()
			}
			if elem.Kind() == reflect.Struct {
				f := elem.FieldByName(fieldName)
				if f.IsValid() && f.CanSet() {
					f.Set(reflect.ValueOf(userID))
				}
			}
		}
	case reflect.Struct:
		field.Set(tx.Statement.Context, tx.Statement.ReflectValue, *userID)
	}
}
