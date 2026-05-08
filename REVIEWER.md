# PoisonTrace Comprehensive Reviewer

This document maps core CS/software concepts to real code in the PoisonTrace repository (Go backend + TypeScript/React frontend).

## 1) OOP Principles

### Encapsulation
Definition: Bundle state + behavior behind a public API.

Where:
- `internal/api/server.go` (`type Server`, `NewServer`, `Handler`, `handle*` methods)
- `internal/helius/client.go` (`type HTTPClient`, `NewHTTPClient`, `FetchEnhancedPage`)

Snippet:
```go
// internal/api/server.go
type Server struct {
	repo         ReadRepository
	cfg          config.Config
	exportSource exportspkg.DatasetSource
	exportRoot   string
	settings     SettingsStore
	exportJobs   ExportJobStore
	ops          OpsStore
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/overview", s.handleOverview)
	...
	return withCORS(withAuth(mux, s.cfg.APIBearerToken))
}
```

### Inheritance
Definition: Child type derives from parent type.

Where in this codebase:
- No classical class inheritance (Go does not use class inheritance; TS app uses composition/hooks, not class-based inheritance).

Related composition pattern:
```go
// internal/helius/client.go
type HTTPClient struct {
	baseURL    *url.URL
	apiKey     string
	httpClient *http.Client
	pageLimit  int
}
```

### Polymorphism
Definition: Same interface, multiple concrete implementations.

Where:
- `internal/storage/repository.go` interfaces consumed by pipeline/API.
- Runtime type assertions in `NewServer` to activate optional capabilities.

Snippet:
```go
// internal/storage/repository.go
type RunRepository interface {
	CreateIngestionRun(ctx context.Context, startedAt time.Time) (int64, error)
	FinalizeIngestionRun(...)
	...
}

// internal/api/server.go
if ds, ok := any(repo).(exportspkg.DatasetSource); ok {
	s.exportSource = ds
}
```

### Abstraction
Definition: Hide implementation details behind contracts.

Where:
- `ReadRepository`, `SettingsStore`, `ExportJobStore`, `OpsStore` in `internal/api/server.go`
- `helius.Client` in `internal/helius/client.go`

Snippet:
```go
// internal/helius/client.go
type Client interface {
	FetchEnhancedPage(ctx context.Context, walletAddress string, before string) (EnhancedPage, error)
}
```

## 2) Data Structures & Algorithms

### Arrays/Slices
Where:
- `internal/pipeline/core.go` and `internal/pipeline/fetch.go`

Snippet:
```go
result := FetchWindowResult{Transactions: make([]helius.EnhancedTransaction, 0, p.MaxTx)}
outcomes := make([]walletOutcome, 0, len(walletList))
```

### Maps / Hash Maps
Where:
- `internal/pipeline/core.go` (`map[string]counterparties.Counterparty`)
- `internal/pipeline/core.go` (`mergeReasons` dedupe map)

Snippet:
```go
cpState := make(map[string]counterparties.Counterparty)
uniq := make(map[string]struct{})
```

### Sets
Where:
- `apps/web/src/app/lib/status.ts` uses JS `Set` for status membership checks.

Snippet:
```ts
const knownRunStatuses = new Set<RunStatus>([
  "running", "succeeded", "partially_succeeded", "failed", "timed_out", "cancelled",
]);
```

### Sorting
Where:
- `internal/pipeline/orchestrator.go` (`sort.Strings`, `sort.Slice`)
- `internal/api/server.go` (`sort.Slice` export files)

Snippet:
```go
sort.Slice(outcomes, func(i, j int) bool {
	if outcomes[i].address == outcomes[j].address {
		return outcomes[i].errString() < outcomes[j].errString()
	}
	return outcomes[i].address < outcomes[j].address
})
```

### Searching / Membership
Where:
- `status.ts` set lookup, relation mapping checks in `counterparties/service.go`.

Snippet:
```ts
return knownRunStatuses.has(status as RunStatus) ? (status as RunStatus) : "unknown";
```

