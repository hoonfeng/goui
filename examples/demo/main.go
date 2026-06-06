// goui 声明式 UI 演示 Demo
//
// 本演示展示了 goui 的两大核心能力：
//   1. 声明式 JSON 配置驱动 UI（加载外部配置文件，无需重编译即可修改 UI）
//   2. Web 风格组件 + 主题系统 + CSS 样式类
//
// 运行方式:
//   go run ./examples/demo/
//
// 一键启动，无需额外依赖。
package main

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/user/goui/internal/app"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

func init() {
	// ═══════════════════════════════════════════════
	// 1. 配置主题（类似 CSS 变量 / Design Tokens）
	// ═══════════════════════════════════════════════
	theme := widget.DefaultTheme()
	theme.PrimaryColor = types.ColorFromRGB(66, 133, 244)    // Google Blue
	theme.BGColor = types.ColorFromRGB(245, 247, 250)        // 浅灰蓝背景
	theme.SurfaceColor = types.ColorFromRGB(255, 255, 255)   // 白色卡片
	theme.TextColor = types.ColorFromRGB(33, 33, 33)         // 深灰文字
	theme.SecondaryText = types.ColorFromRGB(120, 120, 120)  // 次要文字
	theme.BorderColor = types.ColorFromRGB(220, 224, 228)    // 浅边框
	theme.DividerColor = types.ColorFromRGB(230, 234, 238)   // 分割线
	theme.SuccessColor = types.ColorFromRGB(52, 168, 83)     // 成功绿
	theme.WarningColor = types.ColorFromRGB(251, 188, 4)     // 警告黄
	theme.ErrorColor = types.ColorFromRGB(234, 67, 53)       // 错误红
	widget.SetTheme(theme)

	// ═══════════════════════════════════════════════
	// 2. 定义 CSS 风格样式类（类似 CSS .class {}）
	// ═══════════════════════════════════════════════

	widget.Define("page", widget.Style{
		Padding:         types.EdgeInsetsLTRB(40, 32, 40, 32),
		BackgroundColor: types.ColorRef(245, 247, 250),
	})

	widget.Define("header-card", widget.Style{
		Padding:         types.EdgeInsets(24),
		BackgroundColor: types.ColorRef(66, 133, 244),
		BorderRadius:    12,
		Margin:          types.EdgeInsetsLTRB(0, 0, 0, 20),
	})

	widget.Define("demo-card", widget.Style{
		Padding:         types.EdgeInsets(20),
		BackgroundColor: types.ColorRef(255, 255, 255),
		BorderColor:     types.ColorRef(220, 224, 228),
		BorderWidth:     1,
		BorderRadius:    10,
		Margin:          types.EdgeInsetsLTRB(0, 0, 16, 16),
	})

	widget.Define("form-group", widget.Style{
		Padding: types.EdgeInsetsLTRB(0, 0, 0, 12),
	})

	widget.Define("button-row", widget.Style{
		Padding: types.EdgeInsetsLTRB(0, 8, 0, 0),
	})

	widget.Define("status-bar", widget.Style{
		Padding:         types.EdgeInsetsLTRB(0, 16, 0, 4),
		BackgroundColor: types.ColorRef(33, 33, 33),
		BorderRadius:    8,
	})
}

func main() {
	runtime.LockOSThread()

	// ═══════════════════════════════════════════════
	// 3. 定义事件处理器
	// ═══════════════════════════════════════════════
	handlers := widget.Handlers{
		"onLogin": func(ctx widget.EventContext) {
			log.Println("[事件] 登录按钮点击")
		},
		"onRegister": func(ctx widget.EventContext) {
			log.Println("[事件] 注册按钮点击")
		},
		"onRemember": func(ctx widget.EventContext) {
			if checked, ok := ctx.Data.(bool); ok {
				log.Printf("[事件] 记住登录 = %v", checked)
			}
		},
		"onUserChange": func(ctx widget.EventContext) {
			if text, ok := ctx.Data.(string); ok {
				log.Printf("[事件] 用户名输入: %s", text)
			}
		},
		"onPassChange": func(ctx widget.EventContext) {
			log.Println("[事件] 密码输入变更")
		},
		"onNotify": func(ctx widget.EventContext) {
			if val, ok := ctx.Data.(bool); ok {
				log.Printf("[事件] 推送通知 = %v", val)
			}
		},
		"onVolume": func(ctx widget.EventContext) {
			if val, ok := ctx.Data.(float64); ok {
				log.Printf("[事件] 音量 = %.0f%%", val)
			}
		},
		"onThemeLight": func(ctx widget.EventContext) {
			log.Println("[事件] 切换为浅色主题")
		},
		"onThemeDark": func(ctx widget.EventContext) {
			log.Println("[事件] 切换为深色主题")
		},
	}

	// ═══════════════════════════════════════════════
	// 4. 从 JSON 配置文件加载 UI
	// ═══════════════════════════════════════════════
	//
	// 核心优势：UI 描述在 JSON 文件中，修改 UI 无需重编译！
	// 设计师/产品经理可直接编辑 JSON 调整布局。
	configPath := filepath.Join(findBaseDir(), "examples", "demo", "demo_config.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("读取配置文件失败: %v", err)
	}

	root, err := widget.LoadConfig(data, handlers)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	log.Printf("UI 配置加载成功 (%s)", configPath)

	// ═══════════════════════════════════════════════
	// 5. 创建并启动应用
	// ═══════════════════════════════════════════════
	application := app.NewApplication()
	application.SetRootWidget(root)

	config := app.DefaultConfig()
	config.Title = "goui 声明式 UI 演示"
	config.Width = 720
	config.Height = 960
	config.Resizable = true

	log.Println("启动 goui 演示应用...")
	if err := application.Run(config); err != nil {
		log.Fatalf("应用运行失败: %v", err)
	}
	log.Println("应用正常退出")
}

// ─── 辅助函数 ──────────────────────────────────

// findBaseDir 查找项目根目录（从可执行文件所在目录或 CWD 向上查找）
func findBaseDir() string {
	exe, err := os.Executable()
	if err == nil {
		dir := filepath.Dir(exe)
		if fileExists(filepath.Join(dir, "go.mod")) {
			return dir
		}
		if fileExists(filepath.Join(dir, "..", "go.mod")) {
			return filepath.Dir(dir)
		}
	}
	cwd, _ := os.Getwd()
	return cwd
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
