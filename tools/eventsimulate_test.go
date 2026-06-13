package tools

import (
	"strings"
	"testing"
)

func TestEventSimulate(t *testing.T) {
	// 测试空事件类型
	_, err := EventSimulate("", "button1", nil)
	if err == nil {
		t.Error("EventSimulate should fail on empty eventType")
	}

	// 测试空目标
	_, err = EventSimulate("click", "", nil)
	if err == nil {
		t.Error("EventSimulate should fail on empty target")
	}

	// 测试 click 事件
	got, err := EventSimulate("click", "#submit-btn", map[string]any{
		"x": float64(100),
		"y": float64(50),
	})
	if err != nil {
		t.Fatalf("EventSimulate click failed: %v", err)
	}
	if !strings.Contains(got, "点击") {
		t.Errorf("EventSimulate unexpected: %s", got)
	}
	if !strings.Contains(got, "#submit-btn") {
		t.Errorf("EventSimulate missing target: %s", got)
	}

	// 测试 input 事件
	got, err = EventSimulate("input", "#name-input", map[string]any{
		"value": "test user",
	})
	if err != nil {
		t.Fatalf("EventSimulate input failed: %v", err)
	}
	if !strings.Contains(got, "输入") {
		t.Errorf("EventSimulate input unexpected: %s", got)
	}
	if !strings.Contains(got, "test user") {
		t.Errorf("EventSimulate input missing value: %s", got)
	}

	// 测试 keydown 事件
	got, err = EventSimulate("keydown", "#search-box", map[string]any{
		"key": "Enter",
	})
	if err != nil {
		t.Fatalf("EventSimulate keydown failed: %v", err)
	}
	if !strings.Contains(got, "keydown") || !strings.Contains(got, "Enter") {
		t.Errorf("EventSimulate keydown unexpected: %s", got)
	}

	// 测试不支持的事件类型
	_, err = EventSimulate("swipe", "#el", nil)
	if err == nil {
		t.Error("EventSimulate should fail on unsupported event type")
	}
}

func TestListSupportedEvents(t *testing.T) {
	got := ListSupportedEvents()
	if !strings.Contains(got, "click") {
		t.Errorf("ListSupportedEvents missing click: %s", got)
	}
	if !strings.Contains(got, "input") {
		t.Errorf("ListSupportedEvents missing input: %s", got)
	}
	if !strings.Contains(got, "keydown") {
		t.Errorf("ListSupportedEvents missing keydown: %s", got)
	}
}
