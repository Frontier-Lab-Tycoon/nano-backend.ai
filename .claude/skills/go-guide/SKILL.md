---
name: go-guide
description: Go coding conventions for nano-backend.ai — error handling, type design, concurrency patterns, quality enforcement
user-invocable: true
---

# Go Coding Guide

Condensed conventions for all Go code in this project.

**References:**
- [Effective Go](https://go.dev/doc/effective_go), [Go Code Review Comments](https://go.dev/wiki/CodeReviewComments)
- [GO_CODING_GUIDELINES.md](./GO_CODING_GUIDELINES.md) — production codebase conventions (interface design, DI, composition, package structure). Read this file for detailed patterns beyond the condensed guide below.

## Error Handling

- Return `error` as the **last return value** for all fallible operations
- Define domain error types as structs implementing the `error` interface
- Use `fmt.Errorf` with `%w` for wrapping errors (enables `errors.Is` / `errors.As`)
- **No silent failures**: never discard errors with `_`
- Use the **"comma ok" idiom** for map lookups and type assertions

```go
// Good — domain error type
type KernelError struct {
    Op   string
    ID   KernelID
    Err  error
}

func (e *KernelError) Error() string {
    return fmt.Sprintf("kernel %s: %s: %v", e.Op, e.ID, e.Err)
}

func (e *KernelError) Unwrap() error { return e.Err }

// Good — wrapping with context
if err != nil {
    return fmt.Errorf("create kernel %s: %w", id, err)
}

// Good — sentinel errors
var (
    ErrNotFound      = errors.New("kernel not found")
    ErrAlreadyExists = errors.New("kernel already exists")
)

// Bad — primitive string errors
func doThing() error { return errors.New("something broke") }

// Bad — discarding errors
result, _ := riskyOperation()
```

## Type Design

- **Avoid primitive obsession** — wrap IDs, keys, tokens in named types
- Use composite literals for constructors; export `NewXxx()` functions
- Zero value should be useful when possible (e.g., `sync.Mutex`, `bytes.Buffer`)
- Omit package name from exported names: `kernel.ID` not `kernel.KernelID`

```go
// Good — named type for IDs
type ID string

func NewID() ID {
    return ID(uuid.New().String())
}

// Good — composite literal constructor
func NewSpec(command []string) Spec {
    return Spec{Command: command}
}

// Good — zero value is useful
type Counter struct {
    mu    sync.Mutex
    count int
}
// Ready to use without initialization
```

## Naming (Effective Go)

### Packages

- Lowercase, single-word names — **no underscores, no mixedCaps**
- Package name provides context: `kernel.Status` not `kernel.KernelStatus`

### Exported Names

- `MixedCaps` for exported, `mixedCaps` for unexported — **never underscores**
- Getters: `Owner()` not `GetOwner()`; setters: `SetOwner()`
- Avoid generic weasel words: ~~Manager~~, ~~Service~~, ~~Factory~~, ~~Handler~~
- Use names that describe *what it does*: `JobScheduler`, `HeartbeatMonitor`, `SessionStore`

### Interfaces

- One-method interfaces: method name + **-er suffix** (`Reader`, `Writer`, `Closer`)
- Keep interfaces small (1-3 methods)
- Honor canonical names: `Read`, `Write`, `Close`, `String` — use them only with matching semantics

```go
// Good — small, focused interface
type KernelRuntime interface {
    Create(ctx context.Context, spec Spec) (ID, error)
    Destroy(ctx context.Context, id ID) error
    Status(ctx context.Context, id ID) (Status, error)
}

// Good — export interface, not implementation
func NewRuntime(config Config) KernelRuntime {
    return &localProcess{config: config}
}
```

## Panic Policy

- **Never panic in library code** — return `error`
- Panics are only for **unrecoverable programming errors** (invariant violations)
- Valid uses: `init()` when setup is impossible, truly unreachable code paths
- Use `recover()` in goroutine entry points to prevent server crashes

```go
// Good — recover in server goroutine
func safelyDo(work *Work) {
    defer func() {
        if err := recover(); err != nil {
            slog.Error("work panicked", "error", err)
        }
    }()
    do(work)
}

// Good — panic in init for unrecoverable setup failure
func init() {
    if os.Getenv("REQUIRED_VAR") == "" {
        panic("REQUIRED_VAR not set")
    }
}
```

## Concurrency Patterns

- **Do not communicate by sharing memory; share memory by communicating**
- Use **goroutines** for concurrent work — lightweight, multiplexed onto OS threads
- Use **channels** for synchronization and communication
- Use `context.Context` as the first parameter for cancellation and timeouts
- Prefer `select` with `context.Done()` for graceful shutdown
- Use buffered channels as semaphores for rate limiting

```go
// Good — channel for synchronization
func processAll(items []Item) error {
    errc := make(chan error, len(items))
    for _, item := range items {
        go func(it Item) {
            errc <- process(it)
        }(item)
    }
    for range items {
        if err := <-errc; err != nil {
            return err
        }
    }
    return nil
}

// Good — semaphore pattern for rate limiting
var sem = make(chan struct{}, MaxConcurrent)

func handle(r *Request) {
    sem <- struct{}{}  // Acquire
    defer func() { <-sem }()  // Release
    process(r)
}

// Good — graceful shutdown with context, checking channel close
func serve(ctx context.Context, queue <-chan *Request) {
    for {
        select {
        case req, ok := <-queue:
            if !ok {
                slog.Info("queue closed")
                return
            }
            go handle(req)
        case <-ctx.Done():
            slog.Info("shutting down")
            return
        }
    }
}
```

## Resource Management with Defer

- Use `defer` for cleanup: file closing, mutex unlocking, connection release
- Deferred calls execute in **LIFO** order
- Arguments are evaluated when `defer` executes, not when the deferred call runs
- Place `defer` immediately after acquiring the resource

```go
// Good — defer immediately after open
func readFile(path string) ([]byte, error) {
    f, err := os.Open(path)
    if err != nil {
        return nil, err
    }
    defer f.Close()

    return io.ReadAll(f)
}

// Good — mutex unlock
func (c *Counter) Increment() {
    c.mu.Lock()
    defer c.mu.Unlock()
    c.count++
}
```

## Logging (Structured)

- Use `log/slog` — no `fmt.Println` or `log.Println` for application logging
- Structured fields: `slog.Info("started", "kernel_id", id, "status", "running")`
- Redact sensitive data (tokens, passwords) — never log secrets

```go
// Good
slog.Info("kernel created",
    "kernel_id", id,
    "command", spec.Command,
)

slog.Error("kernel failed",
    "kernel_id", id,
    "error", err,
)

// Bad
fmt.Printf("kernel %s created\n", id)
log.Printf("error: %v", err)
```

## Lint & Formatting

```bash
gofmt -w .                     # format all code (or goimports)
golangci-lint run ./...        # comprehensive lint (zero warnings policy)
go vet ./...                   # static analysis
govulncheck ./...              # dependency vulnerability check
```

- Format with `gofmt` — tabs for indentation, no manual alignment
- Opening braces on same line as control structure (mandatory in Go)
- Never suppress lint warnings to avoid fixing issues

## Package Design

- Favor multiple focused packages over a monolith
- Each package has a single clear responsibility
- No circular dependencies between packages
- Export only what is needed — unexported by default
- Use `internal/` directory to prevent external imports

```text
cmd/
  manager/main.go          # binary entry point
  agent/main.go            # binary entry point
internal/
  common/                  # shared types, interfaces
  manager/                 # manager-specific logic
  agent/                   # agent-specific logic
```

## Magic Values

- All hardcoded values must have a comment explaining *why* that value
- Include references to external specs or system requirements
- Use typed constants with `iota` for enumerations

```go
// Maximum heartbeat interval before an agent is considered dead.
// Chosen to tolerate 2 missed beats at the 5s send interval.
const heartbeatTimeout = 15 * time.Second

// KernelStatus represents the runtime state of a kernel.
type KernelStatus int

const (
    StatusRunning KernelStatus = iota
    StatusExited
    StatusFailed
)
```

## Control Flow Idioms

- Omit `else` when `if` body ends in `return`, `break`, `continue`
- Use initialization statements in `if` for scoped variables
- Use `switch` without expression for if-else chains
- No automatic fall-through in switch (use `fallthrough` keyword explicitly if needed)

```go
// Good — early return, no else
f, err := os.Open(name)
if err != nil {
    return err
}
defer f.Close()
// continue with f...

// Good — init statement in if
if err := validate(input); err != nil {
    return err
}

// Good — switch as if-else chain
switch {
case x < 0:
    return -x
case x == 0:
    return 0
default:
    return x
}
```

## Slices, Maps, and Make

- Use `make` for slices, maps, channels (initialized, not just zeroed)
- Use `new` or `&T{}` for pointer allocation (zeroed)
- Preallocate with capacity hint when size is known: `make([]T, 0, n)`
- Use "comma ok" for safe map access: `val, ok := m[key]`

```go
// Good — preallocate with known capacity
results := make([]Result, 0, len(items))
for _, item := range items {
    results = append(results, process(item))
}

// Good — comma ok for map
if val, ok := cache[key]; ok {
    return val, nil
}
return zero, ErrNotFound
```

## Embedding

- Embed interfaces in interfaces for composition: `ReadWriter` = `Reader` + `Writer`
- Embed structs for method promotion — not for inheritance, for composition
- Embedded type's methods have the embedded type as receiver, not the outer type

```go
// Good — interface composition
type ReadWriter interface {
    io.Reader
    io.Writer
}

// Good — struct embedding for reuse
type Server struct {
    *log.Logger
    config Config
}

// server.Println() works — promoted from embedded Logger
```

## Dependencies

Preferred packages:

| Purpose | Package |
|---------|---------|
| HTTP server | `net/http` (stdlib) or `github.com/go-chi/chi/v5` |
| Serialization | `encoding/json` (stdlib) |
| Database | `github.com/jmoiron/sqlx` + `github.com/jackc/pgx/v5` (PostgreSQL) |
| Migrations | `github.com/pressly/goose/v3` |
| Redis/Valkey | `github.com/valkey-io/valkey-glide/go/v2` |
| gRPC | `google.golang.org/grpc` + `google.golang.org/protobuf` |
| Logging | `log/slog` (stdlib) |
| Testing | `testing` (stdlib) + `github.com/stretchr/testify` |
| UUID | `github.com/google/uuid` |
| CLI | `github.com/spf13/cobra` or `github.com/urfave/cli/v2` |
| K8s | `k8s.io/client-go`, `sigs.k8s.io/controller-runtime` |
