package pkg

import (
	"bedrock-claude-proxy/api"
	"bedrock-claude-proxy/models"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"

	log "bedrock-claude-proxy/log"

	"github.com/gorilla/mux"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type HttpConfig struct {
	Listen  string `json:"listen,omitempty"`
	WebRoot string `json:"web_root,omitempty"`
	APIKey  string `json:"api_key,omitempty"`
	DBPath  string `json:"db_path,omitempty"`
}

type HTTPService struct {
	conf        *Config
	db          *gorm.DB
	apiKeyCache map[string]*models.APIKey
	cacheMutex  sync.RWMutex
}

type APIError struct {
	Type    string `json:"type,omitempty"`
	Message string `json:"message,omitempty"`
}

type APIStandardError struct {
	Type  string    `json:"type,omitempty"`
	Error *APIError `json:"error,omitempty"`
}

func NewHttpService(conf *Config) *HTTPService {
	// 直接从环境变量读取MySQL连接信息
	dbHost := os.Getenv("MYSQL_HOST")
	dbPort := os.Getenv("MYSQL_PORT")
	dbUser := os.Getenv("MYSQL_USER")
	dbPassword := os.Getenv("MYSQL_PASSWORD")
	dbName := os.Getenv("MYSQL_DATABASE")

	// 构建DSN (Data Source Name)
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// 连接MySQL数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Logger.Fatalf("Failed to connect to MySQL database: %v", err)
	}
	log.Logger.Infof("Connected to MySQL database: %s@%s:%s/%s", dbUser, dbHost, dbPort, dbName)

	// 初始化数据库模型和默认数据
	if err := InitDB(db); err != nil {
		log.Logger.Fatalf("Failed to initialize database: %v", err)
	}

	service := &HTTPService{
		conf:        conf,
		db:          db,
		apiKeyCache: make(map[string]*models.APIKey),
	}

	return service
}

func (this *HTTPService) RedirectSwagger(writer http.ResponseWriter, request *http.Request) {
	http.Redirect(writer, request, "/swagger/", 301)
}

func (this *HTTPService) NotFoundHandle(writer http.ResponseWriter, request *http.Request) {
	server_error := &APIStandardError{Type: "error", Error: &APIError{
		Type:    "error",
		Message: "not found",
	}}
	json_str, _ := json.Marshal(server_error)
	http.Error(writer, string(json_str), 404)
}

func (this *HTTPService) ResponseError(err error, writer http.ResponseWriter) {
	server_error := &APIStandardError{Type: "error", Error: &APIError{
		Type:    "invalid_request_error",
		Message: err.Error(),
	}}
	json_str, _ := json.Marshal(server_error)
	http.Error(writer, string(json_str), 200)
}

func (this *HTTPService) ResponseJSON(source interface{}, writer http.ResponseWriter) {
	encoder := json.NewEncoder(writer)
	encoder.SetEscapeHTML(false)

	writer.Header().Set("Content-Type", "application/json")
	err := encoder.Encode(source)
	if err != nil {
		this.ResponseError(err, writer)
	}
}

func (this *HTTPService) ResponseSSE(writer http.ResponseWriter, queue <-chan ISSEDecoder) {
	// output & flush SSE
	flusher, ok := writer.(http.Flusher)
	if !ok {
		this.ResponseError(fmt.Errorf("streaming not supported"), writer)
		return
	}
	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	for event := range queue {
		raw := NewSSERaw(event)
		_, err := writer.Write(raw)
		if err != nil {
			log.Logger.Error(err)
			continue
		}
		flusher.Flush()
	}
}

func (this *HTTPService) HandleComplete(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		this.ResponseError(fmt.Errorf("method not allowed"), writer)
		return
	}
	if request.Header.Get("Content-Type") != "application/json" {
		this.ResponseError(fmt.Errorf("invalid content type"), writer)
		return
	}
	defer request.Body.Close()
	// json decode request body
	var req *ClaudeTextCompletionRequest
	err := json.NewDecoder(request.Body).Decode(&req)
	if err != nil {
		this.ResponseError(err, writer)
		return
	}
	// get anthropic-version,x-api-key from request
	//anthropicVersion := request.Header.Get("anthropic-version")
	//anthropicKey := request.Header.Get("x-api-key")

	bedrockClient := NewBedrockClient(this.conf.BedrockConfig)
	response, err := bedrockClient.CompleteText(req)
	if err != nil {
		this.ResponseError(err, writer)
		return
	}

	if response.IsStream() {
		// output & flush SSE
		flusher, ok := writer.(http.Flusher)
		if !ok {
			this.ResponseError(fmt.Errorf("streaming not supported"), writer)
			return
		}
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Header().Set("Connection", "keep-alive")

		for event := range response.GetEvents() {
			_, err = writer.Write(NewSSERaw(event))
			if err != nil {
				log.Logger.Error(err)
				continue
			}
			flusher.Flush()
		}
		return
	}

	this.ResponseJSON(response.GetResponse(), writer)
}

