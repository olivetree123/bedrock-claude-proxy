package pkg

import (
	"bedrock-claude-proxy/models"

	"gorm.io/gorm"
)

// LogAPIUsage 记录API使用情况
func LogAPIUsage(db *gorm.DB, apiKeyName, apiKeyValue, modelName string, inputTokens, outputTokens int) error {
	log := models.Log{
		APIKeyName:  apiKeyName,
		APIKeyValue: apiKeyValue,
		ModelName:   modelName,
		InputToken:  inputTokens,
		OutputToken: outputTokens,
	}
	
	result := db.Create(&log)
	return result.Error
} 