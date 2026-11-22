# Changes

Code differences compared to source project.

## cmd/demo1kratos/wire_gen.go (+2 -2)

```diff
@@ -30,8 +30,8 @@
 		return nil, nil, err
 	}
 	studentService := service.NewStudentService(studentUsecase)
-	grpcServer := server.NewGRPCServer(confServer, studentService, logger)
-	httpServer := server.NewHTTPServer(confServer, studentService, logger)
+	grpcServer := server.NewGRPCServer(confServer, dataData, studentService, logger)
+	httpServer := server.NewHTTPServer(confServer, dataData, studentService, logger)
 	app := newApp(logger, grpcServer, httpServer)
 	return app, func() {
 		cleanup()
```

## internal/data/data.go (+21 -2)

```diff
@@ -3,7 +3,10 @@
 import (
 	"log/slog"
 
+	"github.com/alicebob/miniredis/v2"
+	"github.com/go-redis/redis_rate/v10"
 	"github.com/google/wire"
+	"github.com/redis/go-redis/v9"
 	"github.com/yylego/kratos-examples/demo1kratos/internal/conf"
 	"github.com/yylego/must"
 	"github.com/yylego/rese"
@@ -14,7 +17,8 @@
 var ProviderSet = wire.NewSet(NewData)
 
 type Data struct {
-	db *gorm.DB
+	db          *gorm.DB
+	rateLimiter *redis_rate.Limiter
 }
 
 // DB exposes the underlying gorm handle so the biz code can run true queries.
@@ -24,12 +28,27 @@
 	return d.db
 }
 
+func (d *Data) RateLimiter() *redis_rate.Limiter {
+	return d.rateLimiter
+}
+
 func NewData(c *conf.Data, logger *slog.Logger) (*Data, func(), error) {
 	must.Same(c.Database.Driver, "postgres")
 	db := rese.P1(gorm.Open(postgres.Open(c.Database.Source), &gorm.Config{}))
+
+	// Use miniredis as in-memory Redis for rate limiting demo
+	// 使用 miniredis 作为内存 Redis 用于限流演示
+	miniRedis := rese.P1(miniredis.Run())
+	rdb := redis.NewClient(&redis.Options{
+		Addr: miniRedis.Addr(),
+	})
+	rateLimiter := redis_rate.NewLimiter(rdb)
+
 	cleanup := func() {
 		logger.Info("closing the data resources")
 		_ = rese.P1(db.DB()).Close()
+		_ = rdb.Close()
+		miniRedis.Close()
 	}
-	return &Data{db: db}, cleanup, nil
+	return &Data{db: db, rateLimiter: rateLimiter}, cleanup, nil
 }
```

## internal/server/grpc.go (+41 -1)

```diff
@@ -1,20 +1,34 @@
 package server
 
 import (
+	"context"
 	"log/slog"
+	"time"
 
+	"github.com/go-kratos/kratos/v3/errors"
+	"github.com/go-kratos/kratos/v3/middleware"
 	"github.com/go-kratos/kratos/v3/middleware/recovery"
 	"github.com/go-kratos/kratos/v3/middleware/validate"
 	"github.com/go-kratos/kratos/v3/transport/grpc"
+	"github.com/go-redis/redis_rate/v10"
+	"github.com/yylego/kratos-auth/authkratos"
 	pb "github.com/yylego/kratos-examples/demo1kratos/api/student"
 	"github.com/yylego/kratos-examples/demo1kratos/internal/conf"
+	"github.com/yylego/kratos-examples/demo1kratos/internal/data"
 	"github.com/yylego/kratos-examples/demo1kratos/internal/service"
+	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
 )
 
-func NewGRPCServer(c *conf.Server, student *service.StudentService, logger *slog.Logger) *grpc.Server {
+func NewGRPCServer(
+	c *conf.Server,
+	dataData *data.Data,
+	student *service.StudentService,
+	logger *slog.Logger,
+) *grpc.Server {
 	var opts = []grpc.ServerOption{
 		grpc.Middleware(
 			recovery.Recovery(),
+			NewGRPCRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
 			validate.Validator(validateRequiredFields),
 		),
 	}
@@ -30,4 +44,30 @@
 	srv := grpc.NewServer(opts...)
 	pb.RegisterStudentServiceServer(srv, student)
 	return srv
+}
+
+// NewGRPCRateLimitMiddleware creates rate limiting middleware for gRPC transport
+//
+// NewGRPCRateLimitMiddleware 创建 gRPC 传输层的限流中间件
+func NewGRPCRateLimitMiddleware(dataData *data.Data, logger *slog.Logger) middleware.Middleware {
+	routeScope := authkratos.NewInclude(
+		pb.OperationStudentServiceCreateStudent,
+		pb.OperationStudentServiceUpdateStudent,
+		pb.OperationStudentServiceDeleteStudent,
+		pb.OperationStudentServiceGetStudent,
+		pb.OperationStudentServiceListStudents,
+	)
+	redisLimit := redis_rate.Limit{
+		Rate:   10,
+		Burst:  20,
+		Period: time.Minute,
+	}
+	keyFromCtx := func(ctx context.Context) (string, bool) {
+		return "demo-rate-limit-key", true
+	}
+	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), redisLimit, keyFromCtx).
+		WithExceedErk(func(ctx context.Context) *errors.Error {
+			return errors.New(429, "TOO_MANY_REQUESTS", "slow down")
+		})
+	return ratekratoslimits.NewMiddleware(cfg, logger)
 }
```

