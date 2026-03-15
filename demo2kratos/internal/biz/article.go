package biz

import (
	"context"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/yylego/kratos-ebz/ebzkratos"
	pb "github.com/yylego/kratos-examples/demo2kratos/api/article"
	"github.com/yylego/kratos-examples/demo2kratos/internal/data"
	"github.com/yylego/must"
)

type Article struct {
	ID        int64
	Title     string
	Content   string
	StudentID int64
}

type ArticleUsecase struct {
	data *data.Data
	log  *log.Helper
}

func NewArticleUsecase(data *data.Data, logger log.Logger) *ArticleUsecase {
	return &ArticleUsecase{data: data, log: log.NewHelper(logger)}
}

func (uc *ArticleUsecase) CreateArticle(ctx context.Context, a *Article) (*Article, *ebzkratos.Ebz) {
	must.Nice(a.Title)

	var res Article
	if err := gofakeit.Struct(&res); err != nil {
		return nil, ebzkratos.New(pb.ErrorArticleCreateFailure("fake: %v", err))
	}
	return &res, nil
}

func (uc *ArticleUsecase) UpdateArticle(ctx context.Context, a *Article) (*Article, *ebzkratos.Ebz) {
	must.True(a.ID > 0)
	must.Nice(a.Title)

	var res Article
	if err := gofakeit.Struct(&res); err != nil {
		return nil, ebzkratos.New(pb.ErrorServerError("fake: %v", err))
	}
	return &res, nil
}

func (uc *ArticleUsecase) DeleteArticle(ctx context.Context, id int64) *ebzkratos.Ebz {
	must.True(id > 0)

	return nil
}

func (uc *ArticleUsecase) GetArticle(ctx context.Context, id int64) (*Article, *ebzkratos.Ebz) {
	must.True(id > 0)

	var res Article
	if err := gofakeit.Struct(&res); err != nil {
		return nil, ebzkratos.New(pb.ErrorServerError("fake: %v", err))
	}
	return &res, nil
}

func (uc *ArticleUsecase) ListArticles(ctx context.Context, page int32, pageSize int32) ([]*Article, int32, *ebzkratos.Ebz) {
	var items []*Article
	gofakeit.Slice(&items)
	return items, int32(len(items)), nil
}
