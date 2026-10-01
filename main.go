package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type request struct {
	Command string         `json:"command"`
	Args    map[string]any `json:"args"`
}

func main() {
	statePath := flag.String("state", "", "path to the state file")
	inputPath := flag.String("input", "", "path to the command JSON file")
	flag.Parse()

	if *statePath == "" || *inputPath == "" {
		fail("both --state and --input are required")
	}

	state, err := loadState(*statePath)
	if err != nil {
		fail(err.Error())
	}

	raw, err := os.ReadFile(*inputPath)
	if err != nil {
		fail(err.Error())
	}
	raw = trimBOM(raw)
	var req request
	if err := json.Unmarshal(raw, &req); err != nil {
		fail("invalid command JSON: " + err.Error())
	}

	result, err := dispatch(state, req)
	if err != nil {
		// Failure: state is not modified on disk.
		output(map[string]any{"ok": false, "error": err.Error()})
		os.Exit(1)
	}

	if err := saveState(*statePath, state); err != nil {
		fail(err.Error())
	}
	output(result)
}

func dispatch(s *State, req request) (any, error) {
	switch req.Command {
	case "register_category":
		return s.registerCategory(req.Args)
	case "register_document":
		return s.registerDocument(req.Args)
	case "add_hold":
		return s.applyHold(req.Args, false)
	case "release_hold":
		return s.applyHold(req.Args, true)
	case "query":
		return s.query(req.Args)
	case "preview_destroy":
		return s.previewDestroy(req.Args)
	case "confirm_destroy":
		return s.confirmDestroy(req.Args)
	default:
		return nil, fmt.Errorf("unknown command: %s", req.Command)
	}
}

func loadState(path string) (*State, error) {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return newState(), nil
	}
	if err != nil {
		return nil, err
	}
	raw = trimBOM(raw)
	s := newState()
	if err := json.Unmarshal(raw, s); err != nil {
		return nil, fmt.Errorf("invalid state file: %w", err)
	}
	if s.Categories == nil {
		s.Categories = map[string]Category{}
	}
	if s.Documents == nil {
		s.Documents = map[string]Document{}
	}
	if s.Lists == nil {
		s.Lists = map[string]List{}
	}
	if s.NextListNo == 0 {
		s.NextListNo = 1
	}
	return s, nil
}

// trimBOM removes a UTF-8 byte order mark if present.
func trimBOM(b []byte) []byte {
	if len(b) >= 3 && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		return b[3:]
	}
	return b
}

func saveState(path string, s *State) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func output(v any) {
	data, _ := json.MarshalIndent(v, "", "  ")
	fmt.Println(string(data))
}

func fail(msg string) {
	output(map[string]any{"ok": false, "error": msg})
	os.Exit(1)
}
