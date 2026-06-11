package widget

import (
	"github.com/hoonfeng/goui/pkg/canvas"
	"github.com/hoonfeng/goui/pkg/paint"
	"github.com/hoonfeng/goui/pkg/types"
)

// 缩略图 minimap：右侧窄条画整文件 token 色块鸟瞰 + 当前视口框，点击/拖动跳转。

const (
	ceMiniW     = 78.0 // 缩略图宽
	ceMiniLineH = 3.0  // 每行高（很矮）
	ceMiniCharW = 1.5  // 每字符宽
)

// paintMinimap 画缩略图（miniX=左边界，areaTop/areaH=可视区；viewH/scrollY/maxScrollY=编辑区滚动信息）。
func (e *CodeEditorElement) paintMinimap(cvs canvas.Canvas, miniX, areaTop, areaH, viewH, scrollY, maxScrollY float64) {
	bg := paint.DefaultPaint()
	bg.Color = CurrentTheme().CodeEditor.MinimapBg
	cvs.DrawRect(miniX, areaTop, ceMiniW, areaH, bg)
	sep := paint.DefaultStrokePaint()
	sep.Color = elBorder()
	cvs.DrawLine(miniX, areaTop, miniX, areaTop+areaH, sep)

	// 按视觉段（wrapSegs）逐行画：scrollY 是视觉行单位，缩略图随之对齐（换行时一段一行）。
	nRows := len(e.wrapSegs)
	totalH := float64(nRows) * ceMiniLineH
	// 缩略图内容超出可视高时，按编辑器滚动比例同步偏移
	e.miniScrollY = 0
	if totalH > areaH && maxScrollY > 0 {
		e.miniScrollY = (scrollY / maxScrollY) * (totalH - areaH)
	}
	e.miniRect = types.Rect{X: miniX, Y: areaTop, Width: ceMiniW, Height: areaH}

	cvs.Save()
	cvs.ClipRect(miniX, areaTop, ceMiniW, areaH)
	for vi := 0; vi < nRows; vi++ {
		ly := areaTop + float64(vi)*ceMiniLineH - e.miniScrollY
		if ly+ceMiniLineH < areaTop || ly > areaTop+areaH {
			continue
		}
		s := e.wrapSegs[vi]
		for _, tk := range e.hl[s.line] {
			// token 裁剪到本段 [s.start,s.end)，并以段起列为 0 平移
			a, b := tk.start, tk.end
			if b <= s.start || a >= s.end {
				continue
			}
			if a < s.start {
				a = s.start
			}
			if b > s.end {
				b = s.end
			}
			x := miniX + 3 + float64(a-s.start)*ceMiniCharW
			if x > miniX+ceMiniW {
				continue
			}
			ww := float64(b-a) * ceMiniCharW
			if x+ww > miniX+ceMiniW {
				ww = miniX + ceMiniW - x
			}
			cp := paint.DefaultPaint()
			cp.Color = ceTokenColor(tk.kind)
			cvs.DrawRect(x, ly+0.5, ww, ceMiniLineH-1, cp)
		}
	}
	cvs.Restore()

	// 视口框（当前可见区域）
	if maxScrollY > 0 {
		firstRow := scrollY / ceLineH
		rowsInView := viewH / ceLineH
		boxTop := areaTop + firstRow*ceMiniLineH - e.miniScrollY
		boxH := rowsInView * ceMiniLineH
		vp := paint.DefaultPaint()
		vp.Color = types.ColorFromRGBA(0x40, 0x9E, 0xFF, 38)
		cvs.DrawRect(miniX, boxTop, ceMiniW, boxH, vp)
		vb := paint.DefaultStrokePaint()
		vb.Color = types.ColorFromRGBA(0x40, 0x9E, 0xFF, 120)
		cvs.DrawRect(miniX+0.5, boxTop, ceMiniW-1, boxH, vb)
	}
}

// minimapJump 点击/拖动缩略图 my 处 → 编辑器滚到对应行（居中）。
func (e *CodeEditorElement) minimapJump(my float64) {
	if e.miniRect.Height <= 0 {
		return
	}
	vi := (my - e.miniRect.Y + e.miniScrollY) / ceMiniLineH
	viewH := e.size.Height - 8
	contentH := float64(len(e.wrapSegs)) * ceLineH
	maxScrollY := contentH - viewH
	if maxScrollY < 0 {
		maxScrollY = 0
	}
	e.scrollY = clamp(vi*ceLineH-viewH/2, 0, maxScrollY)
	repaint()
}

// minimapHit 判断点是否在缩略图区域。
func (e *CodeEditorElement) minimapHit(x, y float64) bool {
	r := e.miniRect
	return r.Width > 0 && x >= r.X && x <= r.X+r.Width && y >= r.Y && y <= r.Y+r.Height
}
