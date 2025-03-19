package models

import (
	"gorm.io/gorm"
)

type APIKey struct {
	gorm.Model
	Name  string `gorm:"not null;unique;varchar(255)" json:"name"`
	Value string `gorm:"not null;unique;varchar(255)" json:"value"`
}

func (APIKey) TableName() string {
	return "apikey"
}
