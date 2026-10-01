package main

import (
	"fmt"
	"time"
)

// parseDate 解析并校验 YYYY-MM-DD 格式的日期。
func parseDate(s string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", s, time.UTC)
	if err != nil || t.Format("2006-01-02") != s {
		return time.Time{}, fmt.Errorf("非法日期 %q，要求格式 YYYY-MM-DD", s)
	}
	return t, nil
}

// addYears 计算起算日期加 n 年后的到期日。
// 二月二十九日遇目标非闰年时取二月二十八日。
func addYears(start time.Time, years int) time.Time {
	y, m, d := start.Date()
	target := time.Date(y+years, m, d, 0, 0, 0, 0, time.UTC)
	if m == time.February && d == 29 && target.Day() != 29 {
		target = time.Date(y+years, time.February, 28, 0, 0, 0, 0, time.UTC)
	}
	return target
}

// formatDate 以 YYYY-MM-DD 输出日期。
func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}
