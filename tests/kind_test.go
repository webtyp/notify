//go:build !wasm

package notify_test

import (
	"testing"

	"webtyp.com/lang"
	"webtyp.com/notify"
)

func TestRegistry_RegisterArea(t *testing.T) {
	r := &notify.Registry{}

	err := r.RegisterArea(notify.Area{Code: "", Name: "Empty"})
	if err != notify.ErrAreaCode {
		t.Errorf("expected ErrAreaCode, got %v", err)
	}

	err = r.RegisterArea(notify.Area{Code: "pharmacy", Name: "Pharmacy"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	err = r.RegisterArea(notify.Area{Code: "pharmacy", Name: "Pharmacy Duplicate"})
	if err != notify.ErrDuplicateArea {
		t.Errorf("expected ErrDuplicateArea, got %v", err)
	}
}

func TestRegistry_RegisterKind(t *testing.T) {
	r := &notify.Registry{}

	err := r.RegisterKind(notify.Kind{Code: "", Name: "Empty"})
	if err != notify.ErrKindCode {
		t.Errorf("expected ErrKindCode, got %v", err)
	}

	err = r.RegisterKind(notify.Kind{Code: "code", Name: ""})
	if err != notify.ErrKindText {
		t.Errorf("expected ErrKindText, got %v", err)
	}

	// OK with empty area if no areas are registered
	err = r.RegisterKind(notify.Kind{Code: "test.ok", Name: "Test"})
	if err != nil {
		t.Errorf("expected no error with zero areas registered, got %v", err)
	}

	err = r.RegisterKind(notify.Kind{Code: "test.ok", Name: "Test Duplicate"})
	if err != notify.ErrDuplicateKind {
		t.Errorf("expected ErrDuplicateKind, got %v", err)
	}

	// Cannot register kind with unregistered area
	err = r.RegisterKind(notify.Kind{Code: "test.noarea", Name: "Test", Area: "pharmacy"})
	if err != notify.ErrAreaMissing {
		t.Errorf("expected ErrAreaMissing with unregistered area, got %v", err)
	}

	// Register an area
	err = r.RegisterArea(notify.Area{Code: "pharmacy", Name: "Pharmacy"})
	if err != nil {
		t.Fatalf("failed to register area: %v", err)
	}

	// Once an area is registered, kind without area is rejected
	err = r.RegisterKind(notify.Kind{Code: "test.missingarea", Name: "Test"})
	if err != notify.ErrAreaMissing {
		t.Errorf("expected ErrAreaMissing when areas exist but kind has no area, got %v", err)
	}

	// Kind with registered area is accepted
	err = r.RegisterKind(notify.Kind{Code: "test.pharmacy", Name: "Pharmacy Alert", Area: "pharmacy"})
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestRegistry_KindsAndAreas(t *testing.T) {
	r := &notify.Registry{}

	area := notify.Area{Code: "pharmacy", Name: lang.Text("Pharmacy")}
	kind := notify.Kind{Code: "test.pharmacy", Name: lang.Text("Pharmacy Alert"), Area: "pharmacy"}

	r.RegisterArea(area)
	r.RegisterKind(kind)

	areas := r.Areas()
	if len(areas) != 1 || areas[0].Code != "pharmacy" {
		t.Errorf("unexpected areas: %v", areas)
	}

	kinds := r.Kinds()
	if len(kinds) != 1 || kinds[0].Code != "test.pharmacy" {
		t.Errorf("unexpected kinds: %v", kinds)
	}

	k, ok := r.KindOf("test.pharmacy")
	if !ok || k.Code != "test.pharmacy" {
		t.Errorf("expected to find kind test.pharmacy")
	}

	_, ok = r.KindOf("missing")
	if ok {
		t.Errorf("expected not to find missing kind")
	}

	// Verify mutations do not affect internal state
	areas[0].Code = "changed"
	areas2 := r.Areas()
	if areas2[0].Code == "changed" {
		t.Errorf("mutation of slice element affected registry")
	}

	areas = append(areas, notify.Area{Code: "new"})
	areas3 := r.Areas()
	if len(areas3) != 1 {
		t.Errorf("append to slice affected registry")
	}

	kinds[0].Code = "changed"
	kinds2 := r.Kinds()
	if kinds2[0].Code == "changed" {
		t.Errorf("mutation of slice element affected registry")
	}

	kinds = append(kinds, notify.Kind{Code: "new"})
	kinds3 := r.Kinds()
	if len(kinds3) != 1 {
		t.Errorf("append to slice affected registry")
	}
}
