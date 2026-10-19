---
PLAN: "feat!: notify v0.2.0 — alert severity is msgtype.Type, and an Alert travels as an events payload (Raise / Subscribe)"
TAG: v0.2.0
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
```

## Why (what v0.1.0 duplicated)

An audit of the ecosystem against `webtyp.com/notify` v0.1.0 found two duplications; both are fixed here, before any
consumer exists (the only dependent, `veltylabs/email`, is still being written and pins v0.1.0):

1. **Severity.** `notify.Urgency` (`Info`, `Warning`, `Critical`) re-invents `webtyp.com/msgtype.Type` (`Info`, `Warning`,
   `Error`, `Success`…), the ecosystem's message severity, which `platformd.Notify(msgtype.Error, …)` already uses.
2. **Delivery of the alert itself.** A module that wants to raise an alert had to call a notify-specific path. The ecosystem
   already has `webtyp.com/events` (`Publisher`, `Subscriber`, `Event`, `Broker`, `Fanout`) and every module already receives an
   `events.Publisher`. An alert is an event: modules `Publish` it, `alert_manager` subscribes.

What is **not** a duplicate and stays: `Channel`, `Registry`, `Config`/`Driver`/`Load`, `Validate` and `Deliver`. An
`events.Handler` returns nothing and promises no delivery; a channel must return its error and a missing channel must be a
loud error — that contract does not exist in `events`.

## Ecosystem search (done before writing this plan)

Searched `webtyp/*` and `veltylabs/*`: `msgtype` (severity — **reuse**), `events` (pub/sub — **reuse**), `bus` (a second minimal
pub/sub — not used here, not touched), `outbox` (a browser offline queue of mutations — unrelated), `idempotency` (HTTP
middleware — unrelated). Nothing else in the ecosystem defines a delivery channel with per-channel errors: `Channel` stays.

## Design gate

1. **Prior art.** As in v0.1.0 (Django mail backends, Laravel notification channels, `log/slog.Handler`), plus the
   ecosystem's own `events` contract for how things are announced.
2. **Novice-name test.** `Alert.Level`, `notify.Raise`, `notify.Subscribe`, `notify.TopicAlert`. Read aloud: "raise this alert",
   "subscribe to alerts".
3. **Complexity ledger.**
   ```
   Concepts the developer must learn   +0 / −1   (Urgency disappears; severity is the one they already know)
   Files they must touch to do X       +0 / −0
   Lines at the call site              −5 / −0
   Ways to do the same thing           +0 / −2   (severity: one; announcing an alert: one)
   ```
4. **Where it belongs.** Here: the glue between an `Alert` and `events` is written once, in the library that owns `Alert`.
5. **What it deletes.** `Urgency` and its constants `Info`, `Warning`, `Critical` (breaking; no consumers).

## Stage 1 — the changes

**`notify.go`**

- Delete `type Urgency` and the constants `Info`, `Warning`, `Critical`.
- `Alert` field `Urgency Urgency` becomes `Level msgtype.Type` (import `webtyp.com/msgtype`). Allowed levels: `msgtype.Info`,
  `msgtype.Warning`, `msgtype.Error`. `Alert.Validate` returns the new constant `ErrAlertLevel = "notify: an alert level must be Info, Warning or Error"`
  for any other value (including the zero value `msgtype.Normal`), after the existing code and text checks.
- `*Alert` implements `model.Encodable` (`webtyp.com/model`): `IsNil() bool` (true for a nil receiver) and
  `EncodeFields(w model.FieldWriter)` writing, in this order: `code`, `scope`, `subject`, `body` (Strings) and `level` (Int).
- `const TopicAlert = "notify.alert"`.

**`events.go`** (new file)

```go
// Raise announces an alert on TopicAlert through the module's events.Publisher.
// An invalid alert (Alert.Validate) is NOT published: its error is returned.
func Raise(p events.Publisher, a Alert) error

// Subscribe registers h to receive every alert raised on TopicAlert. A nil h or nil onError → the new constant
// ErrHandler = "notify: Subscribe needs a handler and an error handler" (an error is never dropped silently).
// A payload on TopicAlert that is not an *Alert goes to onError(ErrPayload) with
// ErrPayload = "notify: the payload of a notify.alert event is not an alert".
func Subscribe(s events.Subscriber, h func(Alert), onError func(error)) error
```

`go.mod`: add `webtyp.com/events`, `webtyp.com/model` and `webtyp.com/msgtype` (the exact versions the other `webtyp` repos pin today:
`grep -h "webtyp.com/\(events\|model\|msgtype\) " ../*/go.mod | sort | uniq -c`; use the highest published). No local `replace`.

**`notifytest/recorder.go`**: update to `Level`.

**`AGENTS.md`**: the first bullet of "This repo" says the library imports no network package; add that it may import `events`,
`model` and `msgtype` (the contracts it builds on), nothing else.

## Stage 2 — tests (`tests/`, through the exported API, `//go:build !wasm`, with `webtyp.com/events/mock`'s `Broker`)

- Update every existing use of `Urgency` to `Level`.
- `Alert.Validate`: each level in {Info, Warning, Error} passes; `msgtype.Normal`, `Success` and `Debug` give `ErrAlertLevel`.
- `Raise` + `Subscribe`: a valid alert reaches the handler with identical content; an invalid alert is not published and
  `Raise` returns its error; two subscribers both receive it; a foreign payload reaches `onError(ErrPayload)`; `Subscribe` with
  a nil handler or nil `onError` returns `ErrHandler`.
- `(*Alert).EncodeFields` writes every field (a small recording `model.FieldWriter` in the test); `(*Alert)(nil).IsNil()` is true.

## Stage 3 — docs

`README.md` and `docs/ARCHITECTURE.md`: severity is `msgtype.Type`; how a module raises an alert with the `events.Publisher`
it already has, and how `alert_manager` subscribes; why `Channel` returns errors and is not an `events.Handler`.
Permanent docs never link `docs/PLAN.md`.

## Delivery check

```
gotest                                                       # vet, race, tests and wasm green
GOOS=js GOARCH=wasm go vet ./...
grep -rn "Urgency\|Critical" --include=*.go --include=*.md . | grep -v docs/PLAN.md   # → empty
grep -rn "map\[\|reflect\|\"errors\"\|\"strings\"\|\"strconv\"\|net/http" --include=*.go . | grep -v _test   # → empty
git diff --stat main                                         # only the files of the stages
```

## Stages table

| # | Stage | Files |
|---|---|---|
| 1 | severity and events glue | `notify.go`, `events.go`, `notifytest/recorder.go`, `go.mod`, `AGENTS.md` |
| 2 | tests | `tests/alert_test.go`, `tests/registry_test.go`, `tests/events_test.go` |
| 3 | docs | `README.md`, `docs/ARCHITECTURE.md` |
