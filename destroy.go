package main

import (
	"fmt"
)

// queryDocument 输出文档在指定日期的到期日与状态。
func queryDocument(st *State, cmd Command) (map[string]any, error) {
	doc, ok := st.Documents[cmd.Document]
	if !ok {
		return nil, fmt.Errorf("未知文档 %q", cmd.Document)
	}
	on, err := parseDate(cmd.Date)
	if err != nil {
		return nil, err
	}
	status, due, err := statusOn(st, doc, on)
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"ok":       true,
		"document": doc.ID,
		"date":     cmd.Date,
		"due_date": formatDate(due),
		"status":   status,
	}
	if len(doc.Holds) > 0 {
		out["holds"] = doc.Holds
	}
	if doc.Destroyed {
		out["destroyed_on"] = doc.DestroyedOn
		out["destroy_list"] = doc.DestroyList
	}
	return out, nil
}

// previewDestroy 生成该日期全部可销毁文档的清单并保存。
func previewDestroy(st *State, cmd Command) (map[string]any, bool, error) {
	on, err := parseDate(cmd.Date)
	if err != nil {
		return nil, false, err
	}
	eligible := []string{}
	for _, id := range st.sortedDocIDs() {
		status, _, err := statusOn(st, st.Documents[id], on)
		if err != nil {
			return nil, false, err
		}
		if status == StatusEligible {
			eligible = append(eligible, id)
		}
	}
	st.ListCounter++
	list := &DestroyList{
		ID:          fmt.Sprintf("LIST-%04d", st.ListCounter),
		PreviewDate: cmd.Date,
		Documents:   eligible,
	}
	st.Lists[list.ID] = list
	return map[string]any{
		"ok":           true,
		"list_id":      list.ID,
		"preview_date": cmd.Date,
		"documents":    eligible,
	}, true, nil
}

// confirmDestroy 按确认时台账逐项复查资格，全部通过后整批标记销毁。
func confirmDestroy(st *State, cmd Command) (map[string]any, bool, error) {
	list, ok := st.Lists[cmd.ListID]
	if !ok {
		return nil, false, fmt.Errorf("未知清单 %q", cmd.ListID)
	}
	if list.Confirmed {
		return nil, false, fmt.Errorf("清单 %q 已确认", cmd.ListID)
	}
	if len(list.Documents) == 0 {
		return nil, false, fmt.Errorf("清单 %q 为空，不能确认", cmd.ListID)
	}
	confirmDate, err := parseDate(cmd.Date)
	if err != nil {
		return nil, false, err
	}
	previewDate, err := parseDate(list.PreviewDate)
	if err != nil {
		return nil, false, err
	}
	if confirmDate.Before(previewDate) {
		return nil, false, fmt.Errorf("确认日期 %s 早于预览日期 %s", cmd.Date, list.PreviewDate)
	}
	// 按确认时台账逐项复查：已销毁或被保留均拒绝整批。
	for _, id := range list.Documents {
		doc, ok := st.Documents[id]
		if !ok {
			return nil, false, fmt.Errorf("清单中的文档 %q 不存在", id)
		}
		status, _, err := statusOn(st, doc, confirmDate)
		if err != nil {
			return nil, false, err
		}
		switch status {
		case StatusDestroyed:
			return nil, false, fmt.Errorf("文档 %q 已销毁，整批拒绝", id)
		case StatusHeld:
			return nil, false, fmt.Errorf("文档 %q 存在未解除的保留，整批拒绝", id)
		case StatusNotDue:
			return nil, false, fmt.Errorf("文档 %q 在确认日期尚未到期，整批拒绝", id)
		}
	}
	// 整批通过，统一提交销毁标记；不删除真实文件。
	for _, id := range list.Documents {
		doc := st.Documents[id]
		doc.Destroyed = true
		doc.DestroyedOn = cmd.Date
		doc.DestroyList = list.ID
	}
	list.Confirmed = true
	list.ConfirmDate = cmd.Date
	return map[string]any{
		"ok":           true,
		"list_id":      list.ID,
		"confirm_date": cmd.Date,
		"destroyed":    list.Documents,
	}, true, nil
}
