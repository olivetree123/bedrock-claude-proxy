package api

import (
	log "bedrock-claude-proxy/log"
	"bedrock-claude-proxy/models"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

var (
	jwtKey = []byte("bedrock_claude_proxy_secret_key")
)

// 登录请求结构
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// 登录响应结构
type LoginResponse struct {
	Token string `json:"token"`
}

// Claims 结构包含JWT的标准声明和自定义声明
type Claims struct {
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// 哈希密码
func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return hex.EncodeToString(hash[:])
}

// 登录处理函数
func AdminLogin(db *gorm.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 只接受POST请求
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 解析请求体
		var req LoginRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// 查询数据库验证用户名和密码
		var admin models.Admin
		result := db.Where("username = ?", req.Username).First(&admin)
		if result.Error != nil {
			log.Logger.Errorf("Failed to find admin: %v", result.Error)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// 验证密码
		hashedPassword := hashPassword(req.Password)
		if hashedPassword != admin.Password {
			log.Logger.Warningf("Invalid password attempt for user: %s", req.Username)
			http.Error(w, "Invalid credentials", http.StatusUnauthorized)
			return
		}

		// 创建JWT Token
		expirationTime := time.Now().Add(24 * time.Hour)
		claims := &Claims{
			Username: req.Username,
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(expirationTime),
				IssuedAt:  jwt.NewNumericDate(time.Now()),
				NotBefore: jwt.NewNumericDate(time.Now()),
				Issuer:    "bedrock-claude-proxy",
				Subject:   req.Username,
			},
		}

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
		tokenString, err := token.SignedString(jwtKey)
		if err != nil {
			log.Logger.Errorf("Failed to generate token: %v", err)
			http.Error(w, "Failed to generate token", http.StatusInternalServerError)
			return
		}

		// 返回token
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{Token: tokenString})
		log.Logger.Infof("Admin login successful: %s", req.Username)
	}
}

// 验证JWT Token并获取用户名
func GetUsernameFromToken(tokenString string) (string, error) {
	// 解析token
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// 验证算法
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtKey, nil
	})

	if err != nil {
		return "", err
	}

	if !token.Valid {
		return "", fmt.Errorf("invalid token")
	}

	// 返回用户名
	return claims.Username, nil
}

// 验证管理员权限的中间件
func AdminMiddleware(db *gorm.DB) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 从Authorization头获取token
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, "Authorization header required", http.StatusUnauthorized)
				return
			}

			// 检查Bearer前缀
			if len(authHeader) < 7 || authHeader[:7] != "Bearer " {
				http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
				return
			}

			// 提取token
			tokenString := authHeader[7:]
			username, err := GetUsernameFromToken(tokenString)
			if err != nil {
				log.Logger.Errorf("Invalid token: %v", err)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// 验证该管理员是否存在
			var admin models.Admin
			result := db.Where("username = ?", username).First(&admin)
			if result.Error != nil {
				log.Logger.Errorf("Admin not found: %v", result.Error)
				http.Error(w, "Invalid token", http.StatusUnauthorized)
				return
			}

			// 请求中存储用户名，以便后续处理函数使用
			r = r.WithContext(SetUsername(r.Context(), username))

			// 继续执行下一个处理函数
			next.ServeHTTP(w, r)
		})
	}
}
