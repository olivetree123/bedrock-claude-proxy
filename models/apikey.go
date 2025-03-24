package models

import (
	"gorm.io/gorm"
)

type APIKey struct {
	gorm.Model
	Name    string `gorm:"column:name;not null;unique;varchar(255)" json:"name"`
	Value   string `gorm:"column:value;not null;unique;varchar(255)" json:"value"`
	Enable  bool   `gorm:"column:enable;not null;default:true;bool" json:"enable"`
}

func (APIKey) TableName() string {
	return "apikey"
}

func CreateAPIKey(db *gorm.DB, name, value string) error {
	apiKey := APIKey{
		Name:    name,
		Value:   value,
		Enable:  true,
	}

	return db.Create(&apiKey).Error
}

func GetAPIKey(db *gorm.DB, value string) (APIKey, error) {
	var apiKey APIKey
	result := db.Where("value = ? and enable = ?", value, true).First(&apiKey)
	if result.Error != nil {
		return APIKey{}, result.Error
	}
	return apiKey, nil
}

func UpdateAPIKeyStatusByName(db *gorm.DB, name string, enable bool) error {
	return db.Model(&APIKey{}).Where("name = ?", name).Update("enable", enable).Error
}
