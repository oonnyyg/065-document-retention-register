package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
)

// Category 已登记的文档类别及其保留年数。
type Category struct {
	Name  string `json:"name"`
	Years int    `json:"years"`
}

// Document 已登记的文档。
type Document struct {
	ID          string   `json:"id"`
	Category    string   `json:"category"`
	StartDate   string   `json:"start_date"`
	Holds       []string `json:"holds"`
	Destroyed   bool     `json:"destroyed"`
	DestroyedOn string   `json:"destroyed_on,omitempty"`
	DestroyList string   `json:"destroy_list,omitempty"`
}

// DestroyList 预览生成的销毁清单。
type DestroyList struct {
	ID          string   `json:"id"`
	PreviewDate string   `json:"preview_date"`
	Documents   []string `json:"documents"`
	Confirmed   bool     `json:"confirmed"`
	ConfirmDate string   `json:"confirm_date,omitempty"`
}

// State 台账的完整持久化状态。
type State struct {
	Categories  map[string]*Category    `json:"categories"`
	Documents   map[string]*Document    `json:"documents"`
	Lists       map[string]*DestroyList `json:"lists"`
	ListCounter int                     `json:"list_counter"`
}

func newState() *State {
	return &State{
		Categories: map[string]*Category{},
		Documents:  map[string]*Document{},
		Lists:      map[string]*DestroyList{},
	}
}

// loadState 读取状态文件；文件不存在时返回空台账。
func loadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return newState(), nil
		}
		return nil, fmt.Errorf("读取状态文件失败: %w", err)
	}
	st := newState()
	if err := json.Unmarshal(data, st); err != nil {
		return nil, fmt.Errorf("解析状态文件失败: %w", err)
	}
	return st, nil
}

// save 将台账写回状态文件（先写临时文件再替换，避免中途损坏）。
func (st *State) save(path string) error {
	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("写入状态文件失败: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("替换状态文件失败: %w", err)
	}
	return nil
}

// sortedDocIDs 返回排序后的文档标识，保证输出稳定。
func (st *State) sortedDocIDs() []string {
	ids := make([]string, 0, len(st.Documents))
	for id := range st.Documents {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}
