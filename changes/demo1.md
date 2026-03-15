# Changes

Code differences compared to source project.

## internal/biz/student.go (+10 -0)

```diff
@@ -8,6 +8,7 @@
 	"github.com/yylego/kratos-ebz/ebzkratos"
 	pb "github.com/yylego/kratos-examples/demo1kratos/api/student"
 	"github.com/yylego/kratos-examples/demo1kratos/internal/data"
+	"github.com/yylego/must"
 )
 
 type Student struct {
@@ -27,6 +28,8 @@
 }
 
 func (uc *StudentUsecase) CreateStudent(ctx context.Context, s *Student) (*Student, *ebzkratos.Ebz) {
+	must.Nice(s.Name)
+
 	var res Student
 	if err := gofakeit.Struct(&res); err != nil {
 		return nil, ebzkratos.New(pb.ErrorStudentCreateFailure("fake: %v", err))
@@ -35,6 +38,9 @@
 }
 
 func (uc *StudentUsecase) UpdateStudent(ctx context.Context, s *Student) (*Student, *ebzkratos.Ebz) {
+	must.True(s.ID > 0)
+	must.Nice(s.Name)
+
 	var res Student
 	if err := gofakeit.Struct(&res); err != nil {
 		return nil, ebzkratos.New(pb.ErrorServerError("fake: %v", err))
@@ -43,10 +49,14 @@
 }
 
 func (uc *StudentUsecase) DeleteStudent(ctx context.Context, id int64) *ebzkratos.Ebz {
+	must.True(id > 0)
+
 	return nil
 }
 
 func (uc *StudentUsecase) GetStudent(ctx context.Context, id int64) (*Student, *ebzkratos.Ebz) {
+	must.True(id > 0)
+
 	var res Student
 	if err := gofakeit.Struct(&res); err != nil {
 		return nil, ebzkratos.New(pb.ErrorServerError("fake: %v", err))
```

## internal/service/student.go (+26 -2)

```diff
@@ -18,7 +18,14 @@
 }
 
 func (s *StudentService) CreateStudent(ctx context.Context, req *pb.CreateStudentRequest) (*pb.CreateStudentReply, error) {
-	v, ebz := s.uc.CreateStudent(ctx, nil)
+	if req.Name == "" {
+		return nil, pb.ErrorBadParam("NAME IS REQUIRED")
+	}
+	v, ebz := s.uc.CreateStudent(ctx, &biz.Student{
+		Name:      req.Name,
+		Age:       req.Age,
+		ClassName: req.ClassName,
+	})
 	if ebz != nil {
 		return nil, ebz.Erk
 	}
@@ -26,7 +33,18 @@
 }
 
 func (s *StudentService) UpdateStudent(ctx context.Context, req *pb.UpdateStudentRequest) (*pb.UpdateStudentReply, error) {
-	v, ebz := s.uc.UpdateStudent(ctx, nil)
+	if req.Id <= 0 {
+		return nil, pb.ErrorBadParam("ID IS REQUIRED")
+	}
+	if req.Name == "" {
+		return nil, pb.ErrorBadParam("NAME IS REQUIRED")
+	}
+	v, ebz := s.uc.UpdateStudent(ctx, &biz.Student{
+		ID:        req.Id,
+		Name:      req.Name,
+		Age:       req.Age,
+		ClassName: req.ClassName,
+	})
 	if ebz != nil {
 		return nil, ebz.Erk
 	}
@@ -34,6 +52,9 @@
 }
 
 func (s *StudentService) DeleteStudent(ctx context.Context, req *pb.DeleteStudentRequest) (*pb.DeleteStudentReply, error) {
+	if req.Id <= 0 {
+		return nil, pb.ErrorBadParam("ID IS REQUIRED")
+	}
 	if ebz := s.uc.DeleteStudent(ctx, req.Id); ebz != nil {
 		return nil, ebz.Erk
 	}
@@ -41,6 +62,9 @@
 }
 
 func (s *StudentService) GetStudent(ctx context.Context, req *pb.GetStudentRequest) (*pb.GetStudentReply, error) {
+	if req.Id <= 0 {
+		return nil, pb.ErrorBadParam("ID IS REQUIRED")
+	}
 	v, ebz := s.uc.GetStudent(ctx, req.Id)
 	if ebz != nil {
 		return nil, ebz.Erk
```

