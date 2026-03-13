package data

import (
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/wire"
	"github.com/yylego/kratos-examples/demo2kratos/internal/conf"
	"github.com/yylego/must"
	"github.com/yylego/rese"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var ProviderSet = wire.NewSet(NewData)

type Data struct {
	db *gorm.DB
}

func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	must.Same(c.Database.Driver, "sqlite3")
	db := rese.P1(gorm.Open(sqlite.Open(c.Database.Source), &gorm.Config{}))
	cleanup := func() {
		log.NewHelper(logger).Info("closing the data resources")
		_ = rese.P1(db.DB()).Close()
	}
	return &Data{db: db}, cleanup, nil
}
