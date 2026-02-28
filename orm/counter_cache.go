package orm

import (
	"fmt"
	"reflect"

	"gorm.io/gorm"
)

// RegisterCounterCache sets up hooks to automatically update a counter field on a parent model.
func RegisterCounterCache(db *gorm.DB, relationName string, counterField string) {
	// AfterCreate hook
	db.Callback().Create().After("gorm:create").Register("gails:counter_cache_inc_"+relationName, func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Schema == nil {
			return
		}

		rel, ok := tx.Statement.Schema.Relationships.Relations[relationName]
		if !ok {
			return
		}

		// Get parent ID from the created model
		v := reflectValue(tx.Statement.Model)
		fieldValue, isZero := rel.Field.ValueOf(tx.Statement.Context, v)
		if isZero || fieldValue == nil {
			return
		}

		// Update parent count
		tx.Session(&gorm.Session{NewDB: true}).
			Table(rel.FieldSchema.Table).
			Where("id = ?", fieldValue).
			UpdateColumn(counterField, gorm.Expr(counterField+" + ?", 1))
	})

	// AfterDelete hook
	db.Callback().Delete().After("gorm:delete").Register("gails:counter_cache_dec_"+relationName, func(tx *gorm.DB) {
		if tx.Error != nil || tx.Statement.Schema == nil {
			return
		}

		rel, ok := tx.Statement.Schema.Relationships.Relations[relationName]
		if !ok {
			return
		}

		v := reflectValue(tx.Statement.Model)
		fieldValue, isZero := rel.Field.ValueOf(tx.Statement.Context, v)
		if isZero || fieldValue == nil {
			return
		}

		tx.Session(&gorm.Session{NewDB: true}).
			Table(rel.FieldSchema.Table).
			Where("id = ?", fieldValue).
			UpdateColumn(counterField, gorm.Expr(counterField+" - ?", 1))
	})
}

func reflectValue(obj any) reflect.Value {
	v := reflect.ValueOf(obj)
	for v.Kind() == reflect.Ptr {
		v = v.Elem()
	}
	return v
}

// RecountAll utility to fix desync
func RecountAll[T any](db *gorm.DB, counterField string, childTable string, foreignKey string) error {
	var model T
	stmt := &gorm.Statement{DB: db}
	stmt.Parse(&model)
	parentTable := stmt.Schema.Table

	query := fmt.Sprintf(`
		UPDATE %s p
		SET %s = (
			SELECT COUNT(*)
			FROM %s c
			WHERE c.%s = p.id
		)
	`, parentTable, counterField, childTable, foreignKey)

	return db.Exec(query).Error
}
