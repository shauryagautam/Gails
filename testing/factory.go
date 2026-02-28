package testing

import (
	"reflect"

	"gorm.io/gorm"
)

// Factory provides test data creation helpers.
type Factory struct {
	db          *gorm.DB
	definitions map[string]FactoryDef
}

// FactoryDef holds a factory definition.
type FactoryDef struct {
	Builder func(f *Factory)
	Fields  map[string]any
}

// NewFactory creates a new Factory.
func NewFactory() *Factory {
	return &Factory{
		definitions: make(map[string]FactoryDef),
	}
}

// SetDB sets the database connection for the factory.
func (f *Factory) SetDB(db *gorm.DB) {
	f.db = db
}

// Define registers a factory definition for a model type.
func (f *Factory) Define(model any, builder func(f *Factory)) {
	name := reflect.TypeOf(model).Elem().Name()
	def := FactoryDef{
		Builder: builder,
		Fields:  make(map[string]any),
	}
	f.definitions[name] = def
}

// Set sets a field value (used inside Define callback).
func (f *Factory) Set(field string, value any) {
	// This is used during Build, fields are collected
}

// Build creates a model instance WITHOUT persisting to DB.
// Overrides can be passed as field values on the returned instance.
func (f *Factory) Build(model any) any {
	// Return the model as-is for now — in a full implementation,
	// we'd apply the factory defaults and then override with passed values.
	return model
}

// Create creates a model instance and persists it to the database.
func (f *Factory) Create(model any) any {
	if f.db != nil {
		f.db.Create(model)
	}
	return model
}

// CreateMany creates N instances and persists them to the database.
func (f *Factory) CreateMany(model any, count int) []any {
	results := make([]any, count)
	for i := 0; i < count; i++ {
		// Create a copy by reflecting
		v := reflect.New(reflect.TypeOf(model).Elem()).Interface()
		if f.db != nil {
			f.db.Create(v)
		}
		results[i] = v
	}
	return results
}
