# Vault Mock Service — Testing Guide

## Overview

The `MockVault` is an **in-memory Vault mock** that replaces the real HashiCorp Vault server during tests. It implements all five Vault interfaces used by the `Applier`, enabling full integration testing without network access, Docker containers, or any external dependencies.

**Key features:**

- **Zero dependencies** — pure Go, no external services required
- **Thread-safe** — uses `sync.RWMutex`, safe for concurrent tests
- **Configurable permissions** — simulate full-access and limited users
- **State inspection** — helper methods to assert internal state after operations
- **Copy-on-read/write** — prevents accidental mutation between test steps

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│                       Applier                            │
│  (executes declarative Vault operations from ApplyConfig)│
├──────────┬───────────┬───────────┬───────────┬───────────┤
│ secrets  │ policies  │   auth    │  engines  │  logical  │
│ (R/W)    │ (CRUD)    │ (en/dis)  │ (mnt/umnt)│ (R/W/D)  │
└────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┴─────┬─────┘
     │           │           │           │           │
     ▼           ▼           ▼           ▼           ▼
┌─────────────────────────────────────────────────────────┐
│                      Interfaces                         │
│  SecretReadWriter │ PolicyManager │ AuthManager │ ...   │
└────────┬──────────┴───────┬───────┴──────┬──────────────┘
         │                  │              │
    ┌────▼────┐       ┌─────▼─────┐  ┌─────▼─────┐
    │MockVault│       │apiAdapter │  │apiAdapter │
    │(tests)  │       │(production)│  │(production)│
    └─────────┘       └───────────┘  └───────────┘
```

### Interfaces

All Vault operations are abstracted behind interfaces defined in `vault.go`:

| Interface          | Operations                                               | Used for                |
| ------------------ | -------------------------------------------------------- | ----------------------- |
| `SecretReadWriter` | `ReadSecret`, `WriteSecret`                              | KV v2 secrets           |
| `PolicyManager`    | `ListPolicies`, `GetPolicy`, `PutPolicy`, `DeletePolicy` | Vault policies          |
| `AuthManager`      | `EnableAuth`, `DisableAuth`, `ListAuth`                  | Auth methods            |
| `EngineManager`    | `MountEngine`, `UnmountEngine`, `ListEngines`            | Secrets engines         |
| `LogicalWriter`    | `Write`, `Read`, `Delete`, `List`                        | Roles and generic paths |

`MockVault` implements **all five interfaces** in a single struct, so it can be passed to all slots of the `Applier`.

## Permission Model

The mock supports configurable permissions via the `Permission` struct:

```go
type Permission struct {
    Read   bool  // ReadSecret, ListPolicies, GetPolicy, ListAuth, ListEngines, Read, List
    Write  bool  // WriteSecret, PutPolicy, EnableAuth, MountEngine, Write
    Delete bool  // DeletePolicy, DisableAuth, UnmountEngine, Delete
}
```

**Built-in presets:**

| Preset                 | Read | Write | Delete | Use case                          |
| ---------------------- | ---- | ----- | ------ | --------------------------------- |
| `FullPermission()`     | ✅    | ✅     | ✅      | Admin user — can do everything    |
| `ReadOnlyPermission()` | ✅    | ❌     | ❌      | Limited user — can only read/list |

When a permission is denied, the mock returns `ErrPermissionDenied`, which propagates through the `Applier` and can be checked with `errors.Is()`.

**Custom permissions** are also possible:

```go
// User that can read and write, but cannot delete
writeOnly := vault.Permission{Read: true, Write: true, Delete: false}
mock := vault.NewMockVault(writeOnly)
```

## How to Run Tests

### Run all vault tests (unit + integration)

```bash
go test ./cmd/stackctl/internal/feature/vault/ -v
```

### Run only integration tests (mock-based)

```bash
go test ./cmd/stackctl/internal/feature/vault/ -v -run "TestApplyFullPermission|TestApplyReadOnlyPermission"
```

### Run only unit tests (pure helpers)

```bash
go test ./cmd/stackctl/internal/feature/vault/ -v -run "TestResolveSecretValue|TestResolvePolicyRules|TestBuildRoleData"
```

### Run a specific subtest

```bash
go test ./cmd/stackctl/internal/feature/vault/ -v -run "TestApplyFullPermission/given_full_config"
```

### Run with coverage report

```bash
go test ./cmd/stackctl/internal/feature/vault/ -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
open coverage.html
```

### Run with race detector

```bash
go test ./cmd/stackctl/internal/feature/vault/ -race -v
```

## Test Structure

### Files

| File                        | Description                                                            |
| --------------------------- | ---------------------------------------------------------------------- |
| `mock_vault.go`             | `MockVault` implementation with permission checks and state inspection |
| `apply_test.go`             | **12 unit tests** for pure helper functions                            |
| `apply_integration_test.go` | **21 integration tests** using `MockVault`                             |

### Test Suites

#### `TestApplyFullPermission` — Full-access user (11 tests)

Tests that a user with full permissions can perform all CRUD operations:

| Test                                                         | What it verifies                                        |
| ------------------------------------------------------------ | ------------------------------------------------------- |
| `given full config then applies all operations in order`     | End-to-end: engines → auth → policies → roles → secrets |
| `given secrets add update delete then applies sequentially`  | Secret lifecycle: add 3 keys → update 1 → delete 1      |
| `given policy add update delete then manages lifecycle`      | Policy lifecycle: add → update rules → delete           |
| `given auth enable and disable then manages lifecycle`       | Auth method enable → disable                            |
| `given engine enable and disable then manages lifecycle`     | Engine mount (kv-v2 → kv with version=2) → unmount      |
| `given role add and delete then manages lifecycle`           | Role write with K8s fields → delete                     |
| `given auth enable with default path then uses type as path` | Auth path defaults to type name when omitted            |
| `given empty config then no error`                           | Empty `ApplyConfig{}` is a no-op                        |
| `given secrets path missing then returns error`              | Validates `secrets.path` is required                    |
| `given role with no parameters then returns error`           | Validates role must have at least one field             |
| `given role with unknown action then returns error`          | Validates action must be add/update/delete              |

#### `TestApplyReadOnlyPermission` — Limited user (10 tests)

Tests that a read-only user is blocked from all write/delete operations:

| Test                         | What it verifies                                     |
| ---------------------------- | ---------------------------------------------------- |
| `secrets write fails`        | Cannot add secrets                                   |
| `policy write fails`         | Cannot create policies                               |
| `policy delete fails`        | Cannot delete policies                               |
| `auth enable fails`          | Cannot enable auth methods                           |
| `auth disable fails`         | Cannot disable auth methods                          |
| `engine mount fails`         | Cannot mount engines                                 |
| `engine unmount fails`       | Cannot unmount engines                               |
| `role write fails`           | Cannot create roles                                  |
| `role delete fails`          | Cannot delete roles                                  |
| `empty config then no error` | Empty config is still a no-op (no permission needed) |

## Writing New Tests

### Basic pattern

```go
func TestMyFeature(t *testing.T) {
    t.Run("given some condition then expected result", func(t *testing.T) {
        // 1. Create mock with desired permissions
        mock := vault.NewMockVault(vault.FullPermission())

        // 2. Wire the Applier using the mock for all interfaces
        applier := vault.NewApplierFromInterfaces(mock, mock, mock, mock, mock)

        // 3. Build your ApplyConfig
        cfg := &vault.ApplyConfig{
            Secrets: &vault.SecretsConfig{
                Path: "secret/data/myapp",
                Add: []vault.SecretKVEntry{
                    {Name: "DB_HOST", Value: "localhost"},
                },
            },
        }

        // 4. Execute
        err := applier.Apply(cfg)
        if err != nil {
            t.Fatalf("unexpected error: %v", err)
        }

        // 5. Assert state using inspection helpers
        secrets := mock.GetSecrets("secret/data/myapp")
        if secrets["DB_HOST"] != "localhost" {
            t.Errorf("expected DB_HOST='localhost', got %v", secrets["DB_HOST"])
        }
    })
}
```

### Using test helpers (from `apply_integration_test.go`)

```go
// Creates MockVault(FullPermission) + Applier in one call
mock, applier := newFullApplier()

