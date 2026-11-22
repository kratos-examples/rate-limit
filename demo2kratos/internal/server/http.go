// Package server provides HTTP and gRPC with Redis-backed rate limiting middleware
//
// Package server 提供带 Redis 限流中间件的 HTTP 和 gRPC 服务
package server

import (
	"context"
	"log/slog"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/recovery"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"github.com/go-kratos/kratos/v3/transport/http"
	"github.com/go-redis/redis_rate/v10"
	"github.com/yylego/kratos-auth/authkratos"
	pb "github.com/yylego/kratos-examples/demo2kratos/api/article"
	"github.com/yylego/kratos-examples/demo2kratos/internal/conf"
	"github.com/yylego/kratos-examples/demo2kratos/internal/data"
	"github.com/yylego/kratos-examples/demo2kratos/internal/service"
	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
)

func NewHTTPServer(
	c *conf.Server,
	dataData *data.Data,
	article *service.ArticleService,
	logger *slog.Logger,
) *http.Server {
	var opts = []http.ServerOption{
		http.Middleware(
			recovery.Recovery(),
			NewRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
			validate.Validator(validateRequiredFields),
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
	pb.RegisterArticleServiceHTTPServer(srv, article)
	return srv
}

// NewRateLimitMiddleware creates rate limiting middleware using Redis-backed limiter
// Article CRUD operations share one quota with gRPC; ListStudentArticles is outside the scope
//
// NewRateLimitMiddleware 创建基于 Redis 的限流中间件
// 文章 CRUD 操作与 gRPC 共用一份额度；ListStudentArticles 不在限流范围内
func NewRateLimitMiddleware(dataData *data.Data, logger *slog.Logger) middleware.Middleware {
	routeScope := authkratos.NewInclude( // Create INCLUDE mode route scope // 创建 INCLUDE 模式的路由范围
		pb.OperationArticleServiceCreateArticle,
		pb.OperationArticleServiceUpdateArticle,
		pb.OperationArticleServiceDeleteArticle,
		pb.OperationArticleServiceGetArticle,
		pb.OperationArticleServiceListArticles,
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
	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), redisLimit, keyFromCtx).
		WithExceedErk(func(ctx context.Context) *errors.Error {
			return errors.New(429, "TOO_MANY_REQUESTS", "slow down")
		})
	return ratekratoslimits.NewMiddleware(cfg, logger)
}
