package pkg

import (
	"bedrock-claude-proxy/models"
	"crypto/sha256"
	"encoding/hex"

	"gorm.io/gorm"
)

// InitDB 初始化数据库，执行迁移操作
func InitDB(db *gorm.DB) error {
	// 自动迁移数据库模型
	err := db.AutoMigrate(&models.Admin{}, &models.APIKey{}, &models.Log{})
	if err != nil {
		return err
	}

	// 检查是否需要创建默认管理员账户
	var count int64
	db.Model(&models.Admin{}).Count(&count)

	// 如果没有管理员账户，则创建默认管理员账户
	if count == 0 {
		defaultAdmin := models.Admin{
			Username: "proxy",
			Password: HashPassword("hello@autel.com"), // 默认密码
		}

		result := db.Create(&defaultAdmin)
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

// HashPassword 哈希密码
func HashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}
