package store

import (
	"strings"
	"testing"
)

// The signal insert ends in ON CONFLICT (id) DO NOTHING, which read as
// idempotent but could never fire: the id was a fresh uuid on every call, so a
// redelivered event inserted a second row for the same fact instead of
// colliding with the first. This consumer keeps no dedupe table and does not
// commit the offset over a failed handler, so redelivery is the normal path
// after any failure — duplicate signals accumulated.
//
// Deriving the id from the event is what makes the conflict clause mean
// something.

func TestSameEventProducesTheSameSignalID(t *testing.T) {
	const eventID = "5f2b9c1e-0d3a-4c6b-9f11-8e7a2b3c4d5e"
	first := signalID(eventID)
	if second := signalID(eventID); first != second {
		t.Errorf("the same event produced %s then %s; the redelivery would insert a second signal", first, second)
	}
}

func TestDifferentEventsProduceDifferentSignalIDs(t *testing.T) {
	a := signalID("event-a")
	b := signalID("event-b")
	if a == b {
		t.Errorf("distinct events collapsed onto %s; one of the two signals would be dropped", a)
	}
}

// The column is a TEXT primary key and the rest of the table is written with
// this shape, so the derived id has to keep it.
func TestDerivedSignalIDKeepsTheEstablishedShape(t *testing.T) {
	id := signalID("some-event-id")
	if !strings.HasPrefix(id, "SIG-") {
		t.Errorf("id %q lost the SIG- prefix", id)
	}
	if len(id) != len("SIG-")+8 {
		t.Errorf("id %q is %d characters; the established shape is SIG- plus 8", id, len(id))
	}
	if id != strings.ToUpper(id) {
		t.Errorf("id %q is not upper case, unlike every id written before it", id)
	}
}

// An event with no id cannot be recognised on redelivery by any means, so the
// old random behaviour is kept rather than collapsing every anonymous event
// onto one row — that would silently discard genuinely distinct signals.
func TestAnEventWithNoIDStillGetsAUniqueSignal(t *testing.T) {
	if signalID("") == signalID("") {
		t.Error("anonymous events share a signal id; distinct signals would be discarded")
	}
	if signalID(" ") == signalID(" ") {
		t.Error("blank event ids share a signal id; distinct signals would be discarded")
	}
}
