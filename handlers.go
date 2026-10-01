package main

import (
	"fmt"
	"time"
)

// 文档状态，按优先级：已销毁 > 被保留 > 未到期 > 可销毁。
const (
	StatusDestroyed = "已销毁"
	StatusHeld      = "被保留"
	StatusNotDue    = "未到期"
	StatusEligible  = "可销毁"
)

// dueDate 计算文档的到期日。
func dueDate(st *State, doc *Document) (time.Time, error) {
	cat, ok := st.Categories[doc.Category]
	if !ok {
		return time.Time{}, fmt.Errorf("文档 %q 关联未知类别 %q", doc.ID, doc.Category)
	}
	start, err := parseDate(doc.StartDate)
	if err != nil {
		return time.Time{}, err
	}
	return addYears(start, cat.Years), nil
}

// statusOn 计算文档在指定日期的状态。
func statusOn(st *State, doc *Document, on time.Time) (string, time.Time, error) {
	due, err := dueDate(st, doc)
	if err != nil {
		return "", time.Time{}, err
	}
	switch {
	case doc.Destroyed:
		return StatusDestroyed, due, nil
	case len(doc.Holds) > 0:
		return StatusHeld, due, nil
	case on.Before(due):
		return StatusNotDue, due, nil
	default:
		return StatusEligible, due, nil
	}
}

// Command 单次操作的输入结构。
type Command struct {
	Operation string `json:"operation"`
	Category  string `json:"category,omitempty"`
	Years     int    `json:"years,omitempty"`
	Document  string `json:"document,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	Hold      string `json:"hold,omitempty"`
	Date      string `json:"date,omitempty"`
	ListID    string `json:"list_id,omitempty"`
}

// Execute 执行一次操作；返回的 map 将被序列化为输出 JSON。
// 返回错误时调用方不得保存状态。
func Execute(st *State, cmd Command) (map[string]any, bool, error) {
	switch cmd.Operation {
	case "register-category":
		return registerCategory(st, cmd)
	case "register-document":
		return registerDocument(st, cmd)
	case "add-hold":
		return addHold(st, cmd)
	case "remove-hold":
		return removeHold(st, cmd)
	case "query":
		out, err := queryDocument(st, cmd)
		return out, false, err
	case "preview":
		return previewDestroy(st, cmd)
	case "confirm":
		return confirmDestroy(st, cmd)
	default:
		return nil, false, fmt.Errorf("未知操作 %q", cmd.Operation)
	}
}

func registerCategory(st *State, cmd Command) (map[string]any, bool, error) {
	if cmd.Category == "" {
		return nil, false, fmt.Errorf("缺少类别名称")
	}
	if cmd.Years <= 0 {
		return nil, false, fmt.Errorf("保留年数必须为正整数")
	}
	if _, ok := st.Categories[cmd.Category]; ok {
		return nil, false, fmt.Errorf("类别 %q 已登记", cmd.Category)
	}
	st.Categories[cmd.Category] = &Category{Name: cmd.Category, Years: cmd.Years}
	return map[string]any{"ok": true, "category": cmd.Category, "years": cmd.Years}, true, nil
}

func registerDocument(st *State, cmd Command) (map[string]any, bool, error) {
	if cmd.Document == "" {
		return nil, false, fmt.Errorf("缺少文档标识")
	}
	if _, ok := st.Categories[cmd.Category]; !ok {
		return nil, false, fmt.Errorf("未知类别 %q", cmd.Category)
	}
	if _, err := parseDate(cmd.StartDate); err != nil {
		return nil, false, err
	}
	if _, ok := st.Documents[cmd.Document]; ok {
		return nil, false, fmt.Errorf("文档 %q 已登记", cmd.Document)
	}
	st.Documents[cmd.Document] = &Document{
		ID: cmd.Document, Category: cmd.Category,
		StartDate: cmd.StartDate, Holds: []string{},
	}
	return map[string]any{"ok": true, "document": cmd.Document}, true, nil
}

func mutableDoc(st *State, id string) (*Document, error) {
	doc, ok := st.Documents[id]
	if !ok {
		return nil, fmt.Errorf("未知文档 %q", id)
	}
	if doc.Destroyed {
		return nil, fmt.Errorf("文档 %q 已销毁，禁止修改", id)
	}
	return doc, nil
}

func addHold(st *State, cmd Command) (map[string]any, bool, error) {
	if cmd.Hold == "" {
		return nil, false, fmt.Errorf("缺少保留名称")
	}
	doc, err := mutableDoc(st, cmd.Document)
	if err != nil {
		return nil, false, err
	}
	for _, h := range doc.Holds {
		if h == cmd.Hold {
			return nil, false, fmt.Errorf("文档 %q 已存在保留 %q", cmd.Document, cmd.Hold)
		}
	}
	doc.Holds = append(doc.Holds, cmd.Hold)
	return map[string]any{"ok": true, "document": cmd.Document, "holds": doc.Holds}, true, nil
}

func removeHold(st *State, cmd Command) (map[string]any, bool, error) {
	doc, err := mutableDoc(st, cmd.Document)
	if err != nil {
		return nil, false, err
	}
	for i, h := range doc.Holds {
		if h == cmd.Hold {
			doc.Holds = append(doc.Holds[:i], doc.Holds[i+1:]...)
			return map[string]any{"ok": true, "document": cmd.Document, "holds": doc.Holds}, true, nil
		}
	}
	return nil, false, fmt.Errorf("文档 %q 不存在保留 %q", cmd.Document, cmd.Hold)
}
