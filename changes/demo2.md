# Changes

Code differences compared to source project.

## cmd/demo2kratos/wire_gen.go (+2 -2)

```diff
@@ -34,8 +34,8 @@
 		return nil, nil, err
 	}
 	articleService := service.NewArticleService(articleUsecase)
-	grpcServer := server.NewGRPCServer(confServer, articleService, logger)
-	httpServer := server.NewHTTPServer(confServer, articleService, logger)
+	grpcServer := server.NewGRPCServer(confServer, dataData, articleService, logger)
+	httpServer := server.NewHTTPServer(confServer, dataData, articleService, logger)
 	app := newApp(logger, grpcServer, httpServer)
 	return app, func() {
 		cleanup()
```

## internal/data/data.go (+24 -8)

```diff
@@ -3,33 +3,49 @@
 import (
 	"log/slog"
 
+	"github.com/alicebob/miniredis/v2"
+	"github.com/go-redis/redis_rate/v10"
 	"github.com/google/wire"
+	"github.com/redis/go-redis/v9"
 	"github.com/yylego/kratos-examples/demo2kratos/internal/conf"
 	"github.com/yylego/must"
 	"github.com/yylego/rese"
-	"gorm.io/driver/postgres"
+	"gorm.io/driver/sqlite"
 	"gorm.io/gorm"
 )
 
 var ProviderSet = wire.NewSet(NewData)
 
 type Data struct {
-	db *gorm.DB
+	db          *gorm.DB
+	rateLimiter *redis_rate.Limiter
 }
 
-// DB exposes the underlying gorm handle so the biz code can run true queries.
-//
-// DB 暴露底层 gorm 句柄，供 biz 层执行真实的数据库读写
 func (d *Data) DB() *gorm.DB {
 	return d.db
 }
 
+func (d *Data) RateLimiter() *redis_rate.Limiter {
+	return d.rateLimiter
+}
+
 func NewData(c *conf.Data, logger *slog.Logger) (*Data, func(), error) {
-	must.Same(c.Database.Driver, "postgres")
-	db := rese.P1(gorm.Open(postgres.Open(c.Database.Source), &gorm.Config{}))
+	must.Same(c.Database.Driver, "sqlite3")
+	db := rese.P1(gorm.Open(sqlite.Open(c.Database.Source), &gorm.Config{}))
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

## internal/server/grpc.go (+38 -1)

```diff
@@ -1,19 +1,32 @@
 package server
 
 import (
+	"context"
 	"log/slog"
+	"time"
 
+	"github.com/go-kratos/kratos/v3/middleware"
 	"github.com/go-kratos/kratos/v3/middleware/recovery"
 	"github.com/go-kratos/kratos/v3/transport/grpc"
+	"github.com/go-redis/redis_rate/v10"
+	"github.com/yylego/kratos-auth/authkratos"
 	pb "github.com/yylego/kratos-examples/demo2kratos/api/article"
 	"github.com/yylego/kratos-examples/demo2kratos/internal/conf"
+	"github.com/yylego/kratos-examples/demo2kratos/internal/data"
 	"github.com/yylego/kratos-examples/demo2kratos/internal/service"
+	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
 )
 
