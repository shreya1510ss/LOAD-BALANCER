# Load Balancer in Go

A lightweight HTTP load balancer built from scratch in Go, demonstrating core concepts of traffic distribution, reverse proxying, and health monitoring.

---

## What This Project Is

This project implements an HTTP load balancer that sits in front of multiple backend servers and distributes incoming client requests across them. The load balancer acts as the single entry point (port `8080`) while transparently forwarding requests to one of three backend servers running on ports `8081`, `8082`, and `8083`.

The goal is to understand how load balancers work internally — from connection tracking to reverse proxying to automatic server eviction on failure.

---

## Project Structure

```
LoadBalancer/
├── main.go          # Load balancer core (entry point on :8080)
├── server1/
│   └── main.go      # Backend server 1 (listens on :8081)
├── server2/
│   └── main.go      # Backend server 2 (listens on :8082)
├── server3/
│   └── main.go      # Backend server 3 (listens on :8083)
└── client/
    └── main.go      # Test client that fires repeated requests to the LB
```

---

## How It Works

### 1. Server Pool (`ServerPool`)

All backend servers are registered in a `ServerPool`, which uses a `sync.Map` for concurrent-safe storage. Each server entry tracks:
- `address` — host:port of the backend
- `weight` — reserved for future weighted algorithms
- `connections` — live count of in-flight requests (updated atomically)

### 2. Least Connections Algorithm

When a request arrives, the load balancer picks the backend with the **fewest active connections** at that moment. This naturally routes traffic away from slow or busy servers without any manual configuration.

```
Incoming request → check all servers → pick server with lowest connections → forward
```

The connection counter is incremented atomically when a request starts and decremented (via `defer`) when it finishes, keeping the count accurate under concurrency.

### 3. Reverse Proxy

Once a target server is chosen, the load balancer uses Go's built-in `httputil.NewSingleHostReverseProxy` to forward the request. The client sees only the load balancer — the backend identity is invisible.

### 4. Health Checks

A background goroutine runs every **10 seconds** and pings each registered server with an HTTP GET. If a server fails to respond with `200 OK` (due to timeout, connection refused, or any error), it is **automatically removed** from the pool. Unhealthy servers stop receiving traffic immediately.

```
Every 10s → ping each server → if unhealthy → remove from pool
```

If all servers are removed, the load balancer returns `503 Service Unavailable`.

### 5. Test Client

The `client/` directory contains a simple loop client that continuously fires requests at `localhost:8080` and prints how many requests have been served. It's used to manually observe the load balancer in action.

---

## Running the Project

Open four terminal windows:

```bash
# Terminal 1 — Start backend server 1
cd server1 && go run main.go

# Terminal 2 — Start backend server 2
cd server2 && go run main.go

# Terminal 3 — Start backend server 3
cd server3 && go run main.go

# Terminal 4 — Start the load balancer
go run main.go
```

Then optionally run the test client:

```bash
cd client && go run main.go
```

Or test manually:

```bash
curl http://localhost:8080/
```

---

## What Has Been Built

| Component | Status | Details |
|---|---|---|
| Server pool with concurrent-safe storage | Done | Uses `sync.Map` |
| Add / remove servers dynamically | Done | `AddServer` / `RemoverServer` |
| Least connections routing algorithm | Done | O(n) scan across active servers |
| Atomic connection tracking | Done | `sync/atomic` for race-free counters |
| Reverse proxy forwarding | Done | `httputil.ReverseProxy` |
| Periodic health checks | Done | Runs every 10s in a goroutine |
| Auto-eviction of unhealthy servers | Done | Failed servers removed from pool |
| Three backend servers | Done | Ports 8081, 8082, 8083 |
| Test client | Done | Fires sequential requests, prints count |

---

## Design Decisions

**Why `sync.Map` instead of a regular map with a mutex?**  
The server pool is read on every request and written only during health checks or server registration. `sync.Map` is optimized for this read-heavy, write-rarely pattern and avoids holding a coarse lock across the entire request path.

**Why Least Connections instead of Round Robin?**  
Round Robin distributes requests evenly by count, but ignores the fact that some requests take longer than others. Least Connections routes new requests to whichever server has the most capacity right now, making it more responsive to real workload differences.

**Why atomic counters for connections?**  
Multiple goroutines handle requests concurrently. Using `sync/atomic` to increment and decrement the connection count avoids introducing a mutex on the hot path of every request.

**Why remove unhealthy servers instead of marking them?**  
Simpler state — a server is either in the pool (healthy) or not. If a removed server recovers, it would need to be re-added manually or via a re-registration mechanism (a future improvement).

---

## Possible Next Steps

- Re-registration of recovered servers after health check passes
- Weighted Least Connections (use the `weight` field already on `Server`)
- Round Robin and IP Hashing algorithms
- Graceful shutdown with in-flight request draining
- Metrics endpoint (`/metrics`) showing per-server connection counts
