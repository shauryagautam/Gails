package orm

import (
	"gorm.io/gorm"
)

// QueryBuilder provides a generic, chainable query interface wrapping GORM.
type QueryBuilder[T any] struct {
	db      *gorm.DB
	page    int
	perPage int
}

// Query creates a new QueryBuilder for the given model type.
func Query[T any](db *gorm.DB) *QueryBuilder[T] {
	return &QueryBuilder[T]{db: db, perPage: 25}
}

// Where adds a WHERE clause.
func (q *QueryBuilder[T]) Where(query any, args ...any) *QueryBuilder[T] {
	q.db = q.db.Where(query, args...)
	return q
}

// Order adds an ORDER BY clause.
func (q *QueryBuilder[T]) Order(value any) *QueryBuilder[T] {
	q.db = q.db.Order(value)
	return q
}

// Limit sets the LIMIT.
func (q *QueryBuilder[T]) Limit(limit int) *QueryBuilder[T] {
	q.db = q.db.Limit(limit)
	return q
}

// Offset sets the OFFSET.
func (q *QueryBuilder[T]) Offset(offset int) *QueryBuilder[T] {
	q.db = q.db.Offset(offset)
	return q
}

// Page sets the page number for pagination (1-indexed).
func (q *QueryBuilder[T]) Page(page int) *QueryBuilder[T] {
	if page < 1 {
		page = 1
	}
	q.page = page
	return q
}

// PerPage sets the number of records per page.
func (q *QueryBuilder[T]) PerPage(perPage int) *QueryBuilder[T] {
	if perPage < 1 {
		perPage = 25
	}
	q.perPage = perPage
	return q
}

// Scope applies one or more named scopes (functions that modify the query).
func (q *QueryBuilder[T]) Scope(funcs ...func(*gorm.DB) *gorm.DB) *QueryBuilder[T] {
	for _, f := range funcs {
		q.db = f(q.db)
	}
	return q
}

// With eager-loads an association (wraps GORM Preload).
func (q *QueryBuilder[T]) With(query string, args ...any) *QueryBuilder[T] {
	q.db = q.db.Preload(query, args...)
	return q
}

// applyPagination applies page/perPage to the underlying DB query.
func (q *QueryBuilder[T]) applyPagination() *gorm.DB {
	db := q.db
	if q.page > 0 {
		offset := (q.page - 1) * q.perPage
		db = db.Offset(offset).Limit(q.perPage)
	}
	return db
}

// All returns all matching records.
func (q *QueryBuilder[T]) All() ([]T, error) {
	var results []T
	err := q.applyPagination().Find(&results).Error
	return results, err
}

// First returns the first matching record.
func (q *QueryBuilder[T]) First() (*T, error) {
	var result T
	err := q.db.First(&result).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Find returns a record by primary key.
func (q *QueryBuilder[T]) Find(id any) (*T, error) {
	var result T
	err := q.db.First(&result, id).Error
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Create inserts a new record.
func (q *QueryBuilder[T]) Create(v *T) error {
	return q.db.Create(v).Error
}

// Update saves changes to an existing record.
func (q *QueryBuilder[T]) Update(v *T) error {
	return q.db.Save(v).Error
}

// Delete soft-deletes (or hard-deletes) a record.
func (q *QueryBuilder[T]) Delete(v *T) error {
	return q.db.Delete(v).Error
}

// Count returns the number of matching records.
func (q *QueryBuilder[T]) Count() (int64, error) {
	var count int64
	var model T
	err := q.db.Model(&model).Count(&count).Error
	return count, err
}

// Exists returns true if at least one matching record exists.
func (q *QueryBuilder[T]) Exists(query any, args ...any) (bool, error) {
	var count int64
	var model T
	err := q.db.Model(&model).Where(query, args...).Count(&count).Error
	return count > 0, err
}
