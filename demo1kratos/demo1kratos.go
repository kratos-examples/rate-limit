package demo1kratos

import "github.com/yylego/runpath"

// SourceRoot 返回 demo1kratos 项目的源代码根目录路径
func SourceRoot() string {
	return runpath.PARENT.Path()
}
