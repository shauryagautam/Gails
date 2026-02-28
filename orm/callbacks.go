package orm

import "gorm.io/gorm"

// These interfaces are provided for documentation and type-checking,
// though GORM uses reflection to call these methods automatically.

type BeforeCreateHook interface {
	BeforeCreate(tx *gorm.DB) error
}

type AfterCreateHook interface {
	AfterCreate(tx *gorm.DB) error
}

type BeforeUpdateHook interface {
	BeforeUpdate(tx *gorm.DB) error
}

type AfterUpdateHook interface {
	AfterUpdate(tx *gorm.DB) error
}

type BeforeSaveHook interface {
	BeforeSave(tx *gorm.DB) error
}

type AfterSaveHook interface {
	AfterSave(tx *gorm.DB) error
}

type BeforeDeleteHook interface {
	BeforeDelete(tx *gorm.DB) error
}

type AfterDeleteHook interface {
	AfterDelete(tx *gorm.DB) error
}

type AfterFindHook interface {
	AfterFind(tx *gorm.DB) error
}
