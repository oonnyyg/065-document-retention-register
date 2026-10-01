package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func fail(err error) {
	out, _ := json.Marshal(map[string]any{"ok": false, "error": err.Error()})
	fmt.Println(string(out))
	os.Exit(1)
}

func main() {
	statePath := flag.String("state", "", "状态文件路径")
	inputPath := flag.String("input", "", "命令 JSON 文件路径")
	flag.Parse()
	if *statePath == "" || *inputPath == "" {
		fail(fmt.Errorf("必须提供 --state 与 --input 参数"))
	}

	data, err := os.ReadFile(*inputPath)
	if err != nil {
		fail(fmt.Errorf("读取命令文件失败: %w", err))
	}
	var cmd Command
	if err := json.Unmarshal(data, &cmd); err != nil {
		fail(fmt.Errorf("解析命令 JSON 失败: %w", err))
	}

	st, err := loadState(*statePath)
	if err != nil {
		fail(err)
	}
	out, changed, err := Execute(st, cmd)
	if err != nil {
		fail(err) // 失败不改状态
	}
	if changed {
		if err := st.save(*statePath); err != nil {
			fail(err)
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(out)
}
