package models

import (
	"gorm.io/gorm"
)

type Usage struct {
	gorm.Model
	APIKeyName  string `gorm:"not null;varchar(255)" json:"apikey_name"`
	APIKeyValue string `gorm:"not null;varchar(255)" json:"apikey_value"`
	ModelName    string  `gorm:"not null;varchar(255)" json:"model_name"`
	InputTokens  int     `gorm:"not null;int" json:"input_tokens"`     // 输入token数量
	OutputTokens int     `gorm:"not null;int" json:"output_tokens"`    // 输出token数量
	Quota        int     `gorm:"not null;int;default:0" json:"quota"`  // 额度，乘以0.002就是美元
}

func (Usage) TableName() string {
	return "usage"
}

func CreateUsage(db *gorm.DB, apiKeyName, apiKeyValue, modelName string, inputTokens, outputTokens int, quota int) error {
	log := Usage{
		APIKeyName:  apiKeyName,
		APIKeyValue:  apiKeyValue,
		ModelName:    modelName,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		Quota:        quota,
	}

	result := db.Create(&log)
	return result.Error
}
