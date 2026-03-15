package server

import (
	"context"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-redis/redis_rate/v10"
	"github.com/yylego/kratos-auth/authkratos"
	pb "github.com/yylego/kratos-examples/demo1kratos/api/student"
	"github.com/yylego/kratos-examples/demo1kratos/internal/conf"
	"github.com/yylego/kratos-examples/demo1kratos/internal/data"
	"github.com/yylego/kratos-examples/demo1kratos/internal/service"
	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
)

func NewGRPCServer(
	c *conf.Server,
	dataData *data.Data,
	student *service.StudentService,
	logger log.Logger,
) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(),
			NewGRPCRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
		),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Address != "" {
		opts = append(opts, grpc.Address(c.Grpc.Address))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	pb.RegisterStudentServiceServer(srv, student)
	return srv
}

// NewGRPCRateLimitMiddleware creates rate limiting middleware for gRPC transport
//
// NewGRPCRateLimitMiddleware 创建 gRPC 传输层的限流中间件
func NewGRPCRateLimitMiddleware(dataData *data.Data, logger log.Logger) middleware.Middleware {
	routeScope := authkratos.NewInclude(
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
		return "demo-rate-limit-key", true
	}
	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), &redisLimit, keyFromCtx).
		WithDebugMode(true)
	return ratekratoslimits.NewMiddleware(cfg, logger)
}
