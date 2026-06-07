// 工作区（与 VS Code 一致）：一个工作区 = 一组文件夹。
// - 打开文件夹：用单个文件夹替换当前工作区。
// - 添加文件夹到工作区：把文件夹加进来（变多根工作区）。
// - 从工作区移除文件夹：移除某个根。
// 文件树/终端/agent 统一用工作区；首个文件夹为「主文件夹」(currentRoot，agent/终端默认在此)。
// 整个工作区持久化到 config，启动恢复。此前各处直接 os.Getwd() 互不统一不持久 —— 本文件收口。

//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

// workspaceFolders 工作区的所有文件夹（多根，VS Code 模型）。空=未打开工作区，用运行目录兜底。
var workspaceFolders []string

// currentRoot 主文件夹（工作区首个；agent/终端默认在此）；未打开则运行目录。
func currentRoot() string {
	if len(workspaceFolders) > 0 {
		return workspaceFolders[0]
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// projectName 工作区显示名：单文件夹=文件夹名；多根=「工作区 (N)」；未打开=主文件夹名。
func projectName() string {
	if len(workspaceFolders) > 1 {
		return fmt.Sprintf("工作区 (%d 个文件夹)", len(workspaceFolders))
	}
	return filepath.Base(currentRoot())
}

// loadLastProject 启动恢复工作区（WorkspaceFolders；兼容旧 LastProject 单文件夹）。
func loadLastProject() {
	folders := theSettings.WorkspaceFolders
	if len(folders) == 0 && theSettings.LastProject != "" {
		folders = []string{theSettings.LastProject} // 迁移旧版
	}
	for _, p := range folders {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			workspaceFolders = append(workspaceFolders, p)
		}
	}
}

// setProject 打开文件夹：用单个文件夹替换整个工作区（VS Code「Open Folder」）。
func setProject(p string) {
	if p == "" {
		return
	}
	workspaceFolders = []string{p}
	syncWorkspace(true)
}

// addFolder 添加文件夹到工作区（VS Code「Add Folder to Workspace」），已在则忽略。
func addFolder(p string) {
	if p == "" {
		return
	}
	for _, f := range workspaceFolders {
		if f == p {
			return
		}
	}
	if len(workspaceFolders) == 0 { // 当前无显式工作区（用运行目录）→ 先纳入运行目录再加，避免“吞掉”原视图
		workspaceFolders = append(workspaceFolders, currentRoot())
		if workspaceFolders[0] == p { // 加的就是运行目录
			syncWorkspace(false)
			return
		}
	}
	workspaceFolders = append(workspaceFolders, p)
	syncWorkspace(false) // 加文件夹不重置 agent 主根（主文件夹未变）
}

// removeFolder 从工作区移除某文件夹。
func removeFolder(p string) {
	out := workspaceFolders[:0:0]
	for _, f := range workspaceFolders {
		if f != p {
			out = append(out, f)
		}
	}
	primaryChanged := len(out) == 0 || out[0] != currentRoot()
	workspaceFolders = out
	syncWorkspace(primaryChanged)
}

// syncWorkspace 工作区变更后同步：重建文件树、终端切主根、（主根变了才）重建 agent、持久化。
func syncWorkspace(primaryChanged bool) {
	theFileTree.rebuildRoots()
	theTerminal.openDir(currentRoot())
	if primaryChanged && theChatState != nil && theChatState.bridge != nil {
		theChatState.bridge.resetForNewRoot()
	}
	theSettings.WorkspaceFolders = append([]string{}, workspaceFolders...)
	theSettings.LastProject = currentRoot() // 兼容旧字段
	settingsLoaded = true
	saveSettings()
}
