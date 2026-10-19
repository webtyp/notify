---
PLAN: "feat: notify v0.1.0 — Alert, Recipient, the Channel port, the Registry, and the rule that an automation without a configured channel is an error"
TAG: v0.1.0
EXECUTOR: jules
REVIEWER: none
STATUS: review
SESSION: 15789690096219730598
PR: https://github.com/webtyp/notify/pull/1
REVIEWED_COMMIT: f301909a57dfc0132e57c74d28f6bede70d689f2
---

> Read AGENTS.md at the repo root before any change.
> This is a **new library**. `gonew` created only `go.mod`, `LICENSE`, `README.md` and a stub `notify.go`.

## Tools

```
go install webtyp.com/devflow/cmd/gotest@latest
```

## What this library is (and is not)

A business must tell the right person **in time** that something needs attention (a product is below its minimum, a lot
is about to expire, an amount is due) and must be able to choose **how** (email, WhatsApp, a notice inside the app). The
ways of delivery change from business to business, so they are **injected**; and a rule must never silently stop
working because nobody configured a way to deliver it.

This library is only the **contract**: what an alert is, who receives it, what a channel is, a registry of the channels
the application configured, and the validation that an enabled automation has the channels it needs. It sends nothing
over any network: a concrete channel lives in its own repository (`veltylabs/email`, `veltylabs/whatsapp`) and implements
`Channel`. Which alert goes to whom, and when, is another module's job (`veltylabs/modules/alert_manager`).

## Design gate

