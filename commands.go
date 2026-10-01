package main

import (
	"fmt"
	"time"
)

// register_category: {"id","retention_years"}
func (s *State) registerCategory(cmd map[string]any) (any, error) {
	id, err := nonEmptyString(cmd, "id")
	if err != nil {
		return nil, err
	}
	if _, exists := s.Categories[id]; exists {
		return nil, fmt.Errorf("duplicate category id: %s", id)
	}
	years, err := positiveInt(cmd, "retention_years")
	if err != nil {
		return nil, err
	}
	s.Categories[id] = Category{Name: id, RetentionYears: years}
	return map[string]any{"ok": true, "id": id}, nil
}

// register_document: {"id","category","start_date"}
func (s *State) registerDocument(cmd map[string]any) (any, error) {
	id, err := nonEmptyString(cmd, "id")
	if err != nil {
		return nil, err
	}
	if _, exists := s.Documents[id]; exists {
		return nil, fmt.Errorf("duplicate document id: %s", id)
	}
	category, err := nonEmptyString(cmd, "category")
	if err != nil {
		return nil, err
	}
	if _, ok := s.Categories[category]; !ok {
		return nil, fmt.Errorf("unknown category: %s", category)
	}
	start, err := dateField(cmd, "start_date")
	if err != nil {
		return nil, err
	}
	s.Documents[id] = Document{
		Category:  category,
		StartDate: start.Format(dateLayout),
		Holds:     map[string]bool{},
	}
	return map[string]any{"ok": true, "id": id}, nil
}

// add_hold / release_hold: {"document_id","hold"}
func (s *State) applyHold(cmd map[string]any, release bool) (any, error) {
	docID, err := nonEmptyString(cmd, "document_id")
	if err != nil {
		return nil, err
	}
	doc, ok := s.Documents[docID]
	if !ok {
		return nil, fmt.Errorf("unknown document: %s", docID)
	}
	if doc.Destroyed {
		return nil, fmt.Errorf("document already destroyed: %s", docID)
	}
	hold, err := nonEmptyString(cmd, "hold")
	if err != nil {
		return nil, err
	}
	if release {
		if !doc.Holds[hold] {
			return nil, fmt.Errorf("hold not present: %s", hold)
		}
		delete(doc.Holds, hold)
	} else {
		if doc.Holds[hold] {
			return nil, fmt.Errorf("duplicate hold: %s", hold)
		}
		doc.Holds[hold] = true
	}
	s.Documents[docID] = doc
	return map[string]any{"ok": true, "document_id": docID, "hold": hold}, nil
}

func nonEmptyString(cmd map[string]any, key string) (string, error) {
	v, ok := cmd[key]
	if !ok {
		return "", fmt.Errorf("missing field: %s", key)
	}
	str, ok := v.(string)
	if !ok || str == "" {
		return "", fmt.Errorf("field must be a non-empty string: %s", key)
	}
	return str, nil
}

func positiveInt(cmd map[string]any, key string) (int, error) {
	v, ok := cmd[key]
	if !ok {
		return 0, fmt.Errorf("missing field: %s", key)
	}
	f, ok := v.(float64)
	if !ok || f <= 0 || f != float64(int(f)) {
		return 0, fmt.Errorf("field must be a positive integer: %s", key)
	}
	return int(f), nil
}

func dateField(cmd map[string]any, key string) (time.Time, error) {
	str, err := nonEmptyString(cmd, key)
	if err != nil {
		return time.Time{}, err
	}
	t, err := parseDate(str)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid date for %s: %s", key, str)
	}
	return t, nil
}
