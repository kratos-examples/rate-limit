// Package server provides HTTP and gRPC with Redis-backed rate limiting middleware
//
// Package server 提供带 Redis 限流中间件的 HTTP 和 gRPC 服务
package server

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/go-redis/redis_rate/v10"
	"github.com/yylego/kratos-auth/authkratos"
	pb "github.com/yylego/kratos-examples/demo1kratos/api/student"
	"github.com/yylego/kratos-examples/demo1kratos/internal/conf"
	"github.com/yylego/kratos-examples/demo1kratos/internal/data"
	"github.com/yylego/kratos-examples/demo1kratos/internal/service"
	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
)

func NewHTTPServer(
	c *conf.Server,
	dataData *data.Data,
	student *service.StudentService,
	logger log.Logger,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			NewRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
		),
	}
	if c.Http.Network != "" {
		opts = append(opts, http.Network(c.Http.Network))
	}
	if c.Http.Address != "" {
		opts = append(opts, http.Address(c.Http.Address))
	}
	if c.Http.Timeout != nil {
		opts = append(opts, http.Timeout(c.Http.Timeout.AsDuration()))
	}
	srv := http.NewServer(opts...)
	pb.RegisterStudentServiceHTTPServer(srv, student)
	return srv
}

// NewRateLimitMiddleware creates rate limiting middleware using Redis-backed limiter
// Applies rate limit to all student service operations with per-operation key extraction
//
// NewRateLimitMiddleware 创建基于 Redis 的限流中间件
// 对所有学生服务操作应用限流，按操作名称提取限流键
func NewRateLimitMiddleware(dataData *data.Data, logger log.Logger) middleware.Middleware {
	routeScope := authkratos.NewInclude( // Create INCLUDE mode route scope // 创建 INCLUDE 模式的路由范围
		pb.OperationStudentServiceCreateStudent,
		pb.OperationStudentServiceUpdateStudent,
		pb.OperationStudentServiceDeleteStudent,
		pb.OperationStudentServiceGetStudent,
		pb.OperationStudentServiceListStudents,
	)
	redisLimit := redis_rate.Limit{
		Rate:   10,
		Burst:  20,
		Period: time.Minute,
	}
	keyFromCtx := func(ctx context.Context) (string, bool) {
		// Use a fixed demo key for rate limiting demonstration
		// In production, extract user ID or IP from context
		// 使用固定的演示键进行限流演示，生产环境应从上下文提取用户 ID 或 IP
		return "demo-rate-limit-key", true
	}
	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), &redisLimit, keyFromCtx).
		WithDebugMode(true) // Enable debug mode to log rate limit process // 启用调试模式记录限流过程
	return ratekratoslimits.NewMiddleware(cfg, logger)
}
