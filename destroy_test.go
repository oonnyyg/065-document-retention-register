package main

import "testing"

func preview(t *testing.T, st *State, date string) string {
	t.Helper()
	out, _, err := previewDestroy(st, Command{Operation: "preview", Date: date})
	if err != nil {
		t.Fatalf("预览失败: %v", err)
	}
	return out["list_id"].(string)
}

func TestPreviewAndConfirm(t *testing.T) {
	st := setup(t)
	mustOK(t, run(st, Command{Operation: "register-document", Document: "D2", Category: "合同", StartDate: "2021-06-01"}))
	listID := preview(t, st, "2023-01-15")
	if listID != "LIST-0001" {
		t.Fatalf("清单编号错误: %s", listID)
	}
	if got := len(st.Lists[listID].Documents); got != 1 {
		t.Fatalf("清单应只含 D1，实际 %d 项", got)
	}
	// 确认日期早于预览日期应拒绝。
	mustErr(t, run(st, Command{Operation: "confirm", ListID: listID, Date: "2023-01-14"}))
	out, _, err := confirmDestroy(st, Command{Operation: "confirm", ListID: listID, Date: "2023-01-20"})
	if err != nil {
		t.Fatalf("确认失败: %v", err)
	}
	if out["confirm_date"] != "2023-01-20" {
		t.Fatalf("确认日期未保存: %v", out)
	}
	// 已销毁仍可查询且禁止修改。
	if got := query(t, st, "D1", "2023-01-20")["status"]; got != StatusDestroyed {
		t.Fatalf("应为已销毁，得到 %v", got)
	}
	mustErr(t, run(st, Command{Operation: "add-hold", Document: "D1", Hold: "h"}))
	// 重复确认应拒绝。
	mustErr(t, run(st, Command{Operation: "confirm", ListID: listID, Date: "2023-01-21"}))
}

func TestConfirmRejectsNewHoldAfterPreview(t *testing.T) {
	st := setup(t)
	listID := preview(t, st, "2023-01-15")
	mustOK(t, run(st, Command{Operation: "add-hold", Document: "D1", Hold: "临时保留"}))
	mustErr(t, run(st, Command{Operation: "confirm", ListID: listID, Date: "2023-01-16"}))
	if st.Documents["D1"].Destroyed {
		t.Fatal("整批拒绝后不应标记销毁")
	}
}

func TestPreviewExcludesOutsiders(t *testing.T) {
	st := setup(t)
	// D2 在预览之后才登记并到期，不能混入既有清单。
	listID := preview(t, st, "2023-01-15")
	mustOK(t, run(st, Command{Operation: "register-document", Document: "D2", Category: "合同", StartDate: "2020-01-01"}))
	out, _, err := confirmDestroy(st, Command{Operation: "confirm", ListID: listID, Date: "2023-01-15"})
	if err != nil {
		t.Fatalf("确认失败: %v", err)
	}
	destroyed := out["destroyed"].([]string)
	if len(destroyed) != 1 || destroyed[0] != "D1" {
		t.Fatalf("清单外的文档不能混入确认: %v", destroyed)
	}
	if st.Documents["D2"].Destroyed {
		t.Fatal("D2 不应被销毁")
	}
}

func TestEmptyListCannotConfirm(t *testing.T) {
	st := setup(t)
	listID := preview(t, st, "2020-01-01") // 全部未到期，清单为空
	mustErr(t, run(st, Command{Operation: "confirm", ListID: listID, Date: "2020-01-01"}))
}

func TestUnknownList(t *testing.T) {
	st := setup(t)
	mustErr(t, run(st, Command{Operation: "confirm", ListID: "LIST-9999", Date: "2023-01-15"}))
}
