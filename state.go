package main

// State is the retention register persisted as JSON.
type State struct {
	Categories map[string]Category `json:"categories"`
	Documents  map[string]Document `json:"documents"`
	// ListIDs keeps insertion order of preview lists.
	ListIDs    []string        `json:"list_ids"`
	Lists      map[string]List `json:"lists"`
	NextListNo int             `json:"next_list_no"`
}

// Category has a retention period in whole years.
type Category struct {
	Name           string `json:"name"`
	RetentionYears int    `json:"retention_years"`
}

// Document belongs to a category; holds are named legal holds.
type Document struct {
	Category    string          `json:"category"`
	StartDate   string          `json:"start_date"`
	Holds       map[string]bool `json:"holds"`
	Destroyed   bool            `json:"destroyed"`
	DestroyedAt string          `json:"destroyed_at,omitempty"`
}

// List is a preview list, confirmed or not.
type List struct {
	PreviewDate string   `json:"preview_date"`
	DocumentIDs []string `json:"document_ids"`
	Confirmed   bool     `json:"confirmed"`
	ConfirmDate string   `json:"confirm_date,omitempty"`
}

func newState() *State {
	return &State{
		Categories: map[string]Category{},
		Documents:  map[string]Document{},
		Lists:      map[string]List{},
		NextListNo: 1,
	}
}
