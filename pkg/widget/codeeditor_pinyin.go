package widget

import (
	"strings"
	"sync"

	"golang.org/x/text/encoding/simplifiedchinese"
)

// 拼音首字母：用 GB2312/GBK「按拼音排序」的区位特性——一级汉字区(0xB0A1~0xD7FA)按拼音首字母
// 连续排布，只需 23 个分界点即可把汉字映射到首字母。用于补全的拼音首字母匹配（如 "scts"→"输出调试…"）。

var pyBounds = []struct {
	code uint16
	ch   byte
}{
	{0xB0A1, 'a'}, {0xB0C5, 'b'}, {0xB2C1, 'c'}, {0xB4EE, 'd'},
	{0xB6EA, 'e'}, {0xB7A2, 'f'}, {0xB8C1, 'g'}, {0xB9FE, 'h'},
	{0xBBF7, 'j'}, {0xBFA6, 'k'}, {0xC0AC, 'l'}, {0xC2E8, 'm'},
	{0xC4C3, 'n'}, {0xC5B6, 'o'}, {0xC5BE, 'p'}, {0xC6DA, 'q'},
	{0xC8BB, 'r'}, {0xC8F6, 's'}, {0xCBFA, 't'}, {0xCDDA, 'w'},
	{0xCEF4, 'x'}, {0xD1B9, 'y'}, {0xD4D8, 'z'},
}

// pinyinInitial 返回单个汉字的拼音首字母（GBK 区位法，取主读音）；非一级常用汉字返回 0。
func pinyinInitial(r rune) byte {
	gbk, err := simplifiedchinese.GBK.NewEncoder().String(string(r))
	if err != nil || len(gbk) != 2 {
		return 0
	}
	code := uint16(gbk[0])<<8 | uint16(gbk[1])
	if code < 0xB0A1 || code > 0xD7FA {
		return 0 // 非一级汉字区（符号/二级字等）
	}
	for i := len(pyBounds) - 1; i >= 0; i-- {
		if code >= pyBounds[i].code {
			return pyBounds[i].ch
		}
	}
	return 0
}

var (
	pyCacheMu sync.Mutex
	pyCache   = map[string]string{}
)

// pinyinInitials 把字符串转成「拼音首字母串」：汉字→首字母、ASCII 字母→小写、其余字符跳过；带缓存。
func pinyinInitials(s string) string {
	pyCacheMu.Lock()
	if v, ok := pyCache[s]; ok {
		pyCacheMu.Unlock()
		return v
	}
	pyCacheMu.Unlock()
	var b strings.Builder
	for _, r := range s {
		if r < 0x80 {
			if r >= 'A' && r <= 'Z' {
				r += 32
			}
			b.WriteRune(r)
		} else if c := pinyinInitial(r); c != 0 {
			b.WriteByte(c)
		}
	}
	v := b.String()
	pyCacheMu.Lock()
	pyCache[s] = v
	pyCacheMu.Unlock()
	return v
}
