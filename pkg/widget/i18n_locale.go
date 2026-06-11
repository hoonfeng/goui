package widget

import "github.com/hoonfeng/goui/internal/i18n"

// 内置组件语言包：key 对标 Element Plus 的 el.<组件>.<字段>。widget 包加载时注册 zh-CN/en，
// 默认 zh-CN（保持中文 UI）。用户可 i18n.Register 覆盖/增补、注册新语言，再 i18n.SetLocale 切换。
//
// 组件接文案统一用 orI18n(用户值, key)：用户显式传了就用用户的，没传则取 i18n 默认（随语言变）。

func init() {
	i18n.Register("zh-CN", localeZhCN)
	i18n.Register("en", localeEn)
	i18n.SetFallback("en")
	i18n.SetLocale("zh-CN") // goui 默认中文 UI（init 期 OnChange 未接，无副作用）
}

// orI18n 文案兜底：s 非空用 s（用户自定义），否则取 i18n.T(key)（随当前语言变）。
func orI18n(s, key string) string {
	if s != "" {
		return s
	}
	return i18n.T(key)
}

var localeZhCN = map[string]string{
	"el.empty.description":            "暂无数据",
	"el.select.placeholder":           "请选择",
	"el.select.noMatch":               "无匹配数据",
	"el.select.noData":                "无数据",
	"el.select.loading":               "加载中",
	"el.cascader.placeholder":         "请选择",
	"el.cascader.noMatch":             "无匹配数据",
	"el.cascader.noData":              "暂无数据",
	"el.popconfirm.confirmButtonText": "确定",
	"el.popconfirm.cancelButtonText":  "取消",
	"el.messagebox.confirm":           "确定",
	"el.messagebox.cancel":            "取消",
	"el.messagebox.title":             "提示",
	"el.dialog.confirm":               "确定",
	"el.dialog.cancel":                "取消",
	"el.pagination.total":             "共 {total} 条",
	"el.pagination.pagesize":          "{size} 条/页",
	"el.pagination.goto":              "前往",
	"el.pagination.pageClassifier":    "页",
	"el.pagination.prev":              "上一页",
	"el.pagination.next":              "下一页",
	"el.datepicker.now":               "此刻",
	"el.datepicker.today":             "今天",
	"el.datepicker.cancel":            "取消",
	"el.datepicker.clear":             "清空",
	"el.datepicker.confirm":           "确定",
	"el.datepicker.selectDate":        "选择日期",
	"el.datepicker.selectTime":        "选择时间",
	"el.datepicker.startDate":         "开始日期",
	"el.datepicker.endDate":           "结束日期",
	"el.table.emptyText":              "暂无数据",
	"el.table.confirmFilter":          "筛选",
	"el.table.resetFilter":            "重置",
	"el.tree.emptyText":               "暂无数据",
	"el.transfer.noMatch":             "无匹配数据",
	"el.transfer.noData":              "无数据",
	"el.transfer.titles.source":       "列表 1",
	"el.transfer.titles.target":       "列表 2",
	"el.image.error":                  "加载失败",
	"el.inputtag.placeholder":         "回车添加",
	"el.autocomplete.placeholder":     "请输入",
	"el.mention.placeholder":          "输入 @ 提及某人",
	"el.upload.tip":                   "将文件拖到此处，或点击上传",
	"el.upload.delete":                "删除",
	"el.upload.preview":               "查看图片",
	"el.upload.continue":              "继续上传",
}

var localeEn = map[string]string{
	"el.empty.description":            "No Data",
	"el.select.placeholder":           "Select",
	"el.select.noMatch":               "No matching data",
	"el.select.noData":                "No data",
	"el.select.loading":               "Loading",
	"el.cascader.placeholder":         "Select",
	"el.cascader.noMatch":             "No matching data",
	"el.cascader.noData":              "No data",
	"el.popconfirm.confirmButtonText": "OK",
	"el.popconfirm.cancelButtonText":  "Cancel",
	"el.messagebox.confirm":           "OK",
	"el.messagebox.cancel":            "Cancel",
	"el.messagebox.title":             "Message",
	"el.dialog.confirm":               "OK",
	"el.dialog.cancel":                "Cancel",
	"el.pagination.total":             "Total {total}",
	"el.pagination.pagesize":          "{size}/page",
	"el.pagination.goto":              "Go to",
	"el.pagination.pageClassifier":    "",
	"el.pagination.prev":              "Previous",
	"el.pagination.next":              "Next",
	"el.datepicker.now":               "Now",
	"el.datepicker.today":             "Today",
	"el.datepicker.cancel":            "Cancel",
	"el.datepicker.clear":             "Clear",
	"el.datepicker.confirm":           "OK",
	"el.datepicker.selectDate":        "Select date",
	"el.datepicker.selectTime":        "Select time",
	"el.datepicker.startDate":         "Start date",
	"el.datepicker.endDate":           "End date",
	"el.table.emptyText":              "No Data",
	"el.table.confirmFilter":          "Confirm",
	"el.table.resetFilter":            "Reset",
	"el.tree.emptyText":               "No Data",
	"el.transfer.noMatch":             "No matching data",
	"el.transfer.noData":              "No data",
	"el.transfer.titles.source":       "List 1",
	"el.transfer.titles.target":       "List 2",
	"el.image.error":                  "FAILED",
	"el.inputtag.placeholder":         "Press Enter",
	"el.autocomplete.placeholder":     "Please input",
	"el.mention.placeholder":          "Type @ to mention",
	"el.upload.tip":                   "Drop file here or click to upload",
	"el.upload.delete":                "Delete",
	"el.upload.preview":               "Preview",
	"el.upload.continue":              "Continue",
}
