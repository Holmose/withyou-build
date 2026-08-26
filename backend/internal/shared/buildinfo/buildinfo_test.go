package buildinfo

import "testing"

func TestSnapshotUsesWithYouProductName(t *testing.T) {
	if got := Snapshot().Product; got != "WithYou" {
		t.Fatalf("expected product WithYou, got %q", got)
	}
}
