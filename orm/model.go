package orm

import (
	"fmt"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// Model is the base model for all Gails models, equivalent to ActiveRecord::Base.
type Model struct {
	ID        uint `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

// IDString returns the model's ID as a string.
func (m *Model) IDString() string {
	return strconv.Itoa(int(m.ID))
}

// CacheKey generates a versioned cache key based on model type, ID, and last update.
func (m *Model) CacheKey() string {
	return fmt.Sprintf("%T-%d-v%d", m, m.ID, m.UpdatedAt.Unix())
}
