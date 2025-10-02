package app

import "testing"

func TestPopulateMinFilterErrors(t *testing.T) {
	var state filterState
	if err := populateMinFilter(&state, "-1"); err == nil {
		t.Fatal("expected error for negative minutes")
	}
	if err := populateMinFilter(&state, "abc"); err == nil {
		t.Fatal("expected error for invalid integer")
	}
	if err := populateMinFilter(&state, "10"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.minEnabled || state.minMinutes != 10 {
		t.Fatalf("expected state updated, got %+v", state)
	}
}

func TestPopulateMaxFilterErrors(t *testing.T) {
	var state filterState
	if err := populateMaxFilter(&state, "-1"); err == nil {
		t.Fatal("expected error for negative minutes")
	}
	if err := populateMaxFilter(&state, "abc"); err == nil {
		t.Fatal("expected error for invalid integer")
	}
	if err := populateMaxFilter(&state, "20"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !state.maxEnabled || state.maxMinutes != 20 {
		t.Fatalf("expected state updated, got %+v", state)
	}
}
