# Transactions

NSQLite supports SQLite transactions over the RPC API. A transaction lets you group multiple writes atomically: either all succeed or all are rolled back.

## Lifecycle

A transaction follows four steps:

1. **BEGIN** — Start a transaction and get a transaction ID (`txId`)
2. **Execute queries** — Run read/write queries with the `txId`
3. **COMMIT** — Persist all changes
4. **ROLLBACK** — Discard all changes (instead of COMMIT)

```
┌───────┐
│ BEGIN │──→ returns txId
└───┬───┘
    │
    ▼
┌──────────┐
│ Query 1  │── uses txId
└──────────┘
    │
    ▼
┌──────────┐
│ Query 2  │── uses txId
└──────────┘
    │
    ├──────────────┐
    ▼              ▼
┌────────┐   ┌──────────┐
│ COMMIT │   │ ROLLBACK │── uses txId
└────────┘   └──────────┘
```

## Starting a Transaction

Send a `BEGIN TRANSACTION` query. The response includes a `txId` that you use in subsequent queries.

```bash
curl -X POST http://localhost:9876/rpc/Database/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-token" \
  -d '{"queries": [{"query": "BEGIN TRANSACTION"}]}'
```

Response:

```json
{
  "ok": true,
  "output": {
    "time": 0.002,
    "results": [
      {
        "type": "begin",
        "time": 0.002,
        "txId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      }
    ]
  }
}
```

The `txId` is a UUID that identifies your transaction session.

## Executing Queries Within a Transaction

Pass the `txId` in each query you want to execute inside the transaction.

```bash
curl -X POST http://localhost:9876/rpc/Database/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-token" \
  -d '{
    "queries": [
      {
        "query": "INSERT INTO users(name) VALUES(?)",
        "params": [{"value": {"text": "fiona"}}],
        "txId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      },
      {
        "query": "UPDATE accounts SET balance = balance - 100 WHERE user_id = 1",
        "txId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
      }
    ]
  }'
```

All queries in a transaction execute on the **single read-write connection** and are serialized.

## Committing

```bash
curl -X POST http://localhost:9876/rpc/Database/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-token" \
  -d '{"queries": [{"query": "COMMIT", "txId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}]}'
```

Response:

```json
{
  "ok": true,
  "output": {
    "time": 0.003,
    "results": [
      {
        "type": "commit",
        "time": 0.003
      }
    ]
  }
}
```

## Rolling Back

```bash
curl -X POST http://localhost:9876/rpc/Database/query \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer my-token" \
  -d '{"queries": [{"query": "ROLLBACK", "txId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"}]}'
```

## Idle Timeout

If a transaction has no activity within the configured idle window, it is automatically rolled back by the server.

| Variable                  | Default | Description                            |
| ------------------------- | ------- | -------------------------------------- |
| `NSQLITE_TX_IDLE_TIMEOUT` | `10s`   | Maximum idle time before auto-rollback |

Set this to a longer duration for long-running transactional workflows:

```bash
NSQLITE_TX_IDLE_TIMEOUT=60s
```

Valid time units: `ns`, `us`, `µs`, `ms`, `s`, `m`, `h`.

When a transaction times out, subsequent queries with the old `txId` receive:

```json
{
  "ok": true,
  "output": {
    "results": [
      {
        "type": "error",
        "time": 0.001,
        "error": "transaction not found or timed out, check your settings"
      }
    ]
  }
}
```

## Concurrency Model

NSQLite manages concurrency through two locks:

| Lock      | Scope       | Purpose                                     |
| --------- | ----------- | ------------------------------------------- |
| `txMu`    | Transaction | Ensures only one transaction runs at a time |
| `writeMu` | Write query | Ensures write queries are serialized        |

- **Only one transaction** can be active at any time. Other clients that attempt `BEGIN` will queue until the current transaction finishes (COMMIT or ROLLBACK) or times out.
- Read queries outside a transaction can run in parallel across multiple read-only connections.
- While a transaction is open, all queries (including reads) within it go through the single read-write connection.

## Error Messages

| Error                                                            | Meaning                                                |
| ---------------------------------------------------------------- | ------------------------------------------------------ |
| `transaction not found or timed out, check your settings`        | The `txId` is invalid, already completed, or timed out |
| `transaction ID does not match the currently active transaction` | The `txId` does not match the open transaction         |
| `cannot start a transaction within a transaction`                | Called `BEGIN` while a transaction is already open     |
| `transaction ID is required for this operation`                  | Used COMMIT, ROLLBACK, or END without a `txId`         |

## Best Practices

- **Always use a `txId`** with COMMIT and ROLLBACK queries. The server needs it to identify which transaction to finalize.
- **Keep transactions short.** Idle transactions block other writers.
- **Always COMMIT or ROLLBACK.** Unclosed transactions hold resources and block other clients until the idle timeout fires.
- **Include the `txId` on every query** inside the transaction. Read queries without a `txId` execute outside the transaction on the read pool and if they are write operations, they wait until the active transaction finishes.
