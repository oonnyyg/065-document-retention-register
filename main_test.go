package main

import "testing"

func TestExpiryLeapDay(t *testing.T) {
	leap, _ := parseDate("2020-02-29")
	if got := expiryDate(leap, 1).Format(dateLayout); got != "2021-02-28" {
		t.Fatalf("expected 2021-02-28, got %s", got)
	}
	if got := expiryDate(leap, 4).Format(dateLayout); got != "2024-02-29" {
		t.Fatalf("expected 2024-02-29, got %s", got)
	}
	normal, _ := parseDate("2019-06-15")
	if got := expiryDate(normal, 3).Format(dateLayout); got != "2022-06-15" {
		t.Fatalf("expected 2022-06-15, got %s", got)
	}
}

func newTestState() *State {
	s := newState()
	s.Categories["tax"] = Category{Name: "tax", RetentionYears: 3}
	s.Documents["d1"] = Document{
		Category:  "tax",
		StartDate: "2020-01-01",
		Holds:     map[string]bool{},
	}
	s.Documents["d2"] = Document{
		Category:  "tax",
		StartDate: "2024-01-01",
		Holds:     map[string]bool{},
	}
	return s
}

func TestStatusPriority(t *testing.T) {
	s := newTestState()
	on, _ := parseDate("2025-01-01")

	d1 := s.Documents["d1"]
	if got := s.statusOn(d1, on); got != "eligible" {
		t.Fatalf("d1 expected eligible, got %s", got)
	}

	d1.Holds["audit"] = true
	if got := s.statusOn(d1, on); got != "held" {
		t.Fatalf("d1 expected held, got %s", got)
	}

	d1.Holds = map[string]bool{}
	d1.Destroyed = true
	d1.Holds["audit"] = true
	if got := s.statusOn(d1, on); got != "destroyed" {
		t.Fatalf("destroyed outranks held, got %s", got)
	}

	d2 := s.Documents["d2"]
	if got := s.statusOn(d2, on); got != "not_due" {
		t.Fatalf("d2 expected not_due, got %s", got)
	}
}

func TestExpiryDayIsEligible(t *testing.T) {
	s := newTestState()
	exact, _ := parseDate("2023-01-01")
	if got := s.statusOn(s.Documents["d1"], exact); got != "eligible" {
		t.Fatalf("expected eligible exactly on expiry day, got %s", got)
	}
}

func TestPreviewAndConfirm(t *testing.T) {
	s := newTestState()
	args := map[string]any{"date": "2025-01-01"}

	res, err := s.previewDestroy(args)
	if err != nil {
		t.Fatal(err)
	}
	rm := res.(map[string]any)
	if rm["list_id"] != "L0001" {
		t.Fatalf("unexpected list id %v", rm["list_id"])
	}
	ids := rm["document_ids"].([]string)
	if len(ids) != 1 || ids[0] != "d1" {
		t.Fatalf("preview should contain only d1, got %v", ids)
	}

	// A hold placed after preview rejects the batch.
	doc := s.Documents["d1"]
	doc.Holds["court"] = true
	s.Documents["d1"] = doc
	_, err = s.confirmDestroy(map[string]any{"list_id": "L0001", "confirm_date": "2025-02-01"})
	if err == nil {
		t.Fatal("expected batch rejection due to hold")
	}
	if s.Documents["d1"].Destroyed {
		t.Fatal("failed confirm must not mutate state")
	}

	// Releasing the hold allows confirmation.
	doc = s.Documents["d1"]
	delete(doc.Holds, "court")
	s.Documents["d1"] = doc
	_, err = s.confirmDestroy(map[string]any{"list_id": "L0001", "confirm_date": "2025-02-01"})
	if err != nil {
		t.Fatal(err)
	}
	if !s.Documents["d1"].Destroyed {
		t.Fatal("d1 should be destroyed")
	}

	// Same list cannot be confirmed twice.
	_, err = s.confirmDestroy(map[string]any{"list_id": "L0001", "confirm_date": "2025-03-01"})
	if err == nil {
		t.Fatal("expected double-confirm rejection")
	}
}

func TestConfirmDateBeforePreview(t *testing.T) {
	s := newTestState()
	s.previewDestroy(map[string]any{"date": "2025-01-01"})
	_, err := s.confirmDestroy(map[string]any{"list_id": "L0001", "confirm_date": "2024-12-31"})
	if err == nil {
		t.Fatal("expected date-order rejection")
	}
}

func TestDuplicatesAndUnknowns(t *testing.T) {
	s := newTestState()
	_, err := s.registerCategory(map[string]any{"id": "tax", "retention_years": 5.0})
	if err == nil {
		t.Fatal("expected duplicate category rejection")
	}
	_, err = s.registerDocument(map[string]any{"id": "d1", "category": "tax", "start_date": "2020-01-01"})
	if err == nil {
		t.Fatal("expected duplicate document rejection")
	}
	_, err = s.registerDocument(map[string]any{"id": "d3", "category": "nope", "start_date": "2020-01-01"})
	if err == nil {
		t.Fatal("expected unknown category rejection")
	}
	_, err = s.registerCategory(map[string]any{"id": "c2", "retention_years": 0.0})
	if err == nil {
		t.Fatal("expected non-positive retention rejection")
	}
	_, err = s.registerDocument(map[string]any{"id": "d3", "category": "tax", "start_date": "2020/01/01"})
	if err == nil {
		t.Fatal("expected invalid date rejection")
	}
}

func TestDestroyedDocumentImmutable(t *testing.T) {
	s := newTestState()
	doc := s.Documents["d1"]
	doc.Destroyed = true
	s.Documents["d1"] = doc
	_, err := s.applyHold(map[string]any{"document_id": "d1", "hold": "h"}, false)
	if err == nil {
		t.Fatal("destroyed documents cannot be modified")
	}
}

func TestMultipleHolds(t *testing.T) {
	s := newTestState()
	on, _ := parseDate("2025-01-01")
	doc := s.Documents["d1"]
	doc.Holds["a"] = true
	doc.Holds["b"] = true
	s.Documents["d1"] = doc
	if s.statusOn(doc, on) != "held" {
		t.Fatal("expected held with two holds")
	}
	s.applyHold(map[string]any{"document_id": "d1", "hold": "a"}, true)
	if s.statusOn(s.Documents["d1"], on) != "held" {
		t.Fatal("one remaining hold keeps it held")
	}
	s.applyHold(map[string]any{"document_id": "d1", "hold": "b"}, true)
	if s.statusOn(s.Documents["d1"], on) != "eligible" {
		t.Fatal("releasing all holds restores eligibility")
	}
}
