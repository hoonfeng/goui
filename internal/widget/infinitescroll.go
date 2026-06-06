package widget

// ─── InfiniteScroll 无限滚动（el-infinite-scroll，复用 VirtualList）─
// StatefulWidget 维护已加载条数；VirtualList 滚到接近底部时(OnReachEnd)
// 按 PageSize 增长，直到 Total。

type InfiniteScroll struct {
	StatefulWidget
	Total      int
	PageSize   int
	ItemHeight float64
	Height     float64
	RenderItem func(index int) Widget
}

func NewInfiniteScroll(total, pageSize int, itemH float64, render func(int) Widget) *InfiniteScroll {
	return &InfiniteScroll{Total: total, PageSize: pageSize, ItemHeight: itemH, Height: 280, RenderItem: render}
}
func (i *InfiniteScroll) WithHeight(h float64) *InfiniteScroll { i.Height = h; return i }

func (i *InfiniteScroll) CreateState() State { return &infiniteState{} }

type infiniteState struct {
	BaseState
	loaded int
}

func (s *infiniteState) cfg() *InfiniteScroll {
	if w, ok := s.Widget().(*InfiniteScroll); ok {
		return w
	}
	return &InfiniteScroll{}
}

func (s *infiniteState) Build(ctx BuildContext) Widget {
	c := s.cfg()
	if s.loaded <= 0 {
		s.loaded = c.PageSize
		if s.loaded <= 0 {
			s.loaded = 20
		}
	}
	count := s.loaded
	if count > c.Total {
		count = c.Total
	}
	return NewVirtualList(count, c.ItemHeight, c.RenderItem).
		WithHeight(c.Height).
		WithOnReachEnd(func() {
			if s.loaded < c.Total {
				s.loaded += c.PageSize
				s.SetState()
			}
		})
}
