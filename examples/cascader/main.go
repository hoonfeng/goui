// cascader — 验证 batch④ 重型组件：Cascader 级联选择器(el-cascader 单选核心)。
// 浮层联动需窗口实测；此处 headless 渲染触发器 + 展开三列的面板 preview。
//
// 运行: $env:CGO_ENABLED='1'; go run ./examples/cascader/
package main

import (
	"fmt"
	"os"

	"github.com/user/goui/internal/canvas"
	"github.com/user/goui/internal/render"
	"github.com/user/goui/internal/types"
	"github.com/user/goui/internal/widget"
)

const (
	cw = 560
	ch = 280
)

func main() {
	sk := canvas.NewSkiaCanvas(cw, ch)
	defer sk.Release()

	options := []widget.CascaderOption{
		{Label: "浙江", Value: "zhejiang", Children: []widget.CascaderOption{
			{Label: "杭州", Value: "hangzhou", Children: []widget.CascaderOption{
				{Label: "西湖区", Value: "xihu"},
				{Label: "余杭区", Value: "yuhang"},
				{Label: "滨江区", Value: "binjiang"},
			}},
			{Label: "宁波", Value: "ningbo", Children: []widget.CascaderOption{
				{Label: "海曙区", Value: "haishu"},
				{Label: "江北区", Value: "jiangbei"},
			}},
		}},
		{Label: "江苏", Value: "jiangsu", Children: []widget.CascaderOption{
			{Label: "南京", Value: "nanjing", Children: []widget.CascaderOption{
				{Label: "玄武区", Value: "xuanwu"},
			}},
		}},
	}
	path := []string{"zhejiang", "hangzhou", "xihu"}

	root := widget.Div(
		widget.Style{Width: cw, Height: ch, Padding: types.EdgeInsets(24), BackgroundColor: types.ColorRef(245, 247, 250), FlexDirection: "column", Gap: 16},
		widget.H3("Cascader 级联选择器（点列展开·叶子回填路径）"),
		widget.NewCascader(options).WithValue(path).WithWidth(320),
		widget.CascaderPanelPreview(options, path),
	)

	rootEl := widget.CreateElementFor(root)
	pipe := render.NewPipeline(cw, ch, sk)
	pipe.SetRootElement(rootEl)
	if err := pipe.Render(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := sk.SaveToPNG("cascader_demo.png"); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Println("✅ cascader_demo.png 已保存")
}
