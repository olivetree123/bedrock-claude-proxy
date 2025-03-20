package api

import (
	"bedrock-claude-proxy/log"
	"bedrock-claude-proxy/models"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"gorm.io/gorm"
)

// 分页参数
type PaginationQuery struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// Usage列表响应
type ListUsageResponse struct {
	Total int64         `json:"total"`
	Items []UsageItem   `json:"items"`
}

// Usage项目
type UsageItem struct {
	ID           uint      `json:"id"`
	APIKeyName   string    `json:"apikey_name"`
	ModelName    string    `json:"model_name"`
	InputTokens  int       `json:"input_tokens"`
	OutputTokens int       `json:"output_tokens"`
	Quota        int       `json:"quota"`
	CreatedAt    time.Time `json:"created_at"`
}

// 列出使用记录
func ListUsage(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只接受GET请求
		if r.Method != http.MethodGet {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 解析分页参数
		page := 1
		pageSize := 20

		if pageParam := r.URL.Query().Get("page"); pageParam != "" {
			if p, err := strconv.Atoi(pageParam); err == nil && p > 0 {
				page = p
			}
		}

		if pageSizeParam := r.URL.Query().Get("page_size"); pageSizeParam != "" {
			if ps, err := strconv.Atoi(pageSizeParam); err == nil && ps > 0 && ps <= 100 {
				pageSize = ps
			}
		}

		// 解析过滤参数
		apiKeyName := r.URL.Query().Get("apikey_name")
		modelName := r.URL.Query().Get("model_name")
		startTimeStr := r.URL.Query().Get("start_time") // 格式: 2006-01-02
		endTimeStr := r.URL.Query().Get("end_time")     // 格式: 2006-01-02

		// 构建查询
		query := db.Model(&models.Usage{})

		// 应用过滤条件
		if apiKeyName != "" {
			query = query.Where("apikey_name = ?", apiKeyName)
		}

		if modelName != "" {
			query = query.Where("model_name = ?", modelName)
		}

		if startTimeStr != "" {
			startTime, err := time.Parse("2006-01-02", startTimeStr)
			if err == nil {
				query = query.Where("created_at >= ?", startTime)
			}
		}

		if endTimeStr != "" {
			endTime, err := time.Parse("2006-01-02", endTimeStr)
			if err == nil {
				// 将结束日期设置为当天的最后一刻
				endTime = endTime.Add(24*time.Hour - time.Second)
				query = query.Where("created_at <= ?", endTime)
			}
		}

		// 计算总记录数
		var total int64
		if err := query.Count(&total).Error; err != nil {
			log.Logger.Errorf("查询使用记录总数失败: %v", err)
			http.Error(w, "查询使用记录失败", http.StatusInternalServerError)
			return
		}

		// 查询使用记录
		var usages []models.Usage
		if err := query.Order("created_at DESC").
			Limit(pageSize).
			Offset((page - 1) * pageSize).
			Find(&usages).Error; err != nil {
			log.Logger.Errorf("查询使用记录失败: %v", err)
			http.Error(w, "查询使用记录失败", http.StatusInternalServerError)
			return
		}

		// 转换为响应格式
		items := make([]UsageItem, len(usages))
		for i, usage := range usages {
			items[i] = UsageItem{
				ID:           usage.ID,
				APIKeyName:   usage.APIKeyName,
				ModelName:    usage.ModelName,
				InputTokens:  usage.InputTokens,
				OutputTokens: usage.OutputTokens,
				Quota:        usage.Quota,
				CreatedAt:    usage.CreatedAt,
			}
		}

		// 返回响应
		response := ListUsageResponse{
			Total: total,
			Items: items,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Logger.Errorf("编码响应失败: %v", err)
			http.Error(w, "服务器内部错误", http.StatusInternalServerError)
			return
		}
	}
}
