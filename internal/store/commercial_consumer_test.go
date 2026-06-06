package store

import "testing"

func TestStringField(t *testing.T) {
	data := map[string]any{"deal_id": "D-1", "amount": float64(120000)}
	if got := stringField(data, "deal_id"); got != "D-1" {
		t.Fatalf("got %q", got)
	}
	if got := stringField(data, "amount"); got != "120000" {
		t.Fatalf("got %q", got)
	}
}
