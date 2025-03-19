package models

import (
	"gorm.io/gorm"
)

type Log struct {
	gorm.Model
	APIKeyName  string `gorm:"not null;varchar(255)" json:"apikey_name"`
	APIKeyValue string `gorm:"not null;varchar(255)" json:"apikey_value"`
	ModelName   string `gorm:"not null;varchar(255)" json:"model_name"`
	InputToken  int    `gorm:"not null;int" json:"input_token"`
	OutputToken int    `gorm:"not null;int" json:"output_token"`
}

func (Log) TableName() string {
	return "log"
}
