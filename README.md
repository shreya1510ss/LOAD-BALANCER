# Load Balancer in Go

A lightweight HTTP load balancer built from scratch in Go. It sits in front of multiple backend servers, distributes incoming requests across them, and automatically detects when a server goes down.

---

## How It Works

```
Client Request
      |
      ▼
Load Balancer (:8080)
      |
      ├──▶ Server 1 (:8081)
      ├──▶ Server 2 (:8082)
      └──▶ Server 3 (:8083)
```

When a request comes in, the load balancer picks one of the healthy backend servers using the configured algorithm, forwards the request using a reverse proxy, and streams the response back to the client. The client only ever talks to port `8080` — it has no idea which backend handled the request.

A background health checker pings every server every 10 seconds. If a server stops responding, it is marked unhealthy and removed from rotation automatically. Once it recovers, it starts receiving traffic again.

---

## Algorithms

### Round Robin (default)

Distributes requests one by one across all servers in a fixed cycle.

```
Request 1 → Server 1
Request 2 → Server 2
Request 3 → Server 3
Request 4 → Server 1
...
```

Best for: workloads where all requests take roughly the same time and all servers have equal capacity.

---

### Least Connections

Routes each new request to whichever server currently has the fewest active connections.

```
Server 1: 10 active connections
Server 2: 3 active connections  ← picked
Server 3: 7 active connections
```

Best for: workloads where some requests are slow or expensive — avoids piling new work onto an already busy server.

---

### IP Hash

Hashes the client's IP address and maps it to a specific server. The same client always hits the same server.

```
Client 192.168.1.10  → always Server 2
Client 192.168.1.20  → always Server 1
Client 192.168.1.30  → always Server 3
```

Best for: when the backend stores session data in memory and needs the same client to always reach the same instance.

---

## Switching Algorithms

Open `main.go` and change the `algorithm` line inside `main()`:

```go
// Round Robin (default)
algorithm := RoundRobin

// Least Connections
algorithm := LeastConnections

// IP Hash
algorithm := IPHash
```

---

## Project Structure

```
LoadBalancer/
├── main.go              # Load balancer (runs on :8080)
├── Dockerfile
├── docker-compose.yml
├── server1/
│   ├── main.go          # Backend server 1 (:8081)
│   └── Dockerfile
├── server2/
│   ├── main.go          # Backend server 2 (:8082)
│   └── Dockerfile
├── server3/
│   ├── main.go          # Backend server 3 (:8083)
│   └── Dockerfile
└── client/
    └── main.go          # Test client
```

---

## Running with Docker

**Requirements:** Docker Desktop installed and running.

### Start

```bash
docker compose up --build -d
```

### Verify all containers are up

```bash
docker ps
```

You should see four containers running: `loadbalancer`, `server1`, `server2`, `server3`.

### Test it

```bash
curl http://localhost:8080
curl http://localhost:8080
curl http://localhost:8080
```

Each response comes from a different backend server.

### Watch live logs

```bash
docker compose logs -f
```

### Simulate a server going down

```bash
# Stop one server
docker stop server1

# Requests now route only to server2 and server3
curl http://localhost:8080
curl http://localhost:8080

# Bring it back — it rejoins automatically after the next health check
docker start server1
```

### Stop everything

```bash
docker compose down
```

---

## Running Locally (without Docker)

Open four terminal windows:

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

## Health Checks

Every 10 seconds the load balancer sends a `GET /health` request to each backend. The response determines the server's status:

| Response | Action |
|---|---|
| `200 OK` | Server stays in rotation |
| Timeout / error / non-200 | Server removed from rotation |
| Recovers later | Server automatically added back |

If all servers are unhealthy at the same time, the load balancer returns `503 Service Unavailable`.