## internal/server/http.go (+49 -1)

```diff
@@ -1,20 +1,37 @@
+// Package server provides HTTP and gRPC with Redis-backed rate limiting middleware
+//
+// Package server 提供带 Redis 限流中间件的 HTTP 和 gRPC 服务
 package server
 
 import (
+	"context"
 	"log/slog"
+	"time"
 
+	"github.com/go-kratos/kratos/v3/errors"
+	"github.com/go-kratos/kratos/v3/middleware"
 	"github.com/go-kratos/kratos/v3/middleware/recovery"
 	"github.com/go-kratos/kratos/v3/middleware/validate"
 	"github.com/go-kratos/kratos/v3/transport/http"
+	"github.com/go-redis/redis_rate/v10"
+	"github.com/yylego/kratos-auth/authkratos"
 	pb "github.com/yylego/kratos-examples/demo1kratos/api/student"
 	"github.com/yylego/kratos-examples/demo1kratos/internal/conf"
+	"github.com/yylego/kratos-examples/demo1kratos/internal/data"
 	"github.com/yylego/kratos-examples/demo1kratos/internal/service"
+	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
 )
 
-func NewHTTPServer(c *conf.Server, student *service.StudentService, logger *slog.Logger) *http.Server {
+func NewHTTPServer(
+	c *conf.Server,
+	dataData *data.Data,
+	student *service.StudentService,
+	logger *slog.Logger,
+) *http.Server {
 	var opts = []http.ServerOption{
 		http.Middleware(
 			recovery.Recovery(),
+			NewRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
 			validate.Validator(validateRequiredFields),
 		),
 	}
@@ -30,4 +47,35 @@
 	srv := http.NewServer(opts...)
 	pb.RegisterStudentServiceHTTPServer(srv, student)
 	return srv
+}
+
+// NewRateLimitMiddleware creates rate limiting middleware using Redis-backed limiter
+// Student CRUD operations share one quota with the gRPC transport
+//
+// NewRateLimitMiddleware 创建基于 Redis 的限流中间件
+// 学生 CRUD 操作与 gRPC 传输共用一份限流额度
+func NewRateLimitMiddleware(dataData *data.Data, logger *slog.Logger) middleware.Middleware {
+	routeScope := authkratos.NewInclude( // Create INCLUDE mode route scope // 创建 INCLUDE 模式的路由范围
+		pb.OperationStudentServiceCreateStudent,
+		pb.OperationStudentServiceUpdateStudent,
+		pb.OperationStudentServiceDeleteStudent,
+		pb.OperationStudentServiceGetStudent,
+		pb.OperationStudentServiceListStudents,
+	)
+	redisLimit := redis_rate.Limit{
+		Rate:   10,
+		Burst:  20,
+		Period: time.Minute,
+	}
+	keyFromCtx := func(ctx context.Context) (string, bool) {
+		// Use a fixed demo key for rate limiting demonstration
+		// In production, extract user ID or IP from context
+		// 使用固定的演示键进行限流演示，生产环境应从上下文提取用户 ID 或 IP
+		return "demo-rate-limit-key", true
+	}
+	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), redisLimit, keyFromCtx).
+		WithExceedErk(func(ctx context.Context) *errors.Error {
+			return errors.New(429, "TOO_MANY_REQUESTS", "slow down")
+		})
+	return ratekratoslimits.NewMiddleware(cfg, logger)
 }
```

