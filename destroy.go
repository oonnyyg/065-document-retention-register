package main

import "fmt"

// preview_destroy: {"date"}
// Saves a list with a unique ID containing all eligible documents.
func (s *State) previewDestroy(cmd map[string]any) (any, error) {
	on, err := dateField(cmd, "date")
	if err != nil {
		return nil, err
	}
	ids := s.eligibleIDs(on)
	if len(ids) == 0 {
		return nil, fmt.Errorf("no eligible documents on %s", on.Format(dateLayout))
	}
	id := fmt.Sprintf("L%04d", s.NextListNo)
	s.NextListNo++
	s.Lists[id] = List{
		PreviewDate: on.Format(dateLayout),
		DocumentIDs: ids,
	}
	s.ListIDs = append(s.ListIDs, id)
	return map[string]any{
		"ok":           true,
		"list_id":      id,
		"preview_date": on.Format(dateLayout),
		"document_ids": ids,
		"count":        len(ids),
	}, nil
}

// confirm_destroy: {"list_id","confirm_date"}
func (s *State) confirmDestroy(cmd map[string]any) (any, error) {
	listID, err := nonEmptyString(cmd, "list_id")
	if err != nil {
		return nil, err
	}
	list, ok := s.Lists[listID]
	if !ok {
		return nil, fmt.Errorf("unknown list: %s", listID)
	}
	if list.Confirmed {
		return nil, fmt.Errorf("list already confirmed: %s", listID)
	}
	confirmOn, err := dateField(cmd, "confirm_date")
	if err != nil {
		return nil, err
	}
	previewOn, _ := parseDate(list.PreviewDate)
	if confirmOn.Before(previewOn) {
		return nil, fmt.Errorf("confirm_date %s is before preview_date %s",
			confirmOn.Format(dateLayout), list.PreviewDate)
	}
	if len(list.DocumentIDs) == 0 {
		return nil, fmt.Errorf("empty list cannot be confirmed")
	}

	// Re-check every item against the current register. Any item that is
	// destroyed or held rejects the whole batch.
	for _, id := range list.DocumentIDs {
		doc, ok := s.Documents[id]
		if !ok {
			return nil, fmt.Errorf("document vanished from register: %s", id)
		}
		status := s.statusOn(doc, confirmOn)
		if status == "destroyed" {
			return nil, fmt.Errorf("document already destroyed: %s", id)
		}
		if status == "held" {
			return nil, fmt.Errorf("document is under legal hold: %s", id)
		}
		if status == "not_due" {
			return nil, fmt.Errorf("document is not due: %s", id)
		}
	}

	confirmDate := confirmOn.Format(dateLayout)
	for _, id := range list.DocumentIDs {
		doc := s.Documents[id]
		doc.Destroyed = true
		doc.DestroyedAt = confirmDate
		s.Documents[id] = doc
	}
	list.Confirmed = true
	list.ConfirmDate = confirmDate
	s.Lists[listID] = list

	return map[string]any{
		"ok":           true,
		"list_id":      listID,
		"confirm_date": confirmDate,
		"document_ids": list.DocumentIDs,
		"count":        len(list.DocumentIDs),
	}, nil
}
