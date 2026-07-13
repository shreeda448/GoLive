# GoLive: Distributed CI/CD Deployment Engine
GoLive is a custom-built, Vercel-like deployment engine. It allows users to submit a GitHub repository URL, dynamically clones and compiles the code inside an isolated Docker container, saves the build artifact to S3 cloud storage, and serves the live application through a custom Reverse Proxy.
##  🏗️ Architecture & How It Works

GoLive is built around an asynchronous, event-driven architecture.

- **The Request:** The React frontend sends a GitHub URL to the Go API. The server generates a unique DeployID, saves the QUEUED state to PostgreSQL, and pushes the job to a Go Channel (AsyncQueue).

- **Isolated Build:** A background Go routine picks up the job, communicates with the Docker Engine API, and spins up an ephemeral golang:alpine container. It clones the repo and compiles the binary.

-  **Real-Time Logs:** Using a custom io.Writer interface, the Docker container's stdout is intercepted and streamed in real-time to the React frontend via WebSockets.

- **Cloud Artifacts:** Once compiled, the binary is extracted from the container as a tar stream and uploaded to an S3-compatible Object Storage bucket (MinIO) using the AWS Go SDK v2.

- **Dynamic Execution & Proxy:** The engine downloads the artifact, assigns it a dynamic, unused OS port (via net.Listen on port 0), and executes it as a background process. Finally, a sync.RWMutex protected routing table is updated, allowing the built-in Reverse Proxy to tunnel public internet traffic directly to the hidden application port.

## 💻 Tech Stack

- **Backend Core:** Go (Golang), net/http/httputil, sync.RWMutex, goroutines/channels
- **Infrastructure:** Docker Engine API, AWS SDK v2, MinIO (S3 Object Storage), PostgreSQL
- **Frontend:** React (Vite), WebSockets
- **System Design Patterns:** Reverse Proxy, Async Message Queuing, Stateful Lifecycles

## 🧠 What I Learned Building This

This project pushed me far beyond standard CRUD applications and deep into System Design, Concurrency, and Networking.

### 1. System Design & Architecture

**Reverse Proxies:** Understood how API Gateways and platforms like Vercel route traffic. I implemented a proxy that catches requests at a single public door and secretly tunnels them to dynamically allocated hidden ports.

**Read vs. Write Heavy Workloads:** Upgraded from a standard sync.Mutex to a sync.RWMutex for the proxy routing table, allowing much more concurrent web requests to read the route map simultaneously without blocking each other than with the normal mutex.

**The 12-Factor App:** Avoided port collisions by injecting dynamic OS-assigned ports into the background processes via environment variables (cmd.Env).

### 2. Deep Dive into Go (Golang)

**Interfaces are Powerful:** I built a custom io.Writer struct to seamlessly intercept Docker standard output logs and write them directly into a WebSocket connection.

**Concurrency:** Used goroutines and buffered channels to build an asynchronous queue that prevents the main HTTP server thread from blocking during 3-minute Docker builds.

**OS-Level Execution:** Used the os/exec and os packages to manipulate file permissions (os.Chmod), execute detached background processes (cmd.Start() instead of cmd.Run()), and safely release OS file locks to prevent text file busy panics.

### 3. Networking & Cloud Storage

**WebSockets:** Upgraded standard HTTP requests to stateful TCP WebSocket connections to stream live terminal logs to the frontend.

**S3 Integration:** Utilized the AWS SDK v2 to interface with MinIO, learning about cloud credentials, bucket access policies, and object storage retrieval.
### 3. Docker & Containerization

**Programmatic Container Management:** Used the Docker Engine API (Docker SDK for Go) to programmatically pull images, create configurations, and start containers entirely from Go code, bypassing the CLI.

**Container Isolation & Lifecycles:** Gained a foundational understanding of how containers provide isolated, reproducible environments for building untrusted code, and how to safely track their lifecycle from creation to exit.

**File System Interactions:** Explored how container file systems work, specifically learning how to extract ephemeral build artifacts out of a container and stream them into permanent cloud storage.
## 🚀 Running it Locally

**Prerequisites:** Docker Desktop and Go 1.22+ installed.

1.Clone this repository.

2.Spin up the infrastructure (Postgres & MinIO):
```bash
docker compose up -d
```

3.Set your MinIO bucket to "Public" via the UI at http://localhost:9001 (Creds: admin / password@212). Create a bucket named golive-build-artifacts.

4.Start the Go backend:
```bash
cd backend
go run ./cmd/api
```

5.Start the React frontend:
```bash
cd golive-ui
npm run dev
```
