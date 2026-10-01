package main

import "testing"

func TestAddYearsLeapDay(t *testing.T) {
	start, _ := parseDate("2020-02-29")
	if got := formatDate(addYears(start, 1)); got != "2021-02-28" {
		t.Fatalf("非闰年应取 2 月 28 日，得到 %s", got)
	}
	if got := formatDate(addYears(start, 4)); got != "2024-02-29" {
		t.Fatalf("闰年应保留 2 月 29 日，得到 %s", got)
	}
}

func TestParseDateRejectsInvalid(t *testing.T) {
	for _, s := range []string{"2021-13-01", "2021-02-29", "2021/01/01", "x"} {
		if _, err := parseDate(s); err == nil {
			t.Fatalf("应拒绝非法日期 %q", s)
		}
	}
}
