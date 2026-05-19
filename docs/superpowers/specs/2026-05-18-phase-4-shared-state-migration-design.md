# Phase 4 Shared State Migration Design

## Goal

Fourth phase migration solves the business-state split between the legacy `api` process and the new microservices before moving more traffic. The phase should produce one real read-only cutover slice, not another route-level migration that reads independent in-memory or fake data.

## Current State

The migration has completed three auth-focused steps:

- `POST /api/auth/login` routes through `auth-service`.
- `POST /api/auth/send-code` routes through `auth-service`.
- Other `/api/*` and `/ws` traffic still goes to the legacy `api`.
- The legacy `api` accepts tokens issued by `auth-service`.
- Docker Compose runs `gateway`, `api`, `auth-service`, and `frontend` together.

The next migration blocker is shared state, not HTTP routing. The legacy `api` stores passenger, driver, order, dispatch, payment, review, and driver-location state in `store.MemoryStore`. The new `admin-service` currently returns a fixed driver snapshot, and the new `driver-service` uses its own independent `MemoryRepo`. Cutting admin or driver routes now would create a false migration: the gateway would route successfully, but the new service would not observe the same business data as the old API.

## Recommended Approach

Use a persistence-boundary-first migration, then validate it with one read-only service cutover.

1. Add a MySQL-backed implementation of the existing `store.Store` interface.
2. Keep legacy `api` HTTP routes and service-layer behavior unchanged while switching Compose to the shared store explicitly.
3. Convert `admin-service` from fake snapshots to read-only MySQL queries.
4. Cut only safe admin read routes through the gateway after tests prove `admin-service` can read data written by legacy `api`.

Do not migrate driver write flows, order write flows, dispatch write flows, or websocket traffic in this phase.

## Architecture

### Legacy API

`backend/cmd/api/main.go` should choose the store implementation from configuration:

- Default: `store.NewMemoryStore()` for unit tests and simple local runs.
- Explicit `STORE_BACKEND=mysql`: connect to MySQL and use the MySQL-backed store.

If `STORE_BACKEND=mysql` is set and MySQL cannot be reached or initialized, `api` must fail startup. It must not silently fall back to memory, because that recreates split state.

The existing HTTP handlers, domain state machine, and service constructors should continue to use the `store.Store` interface. This keeps the fourth phase focused on changing the state boundary rather than changing endpoint behavior.

### Shared Store

Create a MySQL-backed `store.Store` implementation that maps the existing domain model to the tables in `infra/schema.sql`:

- `accounts`
- `passenger_profiles`
- `driver_profiles`
- `vehicles`
- `ride_orders`
- `dispatch_tasks`
- `dispatch_attempts`
- `payment_orders`
- `reviews`

Driver location can remain a special case until a schema exists. The preferred fourth-phase option is to add a small `driver_locations` table because passenger location lookups and driver location updates are part of the existing protected API. If scope needs to shrink, keep location in memory and explicitly leave `/api/passenger/orders/:id/driver-location` and `/api/driver/location` out of the shared-state acceptance criteria.

### Admin Service

Replace `admin.Service` fixed data with a repository-backed read model. The service should support real read-only snapshots from MySQL, beginning with drivers and optionally orders once the shared store is in place.

The admin read model must not mutate driver audit state or order state in this phase. `POST /api/admin/drivers/:id/approve` should continue to route to legacy `api` until approval has a real shared-state implementation and tests.

### Gateway

Extend gateway clients with an optional `AdminBase`. When `AdminBase` is configured, route safe admin read endpoints to `admin-service`:

- `GET /api/admin/drivers`
- optionally `GET /api/admin/orders` after the admin read model supports orders

If `AdminBase` is empty, all admin traffic continues to legacy `api`. Admin write endpoints stay on legacy `api` regardless of `AdminBase` during this phase.

## Data Flow

In Docker Compose, `api` starts with `STORE_BACKEND=mysql` and a MySQL DSN. It seeds demo passenger and approved drivers into MySQL using idempotent logic equivalent to the current `SeedDemoData` behavior.

Business writes continue to enter through legacy `api`:

1. Passenger creates an order.
2. Dispatch selects an online idle driver from the shared store.
3. Driver accept, reject, arrive, start, and end flows update order and driver state in MySQL.
4. Payment and review flows write payment and review state in MySQL.
5. Admin read service queries the same MySQL tables and returns snapshots through its own process.

The successful data-flow proof is that data written by legacy `api` is visible through `admin-service` without copying memory, stubs, or fake seed data inside `admin-service`.

## Error Handling

The MySQL-backed store should preserve existing business error semantics where handlers depend on them:

- missing driver: `driver not found`
- missing order: `order not found`
- missing payment: `payment not found`
- unapproved driver going online: `driver not approved`
- unavailable driver assignment: `driver not available`
- invalid order transition: use `domain.CanTransitionOrder`

Infrastructure failures should return errors to the caller or fail startup when the application cannot safely run. The API should not hide infrastructure failures by using in-memory state under an explicit MySQL configuration.

## Testing Strategy

### Store Contract Tests

Introduce shared contract tests that run against both `MemoryStore` and `MySQLStore`. The contract should cover behavior currently relied on by HTTP and service layers:

- seed passenger and find by phone
- seed approved driver and find/list by phone
- driver online/offline status changes
- create and list passenger orders
- dispatch selects an online idle driver and records attempts
- accept updates order and driver state
- reject or timeout advances dispatch or fails when no candidate remains
- trip state transitions set timestamps and final amount
- payment creation and paid marking
- review creation updates order review status

The MySQL contract tests can use `testing` with a DSN from environment and skip when not configured, while Compose/integration validation should run them with a real MySQL container.

### Gateway Tests

Keep existing tests for auth route splitting and legacy fallbacks. Add tests proving:

- `GET /api/admin/drivers` routes to admin upstream when `AdminBase` is configured.
- `GET /api/admin/orders` routes to admin upstream only if included in this phase.
- admin write endpoints still route to legacy `api`.
- admin read endpoints fall back to legacy `api` when `AdminBase` is not configured.

### Compose Integration Tests

After Compose starts:

1. `POST /api/auth/send-code` still succeeds and rate limits.
2. `POST /api/auth/login` still returns a token.
3. A token can still access legacy protected endpoints.
4. Legacy `api` creates or exposes real driver/order data through MySQL.
5. `admin-service` reads that same data through the gateway on the cutover read endpoint.

## Acceptance Criteria

Fourth phase is complete when:

- Compose uses MySQL for legacy `api` business state under explicit configuration.
- Existing auth gateway behavior remains unchanged.
- Existing passenger and driver core flows still pass against the configured shared store.
- `admin-service` no longer returns fixed fake snapshots for the selected read endpoint.
- The gateway can route the selected admin read endpoint to `admin-service` and observe data written by legacy `api`.
- No driver write, order write, dispatch write, payment write, review write, or websocket traffic is prematurely cut to new services.

## Out of Scope

- Full driver-service migration.
- Full order-service migration.
- Kafka-driven consistency between services.
- Dual writes from `MemoryStore` to MySQL.
- Admin approval write migration.
- Production-grade migration tooling beyond the existing Compose schema initialization.
