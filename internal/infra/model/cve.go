package model

import (
	"time"
)

type CVE struct {
	ID          string `gorm:"primaryKey;uniqueIndex;not null"`
	Title       string `gorm:"not null"`
	Description string `gorm:"not null"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Score       float32
	Severity    string `gorm:"not null"`
	Vector      string `gorm:"not null"`
	Vendors     string `gorm:"not null"`
	Link        string `gorm:"not null"`
	TermID      uint
	Term        Term `gorm:"constraint:OnDelete:CASCADE"`
}
