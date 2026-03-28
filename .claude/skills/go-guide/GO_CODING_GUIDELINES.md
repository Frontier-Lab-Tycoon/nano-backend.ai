# Go Coding Guidelines

Directives for writing idiomatic Go code.

---

## Interface Design

- **Define interfaces at the consumer, not the provider.** A service that calls an external client should declare its own interface with only the methods it needs. The concrete implementation satisfies it implicitly.
- **Keep interfaces small.** 1–5 methods per interface. If an interface grows beyond that, split it by responsibility.
- **Return concrete types from constructors, accept interfaces in functions.** Constructors (`New*`) return `*ConcreteType`. Business logic functions accept interface parameters for testability.
- **Name interfaces by capability, not implementation.** Use `SyncClient`, `AuthClient`, `ChatClient` — not `GitHubClientInterface`.

## Error Handling

- **Wrap every error with context using `fmt.Errorf("doing X: %w", err)`.** Every call site that propagates an error must add what operation was attempted.
- **Use a custom error type with a private struct and public factory functions.** The struct implements the `error` interface plus domain-specific methods (e.g., `StatusCode()`, `Code()`). Expose only `Error(code, msg)` and `Errorf(code, format, args...)` as constructors — never the concrete type.
- **Extract error properties via interface assertion with comma-ok.** Define small extractor interfaces (`StatusCodeError`, `CodeError`) and helper functions that safely type-assert, falling back to a default on mismatch.
- **Tie business error codes to HTTP status codes in a single `ErrorCode` struct.** This eliminates scattered status-code decisions in handlers — the error itself knows its HTTP representation.
- **Declare sentinel errors at package level as `var`.** Use `var errNoClient = errdefs.Error(...)` for well-known, reusable error conditions.

## Type System

- **Use named types over raw primitives for domain concepts.** Define `type TaskManageTool string`, `type IssueStatus string`, etc. This prevents accidental interchange of semantically different strings.
- **Define typed constants with the named type.** `const Jira TaskManageTool = "jira"` — not bare `const Jira = "jira"`.
- **Provide `FromStr(string) (T, bool)` parsing functions.** Return `(value, ok)` tuple for safe conversion from external input, matching Go's comma-ok convention.
- **Use semantic type aliases for map keys/values.** `map[LablupEmail]GitHubUserName` is self-documenting and prevents key/value mix-ups that `map[string]string` allows.

## Struct Design

- **Distinguish required vs optional fields with value types vs pointers.** Required fields use value types (`string`, `int`). Optional fields use pointers (`*string`, `*time.Time`) where `nil` means "not set" — distinct from zero value.
- **Provide pointer helper functions** (`StringP`, `IntP`, `BoolP`) to eliminate `&` boilerplate when constructing struct literals with optional fields.
- **Group constructor parameters into an `Args` struct when there are 3+ parameters.** Named fields improve readability and allow non-breaking additions.
- **Place transformation methods on domain structs.** Methods like `Merge(other)`, `ShortPrint()`, `ToIssue()`, `FromIssue()` belong on the data type that owns the fields.

## Composition & Extension

- **Compose, never inherit.** Go has no inheritance. Extend behavior by holding another type as a field and delegating to it.
- **Use the decorator pattern to layer behavior onto interfaces.** Wrap an existing `echo.Binder` with a `validateBinder` that calls the original binder then runs validation — same interface, added behavior.
- **Implement polymorphism via interface + multiple concrete structs.** For example, a `UserQuery` interface with `EmailQuery`, `APIKeyQuery`, `GitHubUserNameQuery` structs each implementing `BuildQueryCondition()`. The repository accepts the interface, not a switch on query type.

## Dependency Injection & Construction