// Creates MockVault(ReadOnlyPermission) + Applier (no mock reference needed)
applier := newReadOnlyApplier()

// Assert no error
requireNoError(t, applier.Apply(cfg))

// Assert permission denied
requirePermissionDenied(t, applier.Apply(cfg))
```

### State inspection helpers

After running operations, inspect the mock's internal state:

```go
mock.GetSecrets("secret/data/app")           // map[string]interface{} — secret key/values
mock.GetPolicies()                            // map[string]string — policy name → HCL rules
mock.GetAuths()                               // map[string]AuthMount — path → {Type, Description}
mock.GetEngines()                             // map[string]EngineMount — path → {Type, Description, Options}
mock.GetLogical("auth/k8s/role/my-role")      // map[string]interface{} — role data
```

### Testing permission denied scenarios

```go
func TestCustomPermission(t *testing.T) {
    // Write-only: can create but not delete
    perm := vault.Permission{Read: true, Write: true, Delete: false}
    mock := vault.NewMockVault(perm)
    applier := vault.NewApplierFromInterfaces(mock, mock, mock, mock, mock)

    // This should succeed
    err := applier.Apply(&vault.ApplyConfig{
        Policies: &vault.PoliciesConfig{
            Add: []vault.PolicyEntry{{Name: "pol", Rules: "path {}"}},
        },
    })
    // err == nil

    // This should fail
    err = applier.Apply(&vault.ApplyConfig{
        Policies: &vault.PoliciesConfig{
            Delete: []string{"pol"},
        },
    })
    // errors.Is(err, vault.ErrPermissionDenied) == true
}
```

## In-Memory Storage Model

The mock stores data in five separate maps, mirroring Vault's resource types:

| Map        | Key                           | Value                                     | Vault equivalent       |
| ---------- | ----------------------------- | ----------------------------------------- | ---------------------- |
| `secrets`  | path (e.g. `secret/data/app`) | `map[string]interface{}`                  | KV v2 secret data      |
| `policies` | policy name                   | HCL rules string                          | `sys/policy/<name>`    |
| `auths`    | mount path                    | `AuthMount{Type, Description}`            | `sys/auth/<path>`      |
| `engines`  | mount path                    | `EngineMount{Type, Description, Options}` | `sys/mounts/<path>`    |
| `logical`  | full path                     | `map[string]interface{}`                  | `<path>` (roles, etc.) |

All read/write operations return **copies** of the stored data to prevent accidental mutation between test steps.