1. **Prior art.** (a) *Django's `django.core.mail` backends and Laravel's Notification channels*: one notification
   object, many interchangeable delivery channels chosen by configuration — adopted. (b) *Go's `log/slog` `Handler`*:
   a one-method interface injected at the edge — adopted for `Channel`. (c) *Twelve-factor "backing services"*: the
   delivery service is an attached resource swapped by configuration — adopted, and the reason the registry is
   filled by the application. It differs in one rule the others lack: **a missing channel is a start-up error**, not a
   silently dropped message (Django's console/dummy backends fail silently by design).
2. **Novice-name test.** `notify.Alert`, `notify.Recipient`, `notify.Channel`, `notify.Registry`, `notify.Automation`,
   `Registry.Register`, `Registry.Validate`, `Registry.Deliver`, `notify.Result`. Read aloud: "register a channel",
   "validate the automations", "deliver this alert to this person by email and WhatsApp".
3. **Complexity ledger.**
   ```
   Concepts the developer must learn   +4 / −0   (Alert, Channel, Registry, Automation)
   Files they must touch to do X       +0 / −0
   Lines at the call site              −30 / −0
   Ways to do the same thing           +0 / −1   (one way to alert, instead of each module sending its own mail)
   ```
4. **Where it belongs.** Its own repo: one concern (the contract), no transport (open/closed: a new channel touches
   zero lines of the library or of its consumers).
5. **What it deletes.** Nothing here (new); it prevents each module from calling a mail or messaging service itself.

## Public API (implement exactly this; package `notify`)

```go
// Error is a failure of a notification rule. Constants, compared by value.
type Error string

func (e Error) Error() string { return string(e) }

const (
	ErrChannelName      Error = "notify: a channel needs a name"
	ErrDuplicateChannel Error = "notify: a channel with that name is already registered"
	ErrChannelMissing   Error = "notify: the channel is not configured"
	ErrNoChannels       Error = "notify: an automation needs at least one channel"
	ErrNoAddress        Error = "notify: the recipient has no address for this channel"
	ErrAlertCode        Error = "notify: an alert needs a code"
	ErrAlertText        Error = "notify: an alert needs a subject and a body"
	ErrNoRecipient      Error = "notify: an alert needs a recipient"
)

// Names of the channels this ecosystem ships. The registry accepts any non-empty name.
const (
	ChannelEmail    = "email"
	ChannelWhatsApp = "whatsapp"
	ChannelInApp    = "in_app"
)

// Urgency orders how much an alert matters.
type Urgency int8

const (
	Info     Urgency = 0
	Warning  Urgency = 1
	Critical Urgency = 2
)

// Recipient is who receives an alert. A channel uses the address it needs; ID is the recipient's id in its own module.
type Recipient struct {
	ID    string
	Name  string
	Email string
	Phone string // international format, digits with a leading +
}

// Alert is what happened that someone must know. Code identifies the rule that raised it ("stock.below_minimum");
// Scope is the area of the business it belongs to, so a report can show only what a person is responsible for.
type Alert struct {
	Code    string
	Scope   string
	Urgency Urgency
	Subject string
	Body    string
}

// Validate returns ErrAlertCode if Code is empty and ErrAlertText if Subject or Body is empty.
func (a Alert) Validate() error

// Channel delivers an alert to one recipient. Send returns nil only if the delivery was accepted by the service behind
// the channel; any failure is returned, never swallowed. A Channel reads the address it needs from the Recipient and
// returns ErrNoAddress if it is empty.
type Channel interface {
	Name() string
	Send(a Alert, to Recipient) error
}

// Automation is a rule the application turned on (an alert code) together with the channel names it must be able to use.
type Automation struct {
	Code     string
	Channels []string
}

// Registry holds the channels the application configured.
type Registry struct { /* unexported: a slice of channels, no map */ }

// Register adds a channel. An empty name → ErrChannelName; a name already registered → ErrDuplicateChannel.
func (r *Registry) Register(c Channel) error

// Has reports whether a channel with that name is registered.
func (r *Registry) Has(name string) bool

// Validate checks every automation against the registry and is meant to run at start-up. An automation with no
// channels → ErrNoChannels; a channel name that is not registered → ErrChannelMissing. It checks ALL of them and
// returns one Error whose text lists every automation code and missing channel (text built with webtyp/fmt), or nil.
func (r *Registry) Validate(autos []Automation) error

// Result is the outcome of delivering one alert through one channel.
type Result struct {
	Channel string
	Err     error // nil = accepted by the channel
}

// Deliver sends the alert to the recipient through each named channel, one Result per name, in the order given.
// It first validates the alert (an invalid alert → every Result carries that error and nothing is sent). A name that is
// not registered gives ErrChannelMissing in its Result; one channel failing never stops the others. Deliver never returns
// fewer Results than names. No names → a single Result with ErrNoChannels.
func (r *Registry) Deliver(a Alert, to Recipient, channels []string) []Result
```

Implementation constraints: no `map`, no `reflect`, no stdlib `errors`/`strings`/`strconv`/`fmt`, no network, no clock.

## Test helper — package `notifytest` (`//go:build !wasm`)

```go
// Recorder is a Channel for tests: it records what it was asked to send and can be made to fail.
type Recorder struct {
	ChannelName string
	Fail        error // returned by every Send when not nil
	Sent        []Sent
}

type Sent struct {
	Alert notify.Alert
	To    notify.Recipient
}

func (r *Recorder) Name() string
func (r *Recorder) Send(a notify.Alert, to notify.Recipient) error // appends to Sent unless Fail is set
```

## Stages

### Stage 1 — `notify.go` (replacing the stub), `registry.go`

`notify.go`: `Error`, the constants, `Urgency`, `Recipient`, `Alert`, `Validate`, `Channel`, `Automation`, `Result`.
`registry.go`: `Registry`, `Register`, `Has`, `Validate`, `Deliver`. `go.mod`: `go 1.26.8`; the only dependency is
`webtyp.com/fmt`.

### Stage 2 — tests (`tests/`, external package `notify_test`, standard library only, through the exported API)

- `Alert.Validate`: each rule.
- `Register`: ok, empty name, duplicate; `Has`.
- `Validate`: no automations → nil; all channels present → nil; one missing → `ErrChannelMissing` text names the code and the
  channel; several problems → one error listing all; an automation with no channels → `ErrNoChannels`.
- `Deliver`: one result per name in order; a failing channel does not stop the next; an unregistered name →
  `ErrChannelMissing` in its result; an invalid alert → every result carries the error and the recorder saw nothing; no
  names → one `ErrNoChannels` result; a channel returning `ErrNoAddress` is reported, not swallowed.
Use `notifytest.Recorder` (Stage 3) as the channel.

### Stage 3 — `notifytest/recorder.go`

As specified above.

### Stage 4 — docs

`README.md` (what it is, the registry at start-up and a delivery in 15 lines), `docs/ARCHITECTURE.md` (why a missing
channel is an error at start-up, how a concrete channel implements `Channel`, why there is no retry or queue here —
that belongs to `alert_manager`). Permanent docs never link `docs/PLAN.md`.

## Delivery check

```
gotest                                                       # vet, race, tests and wasm green
GOOS=js GOARCH=wasm go vet ./...                             # the root package builds for wasm (notifytest is !wasm; tests that import it carry //go:build !wasm)
grep -rn "map\[\|reflect\|\"errors\"\|\"strings\"\|\"strconv\"\|net/http" --include=*.go . | grep -v _test   # → empty
git diff --stat main                                         # only the files of the stages
```

## Stages table

| # | Stage | Files |
|---|---|---|
| 1 | the contract and the registry | `notify.go`, `registry.go`, `go.mod` |
| 2 | tests | `tests/alert_test.go`, `tests/registry_test.go` |
| 3 | test helper | `notifytest/recorder.go` |
| 4 | docs | `README.md`, `docs/ARCHITECTURE.md` |
