# AP2 – Order & Payment Platform

A two-service platform evolving across three assignments:

- **Assignment 1** — REST communication, Clean Architecture, PostgreSQL
- **Assignment 2** — gRPC migration, contract-first proto generation, server-side streaming
- **Assignment 3** — Event-Driven Architecture with RabbitMQ, Notification Service

---

## Architecture Decisions

### Bounded Contexts

| Service | Owns | Database |
|---|---|---|
| Order Service | Order lifecycle & state | `orders_db` (PostgreSQL) |
| Payment Service | Payment authorization | `payments_db` (PostgreSQL) |
| Notification Service | Email/notification dispatch | None (in-memory idempotency store) |

Services **never share** tables, schemas, or internal models. Each defines its own domain types — the `PaymentEvent` in `notification-service/internal/domain` is independent of the Payment Service's `Payment` struct.

### Clean Architecture Layers

```
cmd/main/main.go          ← Composition Root (manual DI)
internal/
  domain/                 ← Entities, invariants, domain errors (no framework deps)
  usecase/                ← Business logic, depends only on interfaces (ports.go)
  repository/             ← DB adapters, gRPC client adapters (infrastructure)
  transport/http/         ← Gin handlers (thin delivery layer)
  transport/grpcserver/   ← gRPC server handlers
  messaging/              ← RabbitMQ producer / consumer (infrastructure)
```

Business rules (amount limits, status transitions, idempotency) live exclusively in `usecase/`. Handlers only parse requests and call use cases.

### gRPC Contract-First Flow (Assignment 2)

1. **Proto Repo** (`akmazhito/protos`) — contains `.proto` files only
2. **GitHub Actions** — on push to `proto/**`, runs `protoc` and pushes `.pb.go` files to:
3. **assignment1_2/ap2 Repo** (`akmazhito/assignment1_2/ap2`) — services import via `go get github.com/akmazhito/assignment1_2/ap2@v1.0.0`

This enforces contract versioning and prevents drift between services.

### Event Flow (Assignment 3)

```
Order Service ──gRPC──► Payment Service ──publish──► RabbitMQ ──consume──► Notification Service
                              │                         │
                         payments_db              payment.completed queue
```

The Notification Service knows nothing about Order or Payment services directly. It only consumes a JSON payload from a queue.

---

## Reliability Design

### At-Least-Once Delivery

- `autoAck: false` — messages are only acknowledged **after** the email log is printed
- Queues are `durable: true` — survive broker restarts
- Messages published with `DeliveryMode: Persistent`

### Idempotency (A1 Bonus + A3 Requirement)

**Order Service** — `Idempotency-Key` header on `POST /orders`. If an order with the same key already exists in the DB, the existing order is returned with HTTP 200 (no duplicate payment call).

**Notification Service** — tracks processed `MessageId` values (from AMQP header) in a `sync.Map`. If the same message arrives twice, the second delivery is ACK'd silently without printing a log.

### Dead Letter Queue (A3 Bonus)

Exchange topology:
```
payments (direct) ──► payment.completed queue
                              │ (on reject/max-retries)
                              ▼
               payments.dlx (direct) ──► payment.completed.dlq
```

To simulate DLQ routing: POST an order with `order_id: "dlq-test"`. The consumer will reject it after 3 attempts and it will appear in `payment.completed.dlq`.

### Failure Handling — Payment Service Unavailable

If Payment Service gRPC call fails with `codes.Unavailable` or `codes.DeadlineExceeded`:
- Order is marked **`Failed`** (not `Pending`)
- HTTP 503 is returned to the caller
- **Reasoning**: marking as Failed (rather than leaving Pending) allows idempotent retry via `Idempotency-Key` — the client can safely resubmit the same request and get a fresh payment attempt. A stuck `Pending` order would block the idempotency check.

### Graceful Shutdown

All three services listen for `SIGINT`/`SIGTERM`, then:
1. Stop accepting new connections (`grpcSrv.GracefulStop()`, `httpSrv.Shutdown(ctx)`)
2. Finish in-flight requests within a 10-second deadline
3. Close DB and AMQP connections

---

## Running Locally

```bash
docker-compose up --build
```

All services, databases, and RabbitMQ start automatically. Migrations run on first boot.

RabbitMQ management UI: http://localhost:15672 (guest/guest)

---

## API Examples

### Create order (normal)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -H "Idempotency-Key: order-abc-001" \
  -d '{"customer_id":"cust-1","item_name":"Book","amount":4999}'
```

### Create order — triggers declined payment (amount > 100000)
```bash
curl -X POST http://localhost:8080/orders \
  -H "Content-Type: application/json" \
  -d '{"customer_id":"cust-1","item_name":"Car","amount":150000}'
```

### Get order
```bash
curl http://localhost:8080/orders/<id>
```

### Cancel order
```bash
curl -X PATCH http://localhost:8080/orders/<id>/cancel
```

### Get payment by order
```bash
curl http://localhost:8081/payments/<order_id>
```

### Trigger DLQ (A3 bonus assignment1_2)
```bash
curl -X POST http://localhost:8081/payments \
  -H "Content-Type: application/json" \
  -d '{"order_id":"dlq-test","amount":100,"customer_email":"test@example.com"}'
# Watch notification-service logs — 3 retries then message moves to DLQ
```

### Watch order status stream (gRPC, A2)
```bash
grpcurl -plaintext -d '{"order_id":"<id>"}' \
  localhost:9090 order.v1.OrderService/SubscribeToOrderUpdates
```

---

## Bonus Features Summary

| Bonus | Location | Description |
|---|---|---|
| A1 — Idempotency-Key | `order-service/internal/usecase/order.go` | Duplicate POST /orders returns existing order |
| A2 — gRPC Interceptor | `payment-service/internal/transport/grpcserver/server.go` | Logs method, duration, status code |
| A3 — DLQ | `payment-service/internal/messaging/publisher.go` + consumer | Messages exceeding 3 retries route to DLQ |