### Traversal / Iteration
Where:
- Pipeline normalization/materialization loops.

Snippet:
```go
for _, tx := range txs {
	normalized, err := transactions.NormalizeEnhancedTx(tx)
	...
	for _, tr := range normalized { ... }
}
```

### Queue/Channel-based Work Distribution
Where:
- `internal/pipeline/orchestrator.go` uses goroutines + buffered channel + semaphore channel.

Snippet:
```go
sem := make(chan struct{}, o.cfg.MaxConcurrentWallets)
outcomeCh := make(chan walletOutcome, len(walletList))
```

### Retry Algorithm + Exponential Backoff
Where:
- `internal/pipeline/fetch.go` (`fetchPageWithRetry`, `retryBackoff`).

Snippet:
```go
backoff := baseBackoff << shift
if backoff > maxBackoff { backoff = maxBackoff }
jitter := time.Duration((jitterSeed*37)%97) * time.Millisecond
```

## 3) TypeScript & Language Fundamentals

### Type aliases and union types
Where:
- `packages/contracts/src/index.ts`

Snippet:
```ts
export type RunStatus =
  | "running"
  | "succeeded"
  | "partially_succeeded"
  | "failed"
  | "timed_out"
  | "cancelled";
```

### Interfaces
Where:
- `packages/contracts/src/index.ts` (API contracts)

Snippet:
```ts
export interface PagedResponse<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
}
```

### Generics
Where:
- `PagedResponse<T>`
- `request<T>()` in `apiClient.ts`
- generic form field contexts in `form.tsx`

