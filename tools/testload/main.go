package main

import (
	"encoding/json"
	"fmt"
	"os"
)

type Test struct {
	Version int `json:"version"`
}

func main() {
	data, err := os.ReadFile("F:\\syproject\\gou-ide\\.pair\\conversations\\history.json")
	if err != nil {
		fmt.Printf("读文件失败: %v\n", err)
		return
	}
	fmt.Printf("大小: %d MB\n", len(data)/1024/1024)
	fmt.Printf("前80字节: %q\n", string(data[:80]))

	// Just try full parse
	var v Test
	if err := json.Unmarshal(data, &v); err != nil {
		fmt.Printf("全文解析失败: %v\n", err)
	} else {
		fmt.Printf("全文解析成功, version=%d\n", v.Version)
	}
}
