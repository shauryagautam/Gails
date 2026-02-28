package db

import (
	"fmt"
	"math/rand"
	"os"
	"time"

	"gorm.io/gorm"
)

// SeedRecord tracks which named seeds have been run.
type SeedRecord struct {
	ID        uint   `gorm:"primarykey"`
	Name      string `gorm:"uniqueIndex;not null"`
	CreatedAt time.Time
}

// Once runs a named seed only once (tracked in the seeds table).
func Once(db *gorm.DB, name string, fn func() error) error {
	// Ensure seeds table exists
	db.AutoMigrate(&SeedRecord{})

	var count int64
	db.Model(&SeedRecord{}).Where("name = ?", name).Count(&count)
	if count > 0 {
		fmt.Printf("[Gails] Seed '%s' already run, skipping\n", name)
		return nil
	}

	if err := fn(); err != nil {
		return fmt.Errorf("[Gails] Seed '%s' failed: %w", name, err)
	}

	db.Create(&SeedRecord{Name: name})
	fmt.Printf("[Gails] Seed '%s' completed\n", name)
	return nil
}

// OnlyIn runs a function only in the specified environment.
func OnlyIn(db *gorm.DB, env string, fn func() error) error {
	currentEnv := os.Getenv("APP_ENV")
	if currentEnv == "" {
		currentEnv = "development"
	}
	if currentEnv != env {
		return nil
	}
	return fn()
}

// Faker provides fake data generation helpers.
type Faker struct {
	r *rand.Rand
}

// NewFaker creates a new Faker instance.
func NewFaker() *Faker {
	return &Faker{r: rand.New(rand.NewSource(time.Now().UnixNano()))}
}

// Name returns a fake name.
func (f *Faker) Name() string {
	first := []string{"Alice", "Bob", "Charlie", "Diana", "Eve", "Frank", "Grace", "Henry", "Ivy", "Jack",
		"Kate", "Liam", "Mia", "Noah", "Olivia", "Paul", "Quinn", "Ryan", "Sarah", "Tom"}
	last := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller", "Davis", "Rodriguez", "Martinez"}
	return first[f.r.Intn(len(first))] + " " + last[f.r.Intn(len(last))]
}

// Email returns a fake email address.
func (f *Faker) Email() string {
	users := []string{"alice", "bob", "charlie", "diana", "eve", "frank", "grace", "henry", "ivy", "jack"}
	domains := []string{"example.com", "test.com", "demo.com", "mail.test"}
	return fmt.Sprintf("%s%d@%s", users[f.r.Intn(len(users))], f.r.Intn(1000), domains[f.r.Intn(len(domains))])
}

// Sentence returns a fake sentence.
func (f *Faker) Sentence() string {
	sentences := []string{
		"The quick brown fox jumps over the lazy dog.",
		"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
		"Go is an open-source programming language.",
		"Gails makes web development in Go feel natural.",
		"Building production-grade applications with batteries included.",
	}
	return sentences[f.r.Intn(len(sentences))]
}

// IntBetween returns a random int between min and max (inclusive).
func (f *Faker) IntBetween(min, max int) int {
	return min + f.r.Intn(max-min+1)
}

// Bool returns a random boolean.
func (f *Faker) Bool() bool {
	return f.r.Intn(2) == 1
}

// Fake creates N fake records using a builder function and inserts them into the database.
func Fake[T any](db *gorm.DB, count int, builder func(f *Faker) T) error {
	faker := NewFaker()
	for i := 0; i < count; i++ {
		record := builder(faker)
		if err := db.Create(&record).Error; err != nil {
			return fmt.Errorf("failed to create fake record %d: %w", i+1, err)
		}
	}
	fmt.Printf("[Gails] Created %d fake records\n", count)
	return nil
}
