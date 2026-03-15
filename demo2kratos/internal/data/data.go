package data

import (
	"github.com/alicebob/miniredis/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis_rate/v10"
	"github.com/google/wire"
	"github.com/redis/go-redis/v9"
	"github.com/yylego/kratos-examples/demo2kratos/internal/conf"
	"github.com/yylego/must"
	"github.com/yylego/rese"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData)

type Data struct {
	db          *gorm.DB
	rateLimiter *redis_rate.Limiter
}

func (d *Data) DB() *gorm.DB {
	return d.db
}

func (d *Data) RateLimiter() *redis_rate.Limiter {
	return d.rateLimiter
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	must.Same(c.Database.Driver, "sqlite3")
	db := rese.P1(gorm.Open(sqlite.Open(c.Database.Source), &gorm.Config{}))

	// Use miniredis as in-memory Redis for rate limiting demo
	// 使用 miniredis 作为内存 Redis 用于限流演示
	miniRedis := rese.P1(miniredis.Run())
	rdb := redis.NewClient(&redis.Options{
		Addr: miniRedis.Addr(),
	})
	rateLimiter := redis_rate.NewLimiter(rdb)

	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		_ = rese.P1(db.DB()).Close()
		_ = rdb.Close()
		miniRedis.Close()
	}
	return &Data{db: db, rateLimiter: rateLimiter}, cleanup, nil
}
