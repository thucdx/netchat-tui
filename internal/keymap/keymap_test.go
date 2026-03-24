package keymap

import (
	"reflect"
	"testing"

	"github.com/charmbracelet/bubbles/key"
)

// TestDefaultKeyMapHasAllBindings verifies every field in KeyMap is non-zero (all bindings populated).
func TestDefaultKeyMapHasAllBindings(t *testing.T) {
	km := DefaultKeyMap()
	v := reflect.ValueOf(km)
	typ := reflect.TypeOf(km)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := typ.Field(i).Name

		binding, ok := field.Interface().(key.Binding)
		if !ok {
			t.Errorf("field %s is not a key.Binding", fieldName)
			continue
		}

		if len(binding.Keys()) == 0 {
			t.Errorf("field %s has no keys set", fieldName)
		}
	}
}

// TestShortHelp verifies ShortHelp() returns exactly 7 bindings.
func TestShortHelp(t *testing.T) {
	km := DefaultKeyMap()
	short := km.ShortHelp()
	if len(short) != 7 {
		t.Errorf("ShortHelp() returned %d bindings, want 7", len(short))
	}
}

// TestFullHelp verifies FullHelp() returns 6 groups.
func TestFullHelp(t *testing.T) {
	km := DefaultKeyMap()
	full := km.FullHelp()
	if len(full) != 6 {
		t.Errorf("FullHelp() returned %d groups, want 6", len(full))
	}
}
