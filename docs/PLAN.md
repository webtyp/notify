---
PLAN: "feat!: a Driver declares its label and the fields it needs, so one screen configures every sending channel"
EXECUTOR: jules
REVIEWER: none
---

> Read AGENTS.md at the repo root before any change.

# notify: drivers declare the data they need

## Why

Owner decision: *each sending driver declares the data it needs; «Sending channels» is one single
screen*. Today a `Driver` only has `Kind()` and `New(c Config, secrets SecretResolver)`: the keys it
reads from `Config.Settings`/`Config.Secrets` are constants hidden inside the driver
(`veltylabs/email`: `account_id`, `from`, `from_name`, `endpoint`, secret `api_token`). A screen
cannot know what to ask. This plan makes the driver say it, typed.

Breaking change (no compatibility shim): `Driver` gains two methods. The only implementation in the
ecosystem is `github.com/veltylabs/email` (updated by its own plan right after this tag).

## Tools

```bash
go install webtyp.com/devflow/cmd/gotest@latest
go install webtyp.com/lang/cmd/langc@latest
```

## Environment

```bash
export GOPRIVATE=webtyp.com/*,github.com/veltylabs/*
export GOFLAGS=-mod=mod
```

A PR that does not compile, or whose tests could not run, is a failed delivery. Never pause to ask
permission or report progress: when the Delivery check passes, commit, push and open the PR. Stuck
30 minutes: open the PR anyway listing what fails.

## Design gate

1. **Prior art:** Home Assistant config flows (`data_schema` declared by each integration), n8n
   node `properties` (each credential type declares its fields, `typeOptions: password`), Grafana
   data-source plugins (`jsonData` vs `secureJsonData`). All split public settings from secrets and
   let one generic form render them.
2. **Novice-name test:** `Driver.Fields() []Field`, `Field{Key, Label, Help, Secret, Required}`
   read as what they are.
3. **Complexity ledger:** +1 type (`Field`), +2 methods on `Driver`, +2 registry readers; −N
   hand-written forms (one per driver) that would otherwise exist.
4. **Where it belongs:** `notify` owns the `Driver` contract; the field list is part of it.
5. **What it deletes:** nothing yet; it prevents per-driver screens.

## Stage 1 — the contract (`driver.go`)

```go
// Field is one value a driver needs to build a channel. A screen renders one input per Field and
// stores it in Config.Settings (Secret == false) or Config.Secrets (Secret == true).
type Field struct {
	Key      string    // the Setting/Secret key the driver reads ("from", "api_token")
	Label    lang.Text // what a person reads above the input
	Help     lang.Text // optional: one line explaining where the value comes from
	Secret   bool      // true: the stored value is the NAME of a secret in the destination's store, never the value
	Required bool      // true: Registry.Load refuses a config that lacks it
}

// Driver builds channels of one kind. The returned Channel's Name() must equal c.Name.
type Driver interface {
	Kind() string
	Label() lang.Text // the kind as a person reads it ("Email")
	Fields() []Field  // every key New reads, in the order a screen shows them
	New(c Config, secrets SecretResolver) (Channel, error)
}
```

`webtyp.com/lang` becomes a direct dependency (`go get webtyp.com/lang@v0.1.7`; it is already
indirect).

## Stage 2 — the registry (`registry.go`)

1. `RegisterDriver(d)`: besides today's checks, validate `d.Fields()`:
   - a field with `Key == ""` → new constant `ErrFieldKey Error = "notify: a driver field needs a key"`;
   - two fields with the same `Key` → `ErrDuplicateField Error = "notify: a driver declares the same field twice"`;
   - a field with `Label == ""` → `ErrFieldLabel Error = "notify: a driver field needs a label"`.
2. New readers:

```go
// DriverKinds returns the registered driver kinds, in registration order.
func (r *Registry) DriverKinds() []string

// DriverOf returns the label and fields of a registered driver kind; ok is false if none.
func (r *Registry) DriverOf(kind string) (label lang.Text, fields []Field, ok bool)
```

3. `Load`: before calling `driver.New(config, secrets)`, for every `Required` field check the key is
   present with a non-empty value in `config.Settings` (when `!Secret`) or `config.Secrets` (when
   `Secret`). Missing → the config fails exactly like a `New` error does today (same error
   collection), with the text `"<config.Name>: missing <key>"` wrapped with `ErrConfig` the same way
   the existing `Load` error path formats its messages (read it and reuse it).

| Call | Result |
|---|---|
| `RegisterDriver` with fields `[{Key:"from",Label:"Sender"}]` | ok |
| field `Key: ""` | `ErrFieldKey` |
| two fields `Key: "from"` | `ErrDuplicateField` |
| field `Label: ""` | `ErrFieldLabel` |
| `DriverKinds()` after registering `email`, `whatsapp` | `["email", "whatsapp"]` |
| `DriverOf("email")` | its label and fields, `true` |
| `DriverOf("sms")` | `"", nil, false` |
| `Load` with a config lacking a `Required` setting | that config is not built; `Load` returns the error naming the key; `New` is **not** called for it |
| `Load` with all required present | as today |

## Stage 3 — tests (`tests/registry_test.go`, package `notify_test`)

Update `testDriver` with `Label() lang.Text { return "Test" }` and
`Fields() []notify.Field { return d.fields }` (new field `fields []notify.Field`; `failNew` and
`nameFail` stay). Add `TestRegistry_DriverFields` with one subtest per row of the Stage 2 table.
The `Load` rows use a `testDriver` that records whether `New` ran.

## Stage 4 — docs

- `README.md`: in «Dynamic Configurations and Drivers», the `Field` type, the two new methods and a
  12-line example of a driver declaring `from` (setting, required) and `api_token` (secret,
  required).
- `docs/` file with the same section (`grep -rln "Dynamic Configurations" docs/`): same text.

## Delivery check

```bash
go build ./...
gotest
git diff --stat main   # driver.go, registry.go, tests/registry_test.go, go.mod, go.sum, README.md, docs/*
```

## Stages

| # | Files | Done when |
|---|---|---|
| 1 | `driver.go`, `go.mod`, `go.sum` | builds |
| 2 | `registry.go` | three errors, two readers, required check in `Load` |
| 3 | `tests/registry_test.go` | table green, existing tests green |
| 4 | `README.md`, `docs/*` | docs match |
