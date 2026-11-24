package model

import "gorm.io/gorm"

type Term struct {
	gorm.Model
	Query string `gorm:"uniqueIndex;not null"`
	Users []User `gorm:"many2many:user_terms;constraint:OnDelete:CASCADE"`
	CVEs  []CVE  `gorm:"constraint:OnDelete:CASCADE"`
}
