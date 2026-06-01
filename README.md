# Load Balancer in Go

A lightweight HTTP load balancer built from scratch in Go, demonstrating core concepts of traffic distribution, reverse proxying, health monitoring, and safe concurrent access.

---

## What This Project Is

This project implements an HTTP load balancer that sits in front of multiple backend servers and distributes incoming client requests across them. The load balancer acts as the single entry point (port `8080`) while transparently forwarding requests to one of three backend servers running on ports `8081`, `8082`, and `8083`.

The goal is to understand how load balancers work internally — from connection tracking to reverse proxying to automatic server failure detection — and why specific Go concurrency primitives are critical to making it correct under real-world load.

---

## Project Structure

```
LoadBalancer/
├── main.go              # Load balancer core (entry point on :8080)
├── Dockerfile           # Docker image for the load balancer
├── docker-compose.yml   # Orchestrates all 4 containers
├── server1/
│   ├── main.go          # Backend server 1 (listens on :8081)
│   └── Dockerfile
├── server2/
│   ├── main.go          # Backend server 2 (listens on :8082)
│   └── Dockerfile
├── server3/
│   ├── main.go          # Backend server 3 (listens on :8083)
│   └── Dockerfile
└── client/
    └── main.go          # Test client that fires repeated requests to the LB
```

---

## How It Works

### 1. Server Pool (`ServerPool`)

All backend servers are registered in a `ServerPool`. Each server entry tracks:

```go
type Server struct {
    address     string
    weight      int
    connections int64  // live in-flight request count
    healthy     int32  // 1 = healthy, 0 = unhealthy
}
```

The pool uses `sync.Map` for concurrent-safe storage, and `GetServers()` returns only servers where `healthy == 1`.

---

### 2. Load Balancing Algorithms

Three algorithms are implemented — you can swap them in `main()`:

#### Round Robin (default)
```go
func RoundRobin(servers []*Server, _ *http.Request) *Server {
    idx := atomic.AddUint64(&rrCounter, 1) - 1
    return servers[idx % uint64(len(servers))]
}
```
Cycles through servers in order: 1 → 2 → 3 → 1 → 2 → 3 ...  
Simple and fair when all requests take roughly the same time.

#### Least Connections
```go
func LeastConnections(servers []*Server, _ *http.Request) *Server
```
Routes each new request to whichever server has the fewest active connections at that moment.  
Better than Round Robin when some requests are slow — avoids piling more work onto an already busy server.

#### IP Hash
```go
func IPHash(servers []*Server, r *http.Request) *Server
```
Hashes the client's IP address and maps it to a server. The same client always hits the same server.  
Useful when the backend needs session stickiness (e.g., in-memory session stores).

---

### 3. Reverse Proxy

Once a target server is chosen, the load balancer uses Go's built-in `httputil.NewSingleHostReverseProxy` to forward the request and stream the response back. The client sees only the load balancer — the backend identity is invisible.

```
Client → :8080 (Load Balancer) → :8081/:8082/:8083 (Backend)
                               ← response streamed back ←
```

---

### 4. Health Checks

A background goroutine runs every **10 seconds** and sends an HTTP GET to `/health` on each registered server. If a server fails to respond with `200 OK`, its `healthy` flag is set to `0` atomically and it stops receiving traffic. When it recovers, the flag is set back to `1`.

```
Every 10s:
  for each server:
    GET /health
    200 OK  → atomic.StoreInt32(&s.healthy, 1)
    anything else → atomic.StoreInt32(&s.healthy, 0)
```

If all servers are unhealthy at once, the load balancer returns `503 Service Unavailable`.

---

## Why `sync/atomic` — and What Breaks Without It

### The problem: data races under concurrency

Go's HTTP server handles every incoming request in its own goroutine. With hundreds of concurrent requests, multiple goroutines read and write shared fields on `Server` simultaneously. Without protection, this is a **data race** — undefined behavior in Go.

### `atomic.AddInt64` — connection counter

```go
atomic.AddInt64(&server.connections, 1)
defer atomic.AddInt64(&server.connections, -1)
```

**What it does:** Increments/decrements `connections` as a single indivisible CPU instruction — no goroutine can observe a partial update.

**Without atomic — what breaks:**

Imagine two goroutines both reading `connections = 5` at the same time, both adding 1, and both writing back `6`. The actual value should be `7` but is `6`. This is called a **lost update** — the counter is silently wrong. The Least Connections algorithm would now make routing decisions based on stale numbers, potentially overloading a server it thinks is idle.

With Go's race detector (`go run -race`), this would show up immediately as:
```
DATA RACE
Write at 0x... by goroutine 12
Previous write at 0x... by goroutine 8
```

---

### `atomic.AddUint64` — round robin counter

```go
idx := atomic.AddUint64(&rrCounter, 1) - 1
```

**What it does:** Atomically increments a global counter shared across all goroutines and returns the value before the increment — giving each request a unique, monotonically increasing index.

**Without atomic — what breaks:**

Two goroutines could read the same value of `rrCounter`, both compute the same index, and both route to the same server — breaking the round robin guarantee. Multiple requests would pile onto one server while others stay idle. Under high concurrency, the distribution would be random and uneven.

---

### `atomic.StoreInt32` / `atomic.LoadInt32` — healthy flag

