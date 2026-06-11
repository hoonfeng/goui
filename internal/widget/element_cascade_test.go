package widget

import "testing"

// tcCfgChild 带一个配置字段的有状态子组件（测试级联重建跳过用）。
type tcCfgChild struct {
	StatefulWidget
	V int
}

func (w *tcCfgChild) CreateState() State { return &tcCfgChildState{} }

type tcCfgChildState struct{ BaseState }

func (s *tcCfgChildState) Build(ctx BuildContext) Widget { return nil }

// TestStatefulUpdateConfigEqual 父级重建对子元素 Update 时：配置深度相等→不标脏（复用缓存，
// 不级联重建——拖动改尺寸等高频 SetState 不再重渲未变面板）；配置变化→照常标脏重建。
func TestStatefulUpdateConfigEqual(t *testing.T) {
	el := createStatefulElement(&tcCfgChild{V: 1})
	el.Build()                   // 初始化 state（CreateState）
	el.Update(&tcCfgChild{V: 1}) // 确立当前 widget = V:1
	el.buildDirty = false        // 模拟构建完成、清脏

	el.Update(&tcCfgChild{V: 1}) // 配置相等
	if el.buildDirty {
		t.Error("配置相等不应标脏：父级重建应跳过未变子面板的级联重建")
	}

	el.Update(&tcCfgChild{V: 2}) // 配置变化
	if !el.buildDirty {
		t.Error("配置变化应标脏重建")
	}
}

// 端到端：config-less 子面板（像 companion 的 ChatPanel/EditorPanel）在父级重建时跳过 state.Build。
var tcExpensiveBuilds int

type tcLeaf struct{ StatefulWidget }

func (w *tcLeaf) CreateState() State { return &tcLeafState{} }

type tcLeafState struct{ BaseState }

func (s *tcLeafState) Build(ctx BuildContext) Widget { return nil }

type tcExpensive struct{ StatefulWidget } // config-less，模拟重渲 Markdown 的对话面板

func (w *tcExpensive) CreateState() State { return &tcExpensiveState{} }

type tcExpensiveState struct{ BaseState }

func (s *tcExpensiveState) Build(ctx BuildContext) Widget {
	tcExpensiveBuilds++
	return &tcLeaf{} // 返回非 nil 子树，使父 Element 缓存生效（e.child != nil）
}

type tcRoot struct{ StatefulWidget }

func (w *tcRoot) CreateState() State { return &tcRootState{} }

type tcRootState struct{ BaseState }

func (s *tcRootState) Build(ctx BuildContext) Widget { return &tcExpensive{} }

// TestCascadeSkipsChildBuild 父级 SetState 后再走 buildTree：config-less 子面板配置未变，
// 不再级联重跑 state.Build（拖动改尺寸等高频重建不再重渲未变面板）。
func TestCascadeSkipsChildBuild(t *testing.T) {
	tcExpensiveBuilds = 0
	root := createStatefulElement(&tcRoot{})
	var rec func(Element)
	rec = func(e Element) {
		for _, c := range e.Build() {
			rec(c)
		}
	}
	rec(root) // 首次：子面板构建 1 次
	if tcExpensiveBuilds != 1 {
		t.Fatalf("首次应构建 1 次，得 %d", tcExpensiveBuilds)
	}
	root.Rebuild() // 父级 SetState（如 shell 拖动改尺寸）
	rec(root)      // 再 buildTree：子面板配置未变 → 跳过
	if tcExpensiveBuilds != 1 {
		t.Errorf("父级重建不应级联重建未变子面板：应仍 1 次，得 %d", tcExpensiveBuilds)
	}
}
