package notify

import "webtyp.com/lang"

// Kind is one catalog entry for an alert code: what it means and where it belongs. A module that raises
// alerts registers one Kind per Code it uses, so a screen can list and explain every possible alert without
// a free-text lookup.
type Kind struct {
	Code string
	Name lang.Text
	Help lang.Text
	Area string // one of the application's registered Areas; "" is invalid once at least one Area is registered
}

// Area is one business area an alert can belong to ("pharmacy", "supplies"…), declared by the application —
// never hardcoded here, never typed free-text by a person (ui-consistency: "no free-text codes").
type Area struct {
	Code string
	Name lang.Text
}
