package app

import "testing"

func TestLoadProgressLifecycle(t *testing.T) {
	var progress loadProgress
	progress.SetTotal(5)
	for i := 0; i < 3; i++ {
		progress.Increment()
	}
	processed, total, done := progress.Snapshot()
	if processed != 3 || total != 5 || done {
		t.Fatalf("unexpected snapshot %d/%d done=%v", processed, total, done)
	}
	progress.MarkDone()
	_, _, done = progress.Snapshot()
	if !done {
		t.Fatal("expected done")
	}
	progress.Reset()
	processed, total, done = progress.Snapshot()
	if processed != 0 || total != 0 || done {
		t.Fatalf("expected reset to zero, got %d/%d done=%v", processed, total, done)
	}
}
