// 项目根：文件树 / 终端 / agent 统一用「当前项目根」（打开的文件夹），可持久化、可切换。
// 此前各处直接 os.Getwd()（运行目录），互不统一、不持久 —— 本文件收口。

//go:build windows

package main

import (
	"os"
	"path/filepath"
)

// projectRoot 当前项目根目录（打开的文件夹）；空=未设，用运行目录兜底。
var projectRoot string

// currentRoot 返回当前项目根（文件树/终端/agent 统一用此）；未设则运行目录。
func currentRoot() string {
	if projectRoot != "" {
		return projectRoot
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// projectName 当前工作区显示名（项目根的目录名）。
func projectName() string {
	return filepath.Base(currentRoot())
}

// loadLastProject 启动时从配置恢复上次项目（须是有效目录）。
func loadLastProject() {
	if p := theSettings.LastProject; p != "" {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			projectRoot = p
		}
	}
}

// setProject 切换项目根：文件树/终端切过去、agent 用新根重建、持久化到配置。
func setProject(p string) {
	if p == "" {
		return
	}
	projectRoot = p
	theFileTree.setRoot(p)
	theTerminal.openDir(p)
	if theChatState != nil && theChatState.bridge != nil {
		theChatState.bridge.resetForNewRoot() // agent 下条消息用新根重建 loop
	}
	theSettings.LastProject = p
	settingsLoaded = true
	saveSettings()
}
