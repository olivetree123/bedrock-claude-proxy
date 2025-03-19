package models

import (
	"gorm.io/gorm"
)

type Usage struct {
	gorm.Model
	APIKeyName  string `gorm:"not null;varchar(255)" json:"apikey_name"`
	APIKeyValue string `gorm:"not null;varchar(255)" json:"apikey_value"`
	ModelName   string `gorm:"not null;varchar(255)" json:"model_name"`
	InputTokens  int    `gorm:"not null;int" json:"input_tokens"`
	OutputTokens int    `gorm:"not null;int" json:"output_tokens"`
}

func (Usage) TableName() string {
	return "usage"
}

func CreateUsage(db *gorm.DB, apiKeyName, apiKeyValue, modelName string, inputTokens, outputTokens int) error {
	log := Usage{
		APIKeyName:  apiKeyName,
		APIKeyValue: apiKeyValue,
		ModelName:   modelName,
		InputTokens: inputTokens,
		OutputTokens: outputTokens,
	}

	result := db.Create(&log)
	return result.Error
}
