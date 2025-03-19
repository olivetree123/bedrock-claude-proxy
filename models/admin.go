package models

import (
	"gorm.io/gorm"
)

type Admin struct {
	gorm.Model
	Username string `gorm:"not null;varchar(255)" json:"username"`
	Password string `gorm:"not null;varchar(255)" json:"password"`
}

func (Admin) TableName() string {
	return "admin"
}