-func NewGRPCServer(c *conf.Server, article *service.ArticleService, logger *slog.Logger) *grpc.Server {
+func NewGRPCServer(
+	c *conf.Server,
+	dataData *data.Data,
+	article *service.ArticleService,
+	logger *slog.Logger,
+) *grpc.Server {
 	var opts = []grpc.ServerOption{
 		grpc.Middleware(
 			recovery.Recovery(),
+			NewGRPCRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
 		),
 	}
 	if c.Grpc.Network != "" {
@@ -28,4 +41,28 @@
 	srv := grpc.NewServer(opts...)
 	pb.RegisterArticleServiceServer(srv, article)
 	return srv
+}
+
+// NewGRPCRateLimitMiddleware creates rate limiting middleware for gRPC transport
+//
+// NewGRPCRateLimitMiddleware 创建 gRPC 传输层的限流中间件
+func NewGRPCRateLimitMiddleware(dataData *data.Data, logger *slog.Logger) middleware.Middleware {
+	routeScope := authkratos.NewInclude(
+		pb.OperationArticleServiceCreateArticle,
+		pb.OperationArticleServiceUpdateArticle,
+		pb.OperationArticleServiceDeleteArticle,
+		pb.OperationArticleServiceGetArticle,
+		pb.OperationArticleServiceListArticles,
+	)
+	redisLimit := redis_rate.Limit{
+		Rate:   10,
+		Burst:  20,
+		Period: time.Minute,
+	}
+	keyFromCtx := func(ctx context.Context) (string, bool) {
+		return "demo-rate-limit-key", true
+	}
+	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), &redisLimit, keyFromCtx).
+		WithDebugMode(true)
+	return ratekratoslimits.NewMiddleware(cfg, logger)
 }
```

## internal/server/http.go (+46 -1)

```diff
@@ -1,19 +1,35 @@
+// Package server provides HTTP and gRPC with Redis-backed rate limiting middleware
+//
+// Package server 提供带 Redis 限流中间件的 HTTP 和 gRPC 服务
 package server
 
 import (
+	"context"
 	"log/slog"
+	"time"
 
+	"github.com/go-kratos/kratos/v3/middleware"
 	"github.com/go-kratos/kratos/v3/middleware/recovery"
 	"github.com/go-kratos/kratos/v3/transport/http"
+	"github.com/go-redis/redis_rate/v10"
+	"github.com/yylego/kratos-auth/authkratos"
 	pb "github.com/yylego/kratos-examples/demo2kratos/api/article"
 	"github.com/yylego/kratos-examples/demo2kratos/internal/conf"
+	"github.com/yylego/kratos-examples/demo2kratos/internal/data"
 	"github.com/yylego/kratos-examples/demo2kratos/internal/service"
+	"github.com/yylego/kratos-rate-limit/ratekratoslimits"
 )
 
-func NewHTTPServer(c *conf.Server, article *service.ArticleService, logger *slog.Logger) *http.Server {
+func NewHTTPServer(
+	c *conf.Server,
+	dataData *data.Data,
+	article *service.ArticleService,
+	logger *slog.Logger,
+) *http.Server {
 	var opts = []http.ServerOption{
 		http.Middleware(
 			recovery.Recovery(),
+			NewRateLimitMiddleware(dataData, logger), // Redis-backed rate limiting // 基于 Redis 的限流中间件
 		),
 	}
 	if c.Http.Network != "" {
@@ -28,4 +44,33 @@
 	srv := http.NewServer(opts...)
 	pb.RegisterArticleServiceHTTPServer(srv, article)
 	return srv
+}
+
+// NewRateLimitMiddleware creates rate limiting middleware using Redis-backed limiter
+// Applies rate limit to all article service operations with per-operation key extraction
+//
+// NewRateLimitMiddleware 创建基于 Redis 的限流中间件
+// 对所有文章服务操作应用限流，按操作名称提取限流键
+func NewRateLimitMiddleware(dataData *data.Data, logger *slog.Logger) middleware.Middleware {
+	routeScope := authkratos.NewInclude( // Create INCLUDE mode route scope // 创建 INCLUDE 模式的路由范围
+		pb.OperationArticleServiceCreateArticle,
+		pb.OperationArticleServiceUpdateArticle,
+		pb.OperationArticleServiceDeleteArticle,
+		pb.OperationArticleServiceGetArticle,
+		pb.OperationArticleServiceListArticles,
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
+	cfg := ratekratoslimits.NewConfig(routeScope, dataData.RateLimiter(), &redisLimit, keyFromCtx).
+		WithDebugMode(true) // Enable debug mode to log rate limit process // 启用调试模式记录限流过程
+	return ratekratoslimits.NewMiddleware(cfg, logger)
 }
```

