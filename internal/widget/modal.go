package widget

// ─── 模态弹出（命令式 API）──────────────────────────────────────
// 把 Dialog/Drawer/Loading 挂到 OverlayHost 顶层真正弹出。
// 依赖 Overlay 的 Fullscreen 模式：浮层铺满视口、HitTest 命中模态层，
// 于是模态自身的 HandleEvent(点 ✕/遮罩关闭)能收到点击。
//
// 用法:
//   id := widget.ShowDialog(widget.NewDialog("提示", body).WithFooter(...))
//   // 用户点 ✕/遮罩自动关闭；也可手动 widget.HideOverlay(id)
//
//   lid := widget.ShowLoading(widget.NewLoading("加载中..."))
//   // 数据就绪后 widget.HideLoading(lid)

// ShowDialog 弹出对话框，返回浮层 id。点 ✕ 或遮罩自动关闭(并回调原 OnClose)。
func ShowDialog(d *Dialog) int {
	var id int
	userClose := d.OnClose
	d.OnClose = func() {
		HideOverlay(id)
		if userClose != nil {
			userClose()
		}
	}
	id = ShowOverlay(&OverlayEntry{Content: d, Fullscreen: true, Transition: "zoom"}) // 缩放+淡入，比纯淡入更平滑、不"跳"
	return id
}

// ShowDrawer 弹出抽屉，返回浮层 id。点遮罩自动关闭(并回调原 OnClose)。
func ShowDrawer(d *Drawer) int {
	var id int
	userClose := d.OnClose
	d.OnClose = func() {
		HideOverlay(id)
		if userClose != nil {
			userClose()
		}
	}
	id = ShowOverlay(&OverlayEntry{Content: d, Fullscreen: true, Transition: "fade"})
	return id
}

// ShowLoading 弹出全屏加载遮罩，返回浮层 id。由程序用 HideLoading(id) 关闭。
func ShowLoading(l *Loading) int {
	return ShowOverlay(&OverlayEntry{Content: l, Fullscreen: true, Transition: "fade"})
}

// HideLoading 关闭加载遮罩(HideOverlay 的语义别名)。
func HideLoading(id int) {
	HideOverlay(id)
}
