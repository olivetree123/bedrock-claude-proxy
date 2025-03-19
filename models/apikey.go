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

func CreateAPIKey(db *gorm.DB, name, value string) error {
	apiKey := APIKey{
		Name:  name,
		Value: value,
	}

	return db.Create(&apiKey).Error
}

func GetAPIKey(db *gorm.DB, value string) (APIKey, error) {
	var apiKey APIKey
	result := db.Where("value = ?", value).First(&apiKey)
	if result.Error != nil {
		return APIKey{}, result.Error
	}
	return apiKey, nil
}