func (this *HTTPService) HandleMessageComplete(writer http.ResponseWriter, request *http.Request) {
	if request.Method != "POST" {
		log.Logger.Errorf("Method not allowed: %s", request.Method)
		this.ResponseError(fmt.Errorf("method not allowed"), writer)
		return
	}
	if request.Header.Get("Content-Type") != "application/json" {
		log.Logger.Errorf("Invalid content type: %s", request.Header.Get("Content-Type"))
		this.ResponseError(fmt.Errorf("invalid content type"), writer)
		return
	}
	// 读取请求 body
	body, err := io.ReadAll(request.Body)
	if err != nil {
		log.Logger.Error(err)
		this.ResponseError(fmt.Errorf("Error reading request body"), writer)
		return
	}
	defer request.Body.Close()

	// json decode request body
	var req ClaudeMessageCompletionRequest
	err = json.Unmarshal(body, &req)
	if err != nil {
		log.Logger.Error(err)
		this.ResponseError(err, writer)
		return
	}
	// fmt.Printf("Request: %+v", req)
	// get anthropic-version,x-api-key from request
	anthropicVersion := request.Header.Get("anthropic-version")
	if len(anthropicVersion) > 0 {
		req.AnthropicVersion = anthropicVersion
	}
	//anthropicKey := request.Header.Get("x-api-key")

	// log.Logger.Debug(string(body))
	for _, msg := range req.Messages {
		log.Logger.Debugf("%+v", msg)
	}

	bedrockClient := NewBedrockClient(this.conf.BedrockConfig)
	response, err := bedrockClient.MessageCompletion(&req)
	if err != nil {
		this.ResponseError(err, writer)
		return
	}

	if response.IsStream() {
		// 创建一个新的通道来拦截事件
		eventQueue := make(chan ISSEDecoder, 10)
		go func() {
			defer close(eventQueue)

			// var lastEvent *ClaudeMessageCompletionStreamEvent
			var inputTokens, outputTokens int
			var usageRecorded bool = false

			// 从原始通道读取事件
			for event := range response.GetEvents() {
				// 传递给新通道
				eventQueue <- event

				// 尝试将事件转换为特定类型以检查 usage 信息
				if streamEvent, ok := event.(*ClaudeMessageCompletionStreamEvent); ok {
					// lastEvent = streamEvent
					eventType := streamEvent.GetEvent()

					// 收集输入和输出token信息
					if eventType == "message_start" && streamEvent.Message != nil && streamEvent.Message.Usage != nil {
						inputTokens = streamEvent.Message.Usage.InputTokens
						log.Logger.Infof("Stream Usage - Input Tokens: %d", inputTokens)
					} else if eventType == "message_delta" && streamEvent.Usage != nil {
						if streamEvent.Usage.OutputTokens > outputTokens {
							outputTokens = streamEvent.Usage.OutputTokens
						}
						log.Logger.Infof("Stream Usage - Output Tokens: %d", outputTokens)
					} else if (eventType == "message_stop" || eventType == "content_block_stop") && !usageRecorded &&
						inputTokens > 0 && outputTokens > 0 {
						// 记录API使用情况
						apiKeyValue := request.Header.Get("x-api-key")
						apiKeyName := "default"

						// 查询API密钥名称 - 使用缓存
						if apiKeyValue != "" {
							if apiKey, err := this.getAPIKeyFromCache(apiKeyValue); err == nil {
								apiKeyName = apiKey.Name
							}
						}

						// 记录使用情况
						quota := int((float64(inputTokens) * ModelMetaMap[req.Model].ModelRatio + float64(outputTokens)*ModelMetaMap[req.Model].CompletionRatio))
						if err := models.CreateUsage(this.db, apiKeyName, apiKeyValue, req.Model,
							inputTokens, outputTokens, quota); err != nil {
							log.Logger.Errorf("Failed to log API usage: %v", err)
						} else {
							usageRecorded = true
							log.Logger.Infof("API usage recorded - Input: %d, Output: %d, Quota: %d", inputTokens, outputTokens, quota)
						}
					}
				}
			}
		}()

		// 使用拦截后的通道进行响应
		this.ResponseSSE(writer, eventQueue)
		return
	}

	// 打印出 usage 信息
	if resp, ok := response.GetResponse().(*ClaudeMessageCompletionResponse); ok && resp.Usage != nil {
		log.Logger.Infof("Usage - Input Tokens: %d, Output Tokens: %d",
			resp.Usage.InputTokens, resp.Usage.OutputTokens)

		// 记录API使用情况
		apiKeyValue := request.Header.Get("x-api-key")
		apiKeyName := "default"

		// 查询API密钥名称 - 使用缓存
		if apiKeyValue != "" {
			if apiKey, err := this.getAPIKeyFromCache(apiKeyValue); err == nil {
				apiKeyName = apiKey.Name
			}
		}

		// 记录使用情况
		quota := int((float64(resp.Usage.InputTokens) * ModelMetaMap[resp.Model].ModelRatio + float64(resp.Usage.OutputTokens)*ModelMetaMap[resp.Model].CompletionRatio))
		if err := models.CreateUsage(this.db, apiKeyName, apiKeyValue, resp.Model,
			resp.Usage.InputTokens, resp.Usage.OutputTokens, quota); err != nil {
			log.Logger.Errorf("Failed to log API usage: %v", err)
		}
	}

	this.ResponseJSON(response.GetResponse(), writer)
}

