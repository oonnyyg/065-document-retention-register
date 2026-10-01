package main

import "testing"

// run 执行操作并只返回错误，便于断言。
func run(st *State, cmd Command) error {
	_, _, err := Execute(st, cmd)
	return err
}

func mustOK(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("操作失败: %v", err)
	}
}

func mustErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("应失败但成功")
	}
}

func setup(t *testing.T) *State {
	t.Helper()
	st := newState()
	mustOK(t, run(st, Command{Operation: "register-category", Category: "合同", Years: 3}))
	mustOK(t, run(st, Command{Operation: "register-document", Document: "D1", Category: "合同", StartDate: "2020-01-15"}))
	return st
}

func query(t *testing.T, st *State, doc, date string) map[string]any {
	t.Helper()
	out, err := queryDocument(st, Command{Operation: "query", Document: doc, Date: date})
	if err != nil {
		t.Fatalf("查询失败: %v", err)
	}
	return out
}

func TestDuplicateAndUnknown(t *testing.T) {
	st := setup(t)
	mustErr(t, run(st, Command{Operation: "register-category", Category: "合同", Years: 1}))
	mustErr(t, run(st, Command{Operation: "register-document", Document: "D1", Category: "合同", StartDate: "2020-01-15"}))
	mustErr(t, run(st, Command{Operation: "register-document", Document: "D2", Category: "未知", StartDate: "2020-01-15"}))
	mustErr(t, run(st, Command{Operation: "register-category", Category: "零年", Years: 0}))
	mustErr(t, run(st, Command{Operation: "add-hold", Document: "不存在", Hold: "h"}))
}

func TestStatusTransitions(t *testing.T) {
	st := setup(t)
	if got := query(t, st, "D1", "2023-01-14")["status"]; got != StatusNotDue {
		t.Fatalf("到期前一天应为未到期，得到 %v", got)
	}
	if got := query(t, st, "D1", "2023-01-15")["status"]; got != StatusEligible {
		t.Fatalf("到期日当天应可销毁，得到 %v", got)
	}
	if got := query(t, st, "D1", "2023-01-15")["due_date"]; got != "2023-01-15" {
		t.Fatalf("到期日错误: %v", got)
	}
}

func TestMultipleHolds(t *testing.T) {
	st := setup(t)
	mustOK(t, run(st, Command{Operation: "add-hold", Document: "D1", Hold: "诉讼A"}))
	mustOK(t, run(st, Command{Operation: "add-hold", Document: "D1", Hold: "审计B"}))
	mustErr(t, run(st, Command{Operation: "add-hold", Document: "D1", Hold: "诉讼A"}))
	if got := query(t, st, "D1", "2024-01-01")["status"]; got != StatusHeld {
		t.Fatalf("有保留时应为被保留，得到 %v", got)
	}
	mustOK(t, run(st, Command{Operation: "remove-hold", Document: "D1", Hold: "诉讼A"}))
	if got := query(t, st, "D1", "2024-01-01")["status"]; got != StatusHeld {
		t.Fatalf("解除一个保留后仍应被保留，得到 %v", got)
	}
	mustOK(t, run(st, Command{Operation: "remove-hold", Document: "D1", Hold: "审计B"}))
	if got := query(t, st, "D1", "2024-01-01")["status"]; got != StatusEligible {
		t.Fatalf("全部解除后应恢复可销毁，得到 %v", got)
	}
	mustErr(t, run(st, Command{Operation: "remove-hold", Document: "D1", Hold: "审计B"}))
}
