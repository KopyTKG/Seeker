package model

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Tag   string `gorm:"uniqueIndex;not null"`
	Terms []Term `gorm:"many2many:user_terms;constraint:OnDelete:CASCADE"`
}
