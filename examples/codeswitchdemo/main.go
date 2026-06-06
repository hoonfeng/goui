// codeswitchdemo — 同一个 Go 程序在「传统代码视图 ⇄ 表格视图」之间切换。
// 运行: $env:CGO_ENABLED='1'; go run ./examples/codeswitchdemo/
package main

import (
	"log"
	"runtime"

	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/widget"
)

const goSrc = `package main

var 全局计数 int

// 阶乘 递归求 n 的阶乘
func 阶乘(n int) int {
	积 := 1
	for i := 1; i <= n; i++ {
		积 *= i
	}
	return 积
}

// 除法 返回商与余数
func 除法(a, b int) (商 int, 余 int) {
	return a / b, a % b
}
`

func main() {
	runtime.LockOSThread()
	wb := widget.NewCodeWorkbench(goSrc).WithSize(820, 700)
	application := app.NewApplication()
	application.SetRootWidget(widget.NewScrollView(wb))
	config := app.DefaultConfig()
	config.Title = "代码视图 ⇄ 表格视图 切换"
	config.Width = 860
	config.Height = 760
	config.Resizable = true
	log.Println("codeswitchdemo: 顶部按钮切换 传统代码 / 表格 视图")
	if err := application.Run(config); err != nil {
		log.Fatalf("codeswitchdemo: %v", err)
	}
}