// APIKeyMiddleware 验证 API Key 的中间件
func (this *HTTPService) APIKeyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		log.Logger.Infof("Request URL Path: %s", request.URL.Path)

		apiKeyValue := request.Header.Get("x-api-key")
		if apiKeyValue == "" {
			this.ResponseError(fmt.Errorf("invalid api key"), writer)
			return
		}

		// 使用缓存检查API Key
		apiKey, err := this.getAPIKeyFromCache(apiKeyValue)
		if err != nil {
			this.ResponseError(fmt.Errorf("invalid api key"), writer)
			return
		}

		// 这里可以添加更多的 API Key 验证逻辑
		if apiKey.Value != apiKeyValue {
			this.ResponseError(fmt.Errorf("invalid api key"), writer)
			return
		}

		next.ServeHTTP(writer, request)
	})
}

// getAPIKeyFromCache 从缓存中获取API Key，如果不存在则从数据库中获取并缓存
func (this *HTTPService) getAPIKeyFromCache(value string) (*models.APIKey, error) {
	// 尝试从缓存中读取
	this.cacheMutex.RLock()
	cachedKey, exists := this.apiKeyCache[value]
	this.cacheMutex.RUnlock()

	// 如果存在，直接返回
	if exists {
		return cachedKey, nil
	}

	// 缓存不存在，从数据库获取
	apiKey, err := models.GetAPIKey(this.db, value)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	this.cacheMutex.Lock()
	this.apiKeyCache[value] = &apiKey
	this.cacheMutex.Unlock()

	return &apiKey, nil
}

// 从缓存中移除API Key
func (this *HTTPService) refreshAPIKeyCache(value string) {
	this.cacheMutex.Lock()
	defer this.cacheMutex.Unlock()

	// 从缓存中删除该API Key，下次请求时会重新从数据库加载
	delete(this.apiKeyCache, value)
}

func (this *HTTPService) HandleAdminLogin(w http.ResponseWriter, r *http.Request) {
	handler := api.AdminLogin(this.db)
	handler(w, r)
}

func (this *HTTPService) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	handler := api.CreateAPIKey(this.db)
	handler(w, r)
}

func (this *HTTPService) DeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// 先获取要删除的API Key值
	var apiKey models.APIKey
	if err := this.db.First(&apiKey, id).Error; err == nil {
		// 记住API Key值
		keyValue := apiKey.Value

		// 执行删除
		handler := api.DeleteAPIKey(this.db)
		handler(w, r)

		// 从缓存中移除
		this.refreshAPIKeyCache(keyValue)
	} else {
		// 如果找不到API Key，仍然执行原始处理程序
		handler := api.DeleteAPIKey(this.db)
		handler(w, r)
	}
}

func (this *HTTPService) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	handler := api.ListAPIKeys(this.db)
	handler(w, r)
}

func (this *HTTPService) ListUsage(w http.ResponseWriter, r *http.Request) {
	handler := api.ListUsage(this.db)
	handler(w, r)
}

func (this *HTTPService) AdminMiddleware(next http.Handler) http.Handler {
	return api.AdminMiddleware(this.db)(next)
}

func (this *HTTPService) Start() {
	rHandler := mux.NewRouter()

	// 管理员登录
	mainRouter := rHandler.PathPrefix("/").Subrouter()
	mainRouter.HandleFunc("/login/admin", this.HandleAdminLogin)

	// 需要管理员权限的路由
	adminRouter := rHandler.PathPrefix("/admin").Subrouter()
	adminRouter.Use(this.AdminMiddleware)
	adminRouter.HandleFunc("/apikey/create", this.CreateAPIKey)
	adminRouter.HandleFunc("/apikey/{id}/delete", this.DeleteAPIKey)
	adminRouter.HandleFunc("/apikey/list", this.ListAPIKeys)
	adminRouter.HandleFunc("/usage/list", this.ListUsage)

	// 需要 API Key 的路由
	apiRouter := rHandler.PathPrefix("/v1").Subrouter()
	apiRouter.Use(this.APIKeyMiddleware)

	apiRouter.HandleFunc("/complete", this.HandleComplete)
	apiRouter.HandleFunc("/messages", this.HandleMessageComplete)

	rHandler.HandleFunc("/", this.RedirectSwagger)
	rHandler.PathPrefix("/").Handler(http.StripPrefix("/",
		http.FileServer(http.Dir(fmt.Sprintf("%s", this.conf.WebRoot)))))
	rHandler.NotFoundHandler = http.HandlerFunc(this.NotFoundHandle)

	log.Logger.Info("http service starting")
	log.Logger.Infof("Please open http://%s\n", this.conf.Listen)
	err := http.ListenAndServe(this.conf.Listen, rHandler)
	if err != nil {
		log.Logger.Error(err)
	}
}