```go
atomic.StoreInt32(&s.healthy, 0)  // health check goroutine writes
atomic.LoadInt32(&s.healthy)      // request goroutines read
```

**What it does:** Ensures that when the health check goroutine marks a server unhealthy, request goroutines immediately see the updated value — no CPU cache staleness, no stale reads.

**Without atomic — what breaks:**

On modern multi-core CPUs, each core has its own cache. A write on Core A (health check goroutine) is not guaranteed to be visible to Core B (request goroutine) without a memory barrier. Without `atomic`, a request goroutine could keep reading `healthy = 1` even after the health check has set it to `0` — routing traffic to a dead server. The bug would be intermittent and hardware-dependent, making it extremely hard to reproduce.

---

## Why `sync.Map` — and What Breaks Without It

### The problem: maps are not goroutine-safe in Go

Go's built-in `map` type has **no concurrency protection**. Reading and writing a regular map from multiple goroutines simultaneously causes a fatal runtime panic:

```
fatal error: concurrent map read and map write
```

### Why `sync.Map` over `map + sync.Mutex`?

A common solution is to wrap a map in a `sync.RWMutex`. That works, but has a cost: every read — including the read that happens on every single incoming request — must acquire the read lock. Under high request rates, goroutines end up queuing on the lock.

`sync.Map` is designed for the **read-heavy, write-rarely** access pattern this project has:
- **Reads** happen on every request (routing needs the server list)
- **Writes** happen only during health checks (every 10s) or server registration

It uses an internal lock-free fast path for reads using atomic pointers, falling back to a mutex only on writes. This means routing reads in the hot path are mostly contention-free.

**Without any protection — what breaks:**

```
fatal error: concurrent map read and map write
goroutine 1 [running]: runtime.throw(...)
```

The process crashes. Not eventually, not sometimes — immediately, under any real concurrent load.

**With a plain `map + RWMutex` instead:**

It would be correct but slower. Every request would block briefly to acquire `RLock()`. With 1000 concurrent requests, there is measurable lock contention. `sync.Map` avoids this on the read path entirely.

---

## Concurrency Model Summary

| Field | Type | Protected By | Why |
|---|---|---|---|
| `server.connections` | `int64` | `sync/atomic` | Incremented/decremented on every request by many goroutines |
| `server.healthy` | `int32` | `sync/atomic` | Written by health check goroutine, read by all request goroutines |
| `rrCounter` | `uint64` | `sync/atomic` | Shared counter incremented by every goroutine picking a server |
| `ServerPool.servers` | `sync.Map` | Built-in | Map accessed concurrently by all request and health check goroutines |

---

## Running with Docker (Recommended)

### Prerequisites
- Docker Desktop installed and running

### Start everything
```powershell
cd "c:\Users\SHREYA SHARMA\Desktop\LoadBalancer"
docker compose up --build -d
```

### Verify all 4 containers are up
```powershell
docker ps
```
Expected output:
```
NAMES          STATUS        PORTS
loadbalancer   Up N seconds  0.0.0.0:8080->8080/tcp
server1        Up N seconds  8081/tcp
server2        Up N seconds  8082/tcp
server3        Up N seconds  8083/tcp
```

### Test round robin
```powershell
curl http://localhost:8080
curl http://localhost:8080
curl http://localhost:8080
```
Each response comes from a different server.

### Watch live logs
```powershell
docker compose logs -f
```

### Simulate a server failure
```powershell
# Kill server1
docker stop server1

# Requests now only go to server2 and server3
curl http://localhost:8080
curl http://localhost:8080

# Bring it back
docker start server1
```

### Stop everything
```powershell
docker compose down
```

---

## Running Locally (without Docker)

```bash
# Terminal 1
cd server1 && go run main.go

# Terminal 2
cd server2 && go run main.go

# Terminal 3
cd server3 && go run main.go

# Terminal 4 — Load balancer
go run main.go
```

Then test:
```bash
curl http://localhost:8080
```

---

## What Has Been Built

| Component | Status | Details |
|---|---|---|
| Server pool with concurrent-safe storage | Done | `sync.Map` |
| Add / remove servers dynamically | Done | `AddServer` / `RemoverServer` |
| Round Robin algorithm | Done | Atomic counter, O(1) per request |
| Least Connections algorithm | Done | O(n) scan across active servers |
| IP Hash algorithm | Done | FNV hash of client IP |
| Atomic connection tracking | Done | `sync/atomic` — race-free counters |
| Atomic health flag | Done | `sync/atomic` — safe cross-goroutine visibility |
| Reverse proxy forwarding | Done | `httputil.ReverseProxy` |
| Periodic health checks | Done | Background goroutine every 10s |
| Auto-marking of unhealthy servers | Done | Servers flagged out, recover automatically |
| Docker support | Done | Multi-stage builds, `docker-compose.yml` |
| Three backend servers | Done | Ports 8081, 8082, 8083 |

---

## Possible Next Steps

- Weighted Least Connections (use the `weight` field already on `Server`)
- Graceful shutdown with in-flight request draining
- Metrics endpoint (`/metrics`) showing per-server connection counts and request rates
- Dynamic server registration via an API endpoint
- Circuit breaker pattern to stop retrying servers that keep failing
