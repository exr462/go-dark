package action

import (
	"encoding/json"
	"testing"
)

// TestAction_JSONRoundTripsByStableName guards the fix for a real bug:
// Action used to (de)serialize as its raw iota int, so inserting a new
// Action anywhere but the very end of the const block silently shifted the
// meaning of every later value already persisted in a user's config.json
// (e.g. a saved "Enter" shortcut re-read itself as "Save" once the enum
// grew). Marshaling now goes through a stable name instead.
func TestAction_JSONRoundTripsByStableName(t *testing.T) {
	for action, name := range actionNames {
		data, err := json.Marshal(action)
		if err != nil {
			t.Fatalf("Marshal(%v) failed: %v", action, err)
		}
		if string(data) != `"`+name+`"` {
			t.Fatalf("expected Action %v to marshal as %q, got %s", action, name, data)
		}

		var decoded Action
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("Unmarshal(%s) failed: %v", data, err)
		}
		if decoded != action {
			t.Fatalf("expected round trip to restore %v, got %v", action, decoded)
		}
	}
}

// TestAction_UnmarshalJSON_LegacyNumericFormBecomesInvalid is the crux of the
// fix: a bare JSON number (the old serialization format) can no longer be
// trusted to mean what it used to, since the const block may have been
// reordered since it was written. It must decode to invalidAction - never a
// real action - so callers (config.LoadConfig) can safely drop and
// backfill it instead of silently mis-binding a shortcut.
func TestAction_UnmarshalJSON_LegacyNumericFormBecomesInvalid(t *testing.T) {
	var decoded Action
	if err := json.Unmarshal([]byte(`11`), &decoded); err != nil {
		t.Fatalf("expected legacy numeric form to decode without error, got %v", err)
	}
	if decoded.IsValid() {
		t.Fatalf("expected a legacy numeric Action to decode as invalid, got a valid action %v", decoded)
	}
}

// TestAction_UnmarshalJSON_UnknownNameBecomesInvalid guards forward
// compatibility: a config.json written by a *future* version of go-dark with
// an action name this binary doesn't know about should degrade gracefully
// instead of erroring out the whole config load.
func TestAction_UnmarshalJSON_UnknownNameBecomesInvalid(t *testing.T) {
	var decoded Action
	if err := json.Unmarshal([]byte(`"SomeFutureAction"`), &decoded); err != nil {
		t.Fatalf("expected unknown action name to decode without error, got %v", err)
	}
	if decoded.IsValid() {
		t.Fatalf("expected an unknown action name to decode as invalid, got a valid action %v", decoded)
	}
}

// TestAction_IsValid_AllDefaultShortcutsAreValid is a sanity check that every
// action.DefaultShortcuts entry is backed by a registered stable name.
func TestAction_IsValid_AllDefaultShortcutsAreValid(t *testing.T) {
	for _, s := range DefaultShortcuts {
		if !s.Action.IsValid() {
			t.Errorf("DefaultShortcuts contains an Action with no registered stable name: %v", s.Action)
		}
	}
}
