package api

import (
	"context"
)

// 上下文键类型
type contextKey string

// 上下文键常量
const (
	usernameKey contextKey = "username"
)

// SetUsername 将用户名存储在上下文中
func SetUsername(ctx context.Context, username string) context.Context {
	return context.WithValue(ctx, usernameKey, username)
}

// GetUsername 从上下文中获取用户名
func GetUsername(ctx context.Context) (string, bool) {
	username, ok := ctx.Value(usernameKey).(string)
	return username, ok
}
