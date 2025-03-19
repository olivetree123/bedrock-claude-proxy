package api

import (
	log "bedrock-claude-proxy/log"
	"bedrock-claude-proxy/models"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

// API密钥创建请求
type CreateAPIKeyRequest struct {
	Name string `json:"name"`
}

// API密钥响应
type APIKeyResponse struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// 列表响应
type ListAPIKeysResponse struct {
	APIKeys []APIKeyResponse `json:"api_keys"`
}

// 创建API密钥
func CreateAPIKey(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只接受POST请求
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 解析请求体
		var req CreateAPIKeyRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			log.Logger.Errorf("Failed to decode request: %v", err)
			http.Error(w, "Invalid request format", http.StatusBadRequest)
			return
		}

		// 验证名称
		if req.Name == "" {
			http.Error(w, "Name is required", http.StatusBadRequest)
			return
		}

		// 检查名称是否已存在
		var existingKey models.APIKey
		result := db.Where("name = ?", req.Name).First(&existingKey)
		if result.Error == nil {
			http.Error(w, "API key with this name already exists", http.StatusConflict)
			return
		}

		// 生成API密钥
		apiKeyValue, err := generateAPIKey()
		if err != nil {
			log.Logger.Errorf("Failed to generate API key: %v", err)
			http.Error(w, "Failed to generate API key", http.StatusInternalServerError)
			return
		}

		// 创建新的API密钥
		apiKey := models.APIKey{
			Name:  req.Name,
			Value: apiKeyValue,
		}

		// 保存到数据库
		result = db.Create(&apiKey)
		if result.Error != nil {
			log.Logger.Errorf("Failed to save API key: %v", result.Error)
			http.Error(w, "Failed to create API key", http.StatusInternalServerError)
			return
		}

		// 返回创建的API密钥
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(APIKeyResponse{
			ID:    apiKey.ID,
			Name:  apiKey.Name,
			Value: apiKey.Value,
		})

		log.Logger.Infof("API key created: %s", req.Name)
	}
}

// 列出所有API密钥
func ListAPIKeys(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只接受GET请求
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 从数据库中获取所有API密钥
		var apiKeys []models.APIKey
		result := db.Find(&apiKeys)
		if result.Error != nil {
			log.Logger.Errorf("Failed to fetch API keys: %v", result.Error)
			http.Error(w, "Failed to fetch API keys", http.StatusInternalServerError)
			return
		}

		// 转换为响应格式
		response := ListAPIKeysResponse{
			APIKeys: make([]APIKeyResponse, len(apiKeys)),
		}
		for i, key := range apiKeys {
			response.APIKeys[i] = APIKeyResponse{
				ID:    key.ID,
				Name:  key.Name,
				Value: key.Value,
			}
		}

		// 返回API密钥列表
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// 删除API密钥
func DeleteAPIKey(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只接受DELETE请求
		if r.Method != http.MethodDelete {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 从URL获取API密钥ID
		vars := mux.Vars(r)
		id := vars["id"]
		if id == "" {
			http.Error(w, "API key ID is required", http.StatusBadRequest)
			return
		}

		// 删除API密钥
		result := db.Delete(&models.APIKey{}, id)
		if result.Error != nil {
			log.Logger.Errorf("Failed to delete API key: %v", result.Error)
			http.Error(w, "Failed to delete API key", http.StatusInternalServerError)
			return
		}

		if result.RowsAffected == 0 {
			http.Error(w, "API key not found", http.StatusNotFound)
			return
		}

		// 返回成功
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"message": "API key deleted successfully"}`))

		log.Logger.Infof("API key deleted: ID %s", id)
	}
}

// 生成随机API密钥
func generateAPIKey() (string, error) {
	bytes := make([]byte, 32) // 256位
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return fmt.Sprintf("bk-%s", hex.EncodeToString(bytes)), nil
}
