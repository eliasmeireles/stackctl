# stackctl Validation Tests

This document describes how to run validation tests for the stackctl CLI and infrastructure inside the Multipass instance.

## Overview

The validation test suite checks:
- stackctl CLI availability and version
- Database connectivity (PostgreSQL, MySQL, MongoDB, RabbitMQ)
- Vault service health
- K3s cluster health
- hapctl service status
- Database credentials feature (when implemented)

## Running Tests

### From Host Machine

Run all validation tests from your host machine:

```bash
make multipass-test
```

### Inside Multipass Instance

You can also run tests directly inside the instance:

```bash
# Enter the instance
make multipass-shell

# Run the test script
bash /home/ubuntu/workdir/../test-stackctl.sh
```

Or in one command:

```bash
multipass exec stackctl -- bash /home/ubuntu/workdir/../test-stackctl.sh
```

## Test Results

Test results are saved to `/tmp/stackctl-test-results/` inside the Multipass instance with timestamps.

Example output:

```
==========================================
stackctl CLI Validation Tests
Started at: Thu Mar 12 13:00:00 UTC 2026
==========================================

[1/10] Testing stackctl binary availability...
✓ PASS: stackctl binary found

[2/10] Testing stackctl version...
✓ PASS: stackctl version command

[3/10] Testing PostgreSQL connectivity...
✓ PASS: PostgreSQL pod running
✓ PASS: PostgreSQL port accessible

...

==========================================
Test Summary
==========================================
Tests Passed: 18
Tests Failed: 0
Total Tests:  18

🎉 All tests passed!
```

## Prerequisites

Before running tests, ensure:
1. Multipass instance `stackctl` is running
2. All database pods are deployed and running
3. stackctl CLI is built and installed in the instance
4. hapctl is configured and running

## Troubleshooting

### stackctl binary not found

Build and install the CLI inside the instance:

```bash
multipass exec stackctl -- bash -c "cd /home/ubuntu/workdir && go install ./cmd/stackctl"
```

### Database pods not running

Check pod status:

```bash
multipass exec stackctl -- kubectl get pods -n databases
```

Restart the setup if needed:

```bash
make multipass-recreate
```

### Network connectivity issues

Ensure IPv6 is disabled if you have connectivity problems:

```bash
sudo sysctl -w net.ipv6.conf.all.disable_ipv6=1
sudo sysctl -w net.ipv6.conf.default.disable_ipv6=1
```

## Test Coverage

| Test Category | Tests | Description |
|--------------|-------|-------------|
| CLI | 2 | Binary availability and version command |
| PostgreSQL | 2 | Pod status and port accessibility |
| MySQL | 2 | Pod status and port accessibility |
| MongoDB | 2 | Pod status and port accessibility |
| RabbitMQ | 3 | Pod status, AMQP port, and Management UI port |
| Vault | 2 | Pod status and API accessibility |
| K3s | 2 | Cluster accessibility and node readiness |
| hapctl | 2 | Service status and bind list command |
| Database Feature | 1 | Database credentials command (optional) |

**Total: ~18 tests**

## Adding New Tests

To add new tests, edit `@/home/eliasmeireles/workspace/personal/projects/stackctl/.dev/multipass/test-stackctl.sh:1` and follow the pattern:

```bash
echo ""
echo "[N/TOTAL] Testing your feature..."
if your_test_command; then
  test_result "your test description" 0
else
  test_result "your test description" 1
fi
```

The `test_result` function automatically tracks passed/failed tests and updates the summary.