Snippet:
```ts
async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`);
  ...
  return response.json() as Promise<T>;
}
```

### Utility types
Where:
- `Pick<>` usage in UI components.

Snippet:
```ts
type PaginationLinkProps = {
  isActive?: boolean;
} & Pick<React.ComponentProps<typeof Button>, "size">;
```

### `as const`
Where:
- `apps/web/src/app/components/ui/chart.tsx`

Snippet:
```ts
const THEMES = { light: "", dark: ".dark" } as const;
```

### Type narrowing / guard-like behavior
Where:
- `status.ts` parsing unknown statuses to safe union.

Snippet:
```ts
if (parsed === "unknown") {
  return { label: "Unknown", tone: "destructive" };
}
```

### Enums / Decorators
Where in this codebase:
- No TS `enum` declarations.
- No TS decorators (`@...`) used.

## 4) Full Stack Concepts

### REST API design + HTTP methods
Where:
- `internal/api/server.go`: endpoints like `/api/overview`, `/api/candidates`, `/api/settings`.

Snippet:
```go
mux.HandleFunc("/api/overview", s.handleOverview)
...
if r.Method != http.MethodGet {
	methodNotAllowed(w)
	return
}
```

### Middleware
Where:
- `withCORS`, `withAuth` in `server.go`.

Snippet:
```go
func withAuth(next http.Handler, bearerToken string) http.Handler {
	...
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/healthz" { ... }
		...
		next.ServeHTTP(w, r)
	})
}
```

### Authentication / Authorization
Where:
- Bearer token gate in `withAuth`.

Snippet:
```go
const prefix = "Bearer "
if !strings.HasPrefix(auth, prefix) || strings.TrimSpace(strings.TrimPrefix(auth, prefix)) != token {
	writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
	return
}
```

### DB interactions
Where:
- `internal/storage/postgres_repository.go` raw SQL with upserts/transactions.

Snippet:
```go
tx, err := s.DB.BeginTx(ctx, nil)
...
ON CONFLICT (signature, transfer_fingerprint) DO UPDATE SET ...
```

### ORM vs query style
Where:
- No ORM found; uses `database/sql` and handwritten SQL.

### Environment/config management
Where:
- `internal/config/config.go` (`LoadFromEnv`, `Validate`).

Snippet:
```go
maxWalletsPerRun, err := getEnvInt("MAX_WALLETS_PER_RUN", 25)
...
if c.MaxConcurrentWallets > c.MaxWalletsPerRun {
	return errors.New("MAX_CONCURRENT_WALLETS must be <= MAX_WALLETS_PER_RUN")
}
```

### Schema and constraints
Where:
- `migrations/0001_phase0_core.sql`, `migrations/0002_phase1_detection.sql`.

Snippet:
```sql
UNIQUE(signature, transfer_fingerprint)
...
UNIQUE(wallet_sync_run_id, signature, transfer_index)
```

## 5) Design Patterns

### Repository Pattern
Where:
- `internal/storage/repository.go` interfaces + `PostgresStore` implementation.

Why:
- Decouples domain services from persistence details.

### Dependency Injection
Where:
- `NewOrchestrator(cfg, opts...)` and `With*` options in `orchestrator.go`.

Snippet:
```go
type Option func(*Orchestrator)
func WithRunRepository(repo storage.RunRepository) Option { ... }
```

### Functional Options Pattern
Where:
- Same as above (`WithRunRepository`, `WithWalletRunner`, etc.).

### Adapter Pattern
Where:
- `internal/helius/client.go` adapts external Helius HTTP wire format into internal `EnhancedPage`.

Snippet:
```go
func decodeEnhancedPage(raw []byte) (EnhancedPage, error) {
	if strings.HasPrefix(trimmed, "[") { ... }
	var wire struct { Transactions []EnhancedTransaction `json:"transactions"`; Before string `json:"before"` }
	...
}
```

### Strategy Pattern (function injection)
Where:
- `CoreSyncParams.ClassifyDust func(...) DustStatus` in `core.go`.

## 6) Async & Concurrency

### Goroutines + WaitGroup
Where:
- `internal/pipeline/orchestrator.go`.

Snippet:
```go
var wg sync.WaitGroup
for _, addr := range walletList {
	wg.Add(1)
	go func() { defer wg.Done(); ... }()
}
wg.Wait()
```

### Concurrency limiting (semaphore channel)
Where:
- `sem := make(chan struct{}, o.cfg.MaxConcurrentWallets)` in orchestrator.

### Async background job
Where:
- Export job worker in API.

Snippet:
```go
go s.runExportJob(jobID, runID, outDir)
```

### Promise/async-await (frontend)
Where:
- `apps/web/src/app/lib/apiClient.ts`.

Snippet:
```ts
async function request<T>(path: string): Promise<T> {
  const response = await fetch(`${API_BASE}${path}`);
  ...
}
```

### Cancellation / timeouts
Where:
- `context.WithTimeout` across API handlers, orchestrator, CLI.

### Error handling
Where:
- Wrapped errors (`fmt.Errorf("...: %w", err)`) and typed errors (`StatusError`, `retryExhaustedError`).

## 7) General CS Fundamentals

### Scope and closures
Where:
- Middleware closures and goroutine variable capture fix.

Snippet:
```go
for _, addr := range walletList {
	addr := addr // avoids capture pitfall
	go func() { ... }()
}
```

### References vs values
Where:
- Pointer fields (`*time.Time`, `*int`) used to represent optional data.

Snippet:
```go
type WalletSyncProgress struct {
	...
}
// and TS optional/nullable in contracts, e.g. completedAt: string | null
```

### Determinism/idempotency
Where:
- SQL unique keys and deterministic fingerprinting in normalization.

Snippet:
```go
TransferFingerprint: BuildTransferFingerprint(...)
```

### Big-O highlights
- Set/map membership in status parsing and dedupe: average O(1).
- Sorting wallet outcomes/files: O(n log n).
- Pagination/normalization loops: O(n) over fetched transactions.
- Retry loop: O(r) attempts where `r <= MaxRetries + 1`.

### Memory behavior
- Pre-allocation used in hot paths (`make(..., 0, len(...))`) to reduce reallocations.

Snippet:
```go
items := make([]map[string]any, 0, len(rows))
```

---

## Quick "What is not used" summary
- No classical OOP inheritance trees.
- No TypeScript decorators.
- No TS `enum` keyword (string unions used instead).
- No ORM layer; direct SQL is the persistence mechanism.
