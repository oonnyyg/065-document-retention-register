package main

import "time"

const dateLayout = "2006-01-02"

// parseDate accepts strictly YYYY-MM-DD calendar dates.
func parseDate(s string) (time.Time, error) {
	t, err := time.Parse(dateLayout, s)
	if err != nil {
		return time.Time{}, err
	}
	return t, nil
}

// expiryDate adds years to the start date. If the result is Feb 29 in a
// non-leap year, it falls back to Feb 28.
func expiryDate(start time.Time, years int) time.Time {
	y, m, d := start.Date()
	loc := start.Location()
	if m == time.February && d == 29 && !isLeap(y+years) {
		return time.Date(y+years, time.February, 28, 0, 0, 0, 0, loc)
	}
	return time.Date(y+years, m, d, 0, 0, 0, 0, loc)
}

func isLeap(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

// statusOn computes the document status as of the given day.
// Priority: destroyed > held > not_due > eligible.
func (s *State) statusOn(doc Document, on time.Time) string {
	if doc.Destroyed {
		return "destroyed"
	}
	if len(doc.Holds) > 0 {
		return "held"
	}
	cat := s.Categories[doc.Category]
	start, _ := parseDate(doc.StartDate)
	if on.Before(expiryDate(start, cat.RetentionYears)) {
		return "not_due"
	}
	return "eligible"
}
