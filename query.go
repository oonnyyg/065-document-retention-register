package main

import (
	"fmt"
	"sort"
	"time"
)

// query: {"document_id","date"}
func (s *State) query(cmd map[string]any) (any, error) {
	docID, err := nonEmptyString(cmd, "document_id")
	if err != nil {
		return nil, err
	}
	doc, ok := s.Documents[docID]
	if !ok {
		return nil, fmt.Errorf("unknown document: %s", docID)
	}
	on, err := dateField(cmd, "date")
	if err != nil {
		return nil, err
	}
	cat := s.Categories[doc.Category]
	start, _ := parseDate(doc.StartDate)
	expiry := expiryDate(start, cat.RetentionYears)
	holds := activeHolds(doc)
	status := s.statusOn(doc, on)
	return map[string]any{
		"document_id": docID,
		"category":    doc.Category,
		"start_date":  doc.StartDate,
		"expiry_date": expiry.Format(dateLayout),
		"holds":       holds,
		"destroyed":   doc.Destroyed,
		"status":      status,
	}, nil
}

func activeHolds(doc Document) []string {
	holds := make([]string, 0, len(doc.Holds))
	for h := range doc.Holds {
		holds = append(holds, h)
	}
	sort.Strings(holds)
	return holds
}

// eligibleIDs returns all eligible documents as of on, sorted by ID.
func (s *State) eligibleIDs(on time.Time) []string {
	var ids []string
	for id, doc := range s.Documents {
		if s.statusOn(doc, on) == "eligible" {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}
