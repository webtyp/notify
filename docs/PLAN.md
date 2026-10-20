---
PLAN: "feat: alert kind catalog, area catalog, and the alert's origin (module + record)"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change. This plan is complete: do not guess; if something does
> not fit, stop and ask in the PR. Design history: `webtyp/docs/MODULE_GUIDANCE_MASTER_PLAN.md` (D7/D8, close
> P7/P8; D12, closes P12; D18, closes P15).

## Goal

Three additions, all additive (no existing field or method removed):

1. **A catalog of alert kinds**, declared by each module that raises alerts: a human name and help text for
   `Alert.Code`, plus which area it belongs to — so a screen can list "Stock bajo (Insumos): se avisa cuando
   el total de un ítem cae bajo su mínimo" instead of a bare code (`ui-consistency`: "no free-text codes").
2. **A catalog of areas**, declared by the application (not hardcoded here) — "Farmacia", "Insumos",
   "Consultorio" — chosen from a list, never typed (same rule).
3. **The alert's origin**: which module and which record produced it, so a consumer (`alert_manager`, D12)
   can offer "Ver" linking back to that record. This mirrors `webtyp.com/money.Ref{Type, ID}` exactly — do
   not invent a different shape.

**Explicitly out of scope — belongs to `alert_manager` (H6), not here:** per-recipient "seen" state (D16).
`notify` has no persistence (same as `money`); do not add storage of any kind.

## Git rules (strict)

- Work on the branch codejob created. Never `git reset`/`checkout -- <file>`/`clean`/`stash`. Commit often.
- **Never pause to ask permission or to report progress.** Delivery check passes → commit, push, open the PR.
  Still failing after 30 minutes → open the PR anyway, list exactly what fails.

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
```

## Stage 1 — `kind.go` (new file)

```go
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
```

Add to `Error` (`notify.go`): `ErrKindCode Error = "notify: a kind needs a code"`, `ErrKindText Error = "notify: a kind needs a name"`, `ErrDuplicateKind Error = "notify: a kind with that code is already registered"`, `ErrAreaCode Error = "notify: an area needs a code"`, `ErrDuplicateArea Error = "notify: an area with that code is already registered"`, `ErrAreaMissing Error = "notify: the area is not registered"`, `ErrKindMissing Error = "notify: the code has no registered kind"`.

## Stage 2 — `registry.go`

Add, following `Register`/`RegisterDriver`'s exact shape (same locking, same duplicate check by code):

- `RegisterArea(a Area) error` — empty `Code` → `ErrAreaCode`; duplicate → `ErrDuplicateArea`.
- `RegisterKind(k Kind) error` — empty `Code` → `ErrKindCode`; empty `Name` → `ErrKindText`; duplicate `Code`
  → `ErrDuplicateKind`; `k.Area` not empty and not a registered `Area.Code` → `ErrAreaMissing` (empty `Area`
  is allowed only while **zero** Areas are registered at all — a module bootstrapping before the app
  declares its areas must not be blocked; once at least one Area exists, every new Kind must name one).
- `Kinds() []Kind` / `Areas() []Area` — read-only snapshots (copy the slice under the read lock, same as
  `Has`), for a screen to list.
- `KindOf(code string) (Kind, bool)` — looks up one, for rendering a single alert's catalog entry.

## Stage 3 — `notify.go`, `Alert`

Add to `Alert` (after `Level`):

```go
// Source names what produced this alert, mirroring webtyp.com/money.Ref: Type is the owning module's own
// name for the thing ("supply_item", "agreement"), ID its id there. Both empty means no origin (a manual
// alert raised by a person, not tied to a record).
Source Ref
```

```go
// Ref identifies a thing in the module that owns it. Same shape as webtyp.com/money.Ref — kept separate
// (notify must not import money) rather than shared, since the two libraries have no other relationship.
type Ref struct {
	Type string
	ID   string
}
```

Update `Validate`: `Source` has no validation of its own (both fields empty is valid — a manual alert).
Update `EncodeFields`/`DecodeFields` (if `Alert` has a `Decode` counterpart — check; if not, this is
encode-only, same as today) to add `w.String("source_type", a.Source.Type)` /
`w.String("source_id", a.Source.ID)`.

## Stage 4 — tests

Through the exported API, `tests/` package `notify_test`:

- `RegisterArea`/`RegisterKind`: empty fields rejected with the right error; duplicates rejected; a Kind
  naming an unregistered Area rejected once at least one Area exists; a Kind with no Area accepted while
  zero Areas exist.
- `Kinds()`/`Areas()`/`KindOf` return what was registered, and a mutation of the returned slice never
  affects the registry's own state (copy semantics).
- `Alert.Validate()` accepts both an empty `Source` and a populated one; `EncodeFields` writes both new
  fields.

## Stage 5 — docs

`README.md`: document `Kind`/`Area`/`Source` next to the existing `Alert`/`Channel` docs, with one short
example of a module registering a Kind at startup.

## Delivery check

- `gotest` green (vet, race, tests, wasm).
- `git diff --stat main --diff-filter=D` empty — nothing deleted; every change additive.
- No behaviour change for an `Alert` that does not set `Source`, or for a `Registry` that never registers an
  `Area`/`Kind`.

## Stages

| # | File(s) | What |
|---|---|---|
| 1 | `kind.go` (new), `notify.go` | `Kind`, `Area`, new `Error` constants |
| 2 | `registry.go` | `RegisterArea`/`RegisterKind`/`Kinds`/`Areas`/`KindOf` |
| 3 | `notify.go` | `Alert.Source Ref`, `Ref` type, `Validate`/`EncodeFields` updates |
| 4 | `tests/*` | registration rules, copy semantics, Alert encode/validate |
| 5 | `README.md` | document the three additions |
