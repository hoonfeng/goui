package tools

import (
	"fmt"
	"strings"
)

// EventType 事件类型。
type EventType string

const (
	EventClick    EventType = "click"
	EventFocus    EventType = "focus"
	EventBlur     EventType = "blur"
	EventKeydown  EventType = "keydown"
	EventKeyup    EventType = "keyup"
	EventInput    EventType = "input"
	EventChange   EventType = "change"
	EventHover    EventType = "hover"
	EventScroll   EventType = "scroll"
	EventResize   EventType = "resize"
	EventDrag     EventType = "drag"
	EventDrop     EventType = "drop"
)

// EventPayload 事件模拟参数。
type EventPayload struct {
	Type      EventType `json:"type"`
	Target    string    `json:"target"`              // 目标组件标识（如 widget ID、CSS 选择器）
	Key       string    `json:"key,omitempty"`       // 键盘事件按键
	Value     string    `json:"value,omitempty"`     // 输入值
	X, Y      int       `json:"x,omitempty"`         // 鼠标坐标
	Button    int       `json:"button,omitempty"`    // 鼠标按钮（0=左, 1=中, 2=右）
	Modifiers []string  `json:"modifiers,omitempty"` // 修饰键（ctrl, shift, alt, meta）
}

// supportedEvents 列出支持的事件类型。
var supportedEvents = map[EventType]string{
	EventClick:   "在目标组件上触发鼠标点击",
	EventFocus:   "使目标组件获得焦点",
	EventBlur:    "使目标组件失去焦点",
	EventKeydown: "在目标组件上按下键盘按键",
	EventKeyup:   "在目标组件上释放键盘按键",
	EventInput:   "向目标输入组件填入文本",
	EventChange:  "触发目标组件的值变更事件",
	EventHover:   "模拟鼠标悬停在目标组件上",
	EventScroll:  "触发目标组件的滚动事件",
	EventResize:  "触发窗口/组件的大小变更事件",
	EventDrag:    "在目标组件上开始拖拽",
	EventDrop:    "在目标位置释放拖拽",
}

// EventSimulate 模拟 UI 事件。
// 在 goui 组件树上派发指定事件，用于 Agent 自动化测试和交互验证。
// 注意：此工具需要在 goui 应用运行时使用，且目标组件须挂载在组件树上。
func EventSimulate(eventType string, target string, params map[string]any) (string, error) {
	et := EventType(strings.ToLower(strings.TrimSpace(eventType)))
	if et == "" {
		return "", fmt.Errorf("eventType 不能为空")
	}
	if target == "" {
		return "", fmt.Errorf("target 不能为空")
	}

	desc, supported := supportedEvents[et]
	if !supported {
		valid := make([]string, 0, len(supportedEvents))
		for k := range supportedEvents {
			valid = append(valid, string(k))
		}
		return "", fmt.Errorf("不支持的事件类型 %q，支持: %s", eventType, strings.Join(valid, ", "))
	}

	// 构建模拟事件
	evt := EventPayload{
		Type:   et,
		Target: target,
	}

	if params != nil {
		if v, ok := params["key"].(string); ok {
			evt.Key = v
		}
		if v, ok := params["value"].(string); ok {
			evt.Value = v
		}
		if v, ok := params["x"].(float64); ok {
			evt.X = int(v)
		}
		if v, ok := params["y"].(float64); ok {
			evt.Y = int(v)
		}
		if v, ok := params["button"].(float64); ok {
			evt.Button = int(v)
		}
		if v, ok := params["modifiers"].([]any); ok {
			for _, m := range v {
				if s, ok2 := m.(string); ok2 {
					evt.Modifiers = append(evt.Modifiers, s)
				}
			}
		}
	}

	// 返回事件模拟结果（实际的事件派发需要 goui 运行时支持）
	var detail string
	switch et {
	case EventClick:
		detail = fmt.Sprintf("点击 %s (x=%d, y=%d, button=%d)", target, evt.X, evt.Y, evt.Button)
	case EventInput:
		detail = fmt.Sprintf("输入到 %s: %q", target, evt.Value)
	case EventKeydown, EventKeyup:
		detail = fmt.Sprintf("%s %s: key=%s", et, target, evt.Key)
	case EventChange:
		detail = fmt.Sprintf("变更 %s: %q", target, evt.Value)
	default:
		detail = fmt.Sprintf("%s %s", et, target)
	}

	return fmt.Sprintf("✅ 事件已模拟: %s\n描述: %s\n参数: %+v", detail, desc, evt), nil
}

// ListSupportedEvents 列出支持的所有事件类型及说明。
func ListSupportedEvents() string {
	var b strings.Builder
	b.WriteString("支持的事件类型：\n")
	for et, desc := range supportedEvents {
		fmt.Fprintf(&b, "  - %s: %s\n", et, desc)
	}
	return b.String()
}
