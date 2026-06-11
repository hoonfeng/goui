// Package event 定义了 goui 框架的事件系统。
package event

// Shortcut 表示一个键盘快捷键组合
type Shortcut struct {
	KeyCode uint32       // 虚拟键码
	Mods    ModifierKeys // 修饰键组合
	Label   string       // 可读标识（如 "Ctrl+C"），用于显示
}

// ShortcutHandler 快捷键触发时的回调函数
type ShortcutHandler func()

// ShortcutManager 快捷键管理器
// 管理全局快捷键注册和匹配，在 KeyDown 事件处理时优先匹配快捷键表
type ShortcutManager struct {
	shortcuts map[Shortcut]ShortcutHandler
}

// NewShortcutManager 创建快捷键管理器
func NewShortcutManager() *ShortcutManager {
	return &ShortcutManager{
		shortcuts: make(map[Shortcut]ShortcutHandler),
	}
}

// Register 注册快捷键
// keyCode: 虚拟键码（如 0x43 表示 C 键）
// mods: 修饰键组合（如 ModCtrl|ModShift）
// handler: 触发回调
// label: 可读描述（如 "Ctrl+Shift+C"），用于菜单显示
func (m *ShortcutManager) Register(keyCode uint32, mods ModifierKeys, handler ShortcutHandler, label string) {
	s := Shortcut{
		KeyCode: keyCode,
		Mods:    mods,
		Label:   label,
	}
	m.shortcuts[s] = handler
}

// Unregister 移除快捷键注册
func (m *ShortcutManager) Unregister(keyCode uint32, mods ModifierKeys) {
	s := Shortcut{
		KeyCode: keyCode,
		Mods:    mods,
	}
	delete(m.shortcuts, s)
}

// Match 尝试匹配快捷键，匹配成功返回 true 并执行回调
// 如果快捷键被匹配到且回调执行，返回 true（表示事件已被消费）
func (m *ShortcutManager) Match(ev *KeyEvent) bool {
	if ev.Type() != TypeKeyDown {
		return false
	}
	s := Shortcut{
		KeyCode: ev.KeyCode,
		Mods:    ev.Mods,
	}
	if handler, ok := m.shortcuts[s]; ok {
		handler()
		return true
	}
	// 特殊处理：精确匹配（不考虑未按下的修饰键）
	// 如果事件带 Ctrl，但注册的是 Ctrl+X，只要 Ctrl 匹配就可行
	// 但如果事件不带 Ctrl，注册需要 Ctrl，则不匹配
	return false
}

// MatchExact 精确匹配快捷键（修饰键必须完全一致）
func (m *ShortcutManager) MatchExact(ev *KeyEvent) bool {
	if ev.Type() != TypeKeyDown {
		return false
	}
	s := Shortcut{
		KeyCode: ev.KeyCode,
		Mods:    ev.Mods,
	}
	if handler, ok := m.shortcuts[s]; ok {
		handler()
		return true
	}
	return false
}

// GetShortcuts 返回所有已注册的快捷键列表
func (m *ShortcutManager) GetShortcuts() []Shortcut {
	result := make([]Shortcut, 0, len(m.shortcuts))
	for s := range m.shortcuts {
		result = append(result, s)
	}
	return result
}

// Clear 清空所有快捷键注册
func (m *ShortcutManager) Clear() {
	m.shortcuts = make(map[Shortcut]ShortcutHandler)
}
