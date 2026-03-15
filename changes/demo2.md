# Changes

Code differences compared to source project.

## internal/biz/article.go (+10 -0)

```diff
@@ -8,6 +8,7 @@
 	"github.com/yylego/kratos-ebz/ebzkratos"
 	pb "github.com/yylego/kratos-examples/demo2kratos/api/article"
 	"github.com/yylego/kratos-examples/demo2kratos/internal/data"
+	"github.com/yylego/must"
 )
 
 type Article struct {
@@ -27,6 +28,8 @@
 }
 
 func (uc *ArticleUsecase) CreateArticle(ctx context.Context, a *Article) (*Article, *ebzkratos.Ebz) {
+	must.Nice(a.Title)
+
 	var res Article
 	if err := gofakeit.Struct(&res); err != nil {
 		return nil, ebzkratos.New(pb.ErrorArticleCreateFailure("fake: %v", err))
@@ -35,6 +38,9 @@
 }
 
 func (uc *ArticleUsecase) UpdateArticle(ctx context.Context, a *Article) (*Article, *ebzkratos.Ebz) {
+	must.True(a.ID > 0)
+	must.Nice(a.Title)
+
 	var res Article
 	if err := gofakeit.Struct(&res); err != nil {
 		return nil, ebzkratos.New(pb.ErrorServerError("fake: %v", err))
@@ -43,10 +49,14 @@
 }
 
 func (uc *ArticleUsecase) DeleteArticle(ctx context.Context, id int64) *ebzkratos.Ebz {
+	must.True(id > 0)
+
 	return nil
 }
 
 func (uc *ArticleUsecase) GetArticle(ctx context.Context, id int64) (*Article, *ebzkratos.Ebz) {
+	must.True(id > 0)
+
 	var res Article
 	if err := gofakeit.Struct(&res); err != nil {
 		return nil, ebzkratos.New(pb.ErrorServerError("fake: %v", err))
```

## internal/service/article.go (+26 -2)

```diff
@@ -18,7 +18,14 @@
 }
 
 func (s *ArticleService) CreateArticle(ctx context.Context, req *pb.CreateArticleRequest) (*pb.CreateArticleReply, error) {
-	v, ebz := s.uc.CreateArticle(ctx, nil)
+	if req.Title == "" {
+		return nil, pb.ErrorBadParam("TITLE IS REQUIRED")
+	}
+	v, ebz := s.uc.CreateArticle(ctx, &biz.Article{
+		Title:     req.Title,
+		Content:   req.Content,
+		StudentID: req.StudentId,
+	})
 	if ebz != nil {
 		return nil, ebz.Erk
 	}
@@ -26,7 +33,18 @@
 }
 
 func (s *ArticleService) UpdateArticle(ctx context.Context, req *pb.UpdateArticleRequest) (*pb.UpdateArticleReply, error) {
-	v, ebz := s.uc.UpdateArticle(ctx, nil)
+	if req.Id <= 0 {
+		return nil, pb.ErrorBadParam("ID IS REQUIRED")
+	}
+	if req.Title == "" {
+		return nil, pb.ErrorBadParam("TITLE IS REQUIRED")
+	}
+	v, ebz := s.uc.UpdateArticle(ctx, &biz.Article{
+		ID:        req.Id,
+		Title:     req.Title,
+		Content:   req.Content,
+		StudentID: req.StudentId,
+	})
 	if ebz != nil {
 		return nil, ebz.Erk
 	}
@@ -34,6 +52,9 @@
 }
 
 func (s *ArticleService) DeleteArticle(ctx context.Context, req *pb.DeleteArticleRequest) (*pb.DeleteArticleReply, error) {
+	if req.Id <= 0 {
+		return nil, pb.ErrorBadParam("ID IS REQUIRED")
+	}
 	if ebz := s.uc.DeleteArticle(ctx, req.Id); ebz != nil {
 		return nil, ebz.Erk
 	}
@@ -41,6 +62,9 @@
 }
 
 func (s *ArticleService) GetArticle(ctx context.Context, req *pb.GetArticleRequest) (*pb.GetArticleReply, error) {
+	if req.Id <= 0 {
+		return nil, pb.ErrorBadParam("ID IS REQUIRED")
+	}
 	v, ebz := s.uc.GetArticle(ctx, req.Id)
 	if ebz != nil {
 		return nil, ebz.Erk
```

