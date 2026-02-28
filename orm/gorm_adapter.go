package orm

import (
	"gorm.io/gorm"
)

type GormAdapter struct {
	DB *gorm.DB
}

func NewGormAdapter(db *gorm.DB) *GormAdapter {
	return &GormAdapter{DB: db}
}
