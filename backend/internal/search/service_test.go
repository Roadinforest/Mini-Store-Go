package search

import "testing"

func TestRankCandidatesOrdersByRRFScore(t *testing.T) {
	scores := map[string]float64{
		"p1": 0.01,
		"p2": 0.03,
		"p3": 0.02,
	}

	got := rankCandidates(scores, 2)
	if len(got) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(got))
	}
	if got[0].ID != "p2" || got[1].ID != "p3" {
		t.Fatalf("unexpected order: %#v", got)
	}
}
