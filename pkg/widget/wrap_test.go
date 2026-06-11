package widget

import (
	"reflect"
	"testing"
)

// TestWrapRunes 验证折行点（O(n) 重写后仍正确）。measure 用「每字宽 1」便于断言。
func TestWrapRunes(t *testing.T) {
	w1 := func(s string) float64 { return float64(len([]rune(s))) }
	cases := []struct {
		name  string
		s     string
		viewW float64
		want  []int
	}{
		{"empty", "", 4, []int{0}},
		{"fits", "abc", 10, []int{0}},
		{"nospace_wrap", "abcdefghij", 4, []int{0, 4, 8}}, // abcd/efgh/ij
		{"space_break", "ab cd ef gh", 5, []int{0, 6}},     // "ab cd "(6)? 见下
		{"zero_width", "abcdef", 0, []int{0}},
	}
	for _, c := range cases {
		got := wrapRunes([]rune(c.s), w1, c.viewW)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: wrapRunes(%q, %g) = %v, want %v", c.name, c.s, c.viewW, got, c.want)
		}
	}
}