- **Use the fluent builder pattern for assembling dependency graphs.** `NewClients().MustWithGHAppClient(args).WithGHClient(args).MustWithJIRAClient(args)` — each method returns `*self` for chaining.
- **Prefix with `Must*` when failure is fatal (panic).** Use only during program initialization where recovery is impossible. Prefix with `With*` for infallible or error-returning setup.
- **Apply the builder pattern consistently across layers.** Clients, Repositories, and Services all follow `New*() → MustWith*/With*` for uniform construction.
- **Inject factory functions as fields for deferred construction.** Store `func(token string) *Client` in a pool struct to create per-user clients on demand, rather than pre-creating all instances.

## Concurrency

- **Use `atomic.Bool` for simple boolean flags** (e.g., "is server closed"). Avoids mutex overhead for single-field state.
- **Use `sync.RWMutex` when reads vastly outnumber writes.** Protects shared resources (HTTP clients, caches) without blocking concurrent readers.
- **Start servers in goroutines, handle signals on the main goroutine.** Use `make(chan os.Signal, 1)` with `signal.Notify` and block on receive for graceful shutdown.
- **Set flags before triggering shutdown.** `closed.Store(true)` before `Shutdown()` — so the `Start()` goroutine can distinguish intentional stop from unexpected error.

## Context & Logging

- **Thread `context.Context` as the first parameter of every public method.** No exceptions. This carries cancellation, deadlines, and request-scoped data.
- **Accumulate structured log fields in context.** Use a pattern like `ctx = clog.Field(ctx, "issueKey", key)` so downstream calls automatically include upstream context in log output.
- **Log at decision points, not at every line.** Log when starting an operation, when a significant branch is taken, and when an error occurs — not every intermediate step.

## Package Structure

- **Organize by architectural layer, not by feature.**
  - `cmd/` — Entry points. Minimal code, just wiring.
  - `config/` — Configuration loading and validation.
  - `data/` — Domain models. Zero external dependencies.
  - `dto/` — Request/response DTOs. API contract boundary.
  - `errdefs/` — Error types and codes. Shared across layers.
  - `clients/` — External API integrations. One sub-package per provider.
  - `services/` — Business logic. Orchestrates clients and repositories.
  - `repository/` — Data persistence. Database-specific implementations in sub-packages.
  - `servers/` — HTTP handlers and middleware. Sub-packages per API version and domain.
  - `utils/` — Shared utilities. Keep minimal.
- **Dependencies flow inward.** `cmd → servers → services → {clients, repository} → data`. Inner layers never import outer layers.
- **One sub-package per external provider.** `clients/ghcli`, `clients/jiracli`, `clients/teamscli` — each owns its own types and API-specific logic.
- **Separate API-specific representations from domain models.** Each client package defines its own `Issue` struct with `FromIssue()`/`ToIssue()` conversion methods. The domain `data.Issue` stays clean.

## Testing

- **Use `t.Parallel()` in tests that don't share mutable state.** Enables concurrent test execution for faster feedback.
- **One test function per behavior, not per method.** Name tests by what they verify: `Test_ToIssue`, `TestIssue_FromGitHubIssue`.
- **Use `testify/assert` for readable assertions.** Prefer `assert.Equal(t, expected, actual)` over manual `if` checks.
- **Small consumer-side interfaces make mocking trivial.** If the interface has 2 methods, the mock has 2 methods. No heavyweight mock frameworks needed.

## Naming

- **Packages:** Short, singular, lowercase. `data`, `config`, `errdefs`. Sub-packages clarify implementation: `ghcli`, `dbrepo`.
- **Constructors:** `NewType(args)` returns `*Type`.
- **Builder methods:** `With*()` (safe) or `MustWith*()` (panics).
- **Conversion methods:** `FromX()` / `ToX()` for bidirectional type mapping.
- **Parsing functions:** `TypeFromStr(s string) (Type, bool)`.
- **Private sentinel errors:** `var errSomething = ...` (lowercase, unexported).
- **Public error codes:** `UserNotFoundCode`, `JiraAPIFailedCode` (exported, descriptive).
- **Private types with public interfaces:** `type err struct` (unexported) implements `error` (exported). Control construction through factory functions.
