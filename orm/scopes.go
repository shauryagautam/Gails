package orm

import "gorm.io/gorm"

// ScopeFunc is a named scope — a function that modifies a GORM query.
// Usage: orm.Query[User](db).Scope(ActiveUsers).All()
type ScopeFunc = func(*gorm.DB) *gorm.DB
