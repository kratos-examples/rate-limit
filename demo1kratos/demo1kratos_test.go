package demo1kratos

import (
	"path/filepath"
	"testing"

	"github.com/yylego/must"
	"github.com/yylego/must/muststrings"
)

// TestSourceRoot 测试获取源代码根目录路径的功能
func TestSourceRoot(t *testing.T) {
	path := SourceRoot()

	must.Nice(path)
	muststrings.HasSuffix(path, "/demo1kratos")
	must.True(filepath.IsAbs(path)) // 路径是绝对路径
}
