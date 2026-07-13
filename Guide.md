# Building GoLive: A Distributed CI/CD Deployment Engine

This guide walks you through the step-by-step process of building a custom deployment engine from scratch. You will build a Go backend that dynamically compiles code inside Docker containers, streams real-time logs via WebSockets, stores artifacts in S3, and routes traffic using a custom Reverse Proxy.

## Phase 1: The Database Infrastructure

**Objective:** Spin up a local PostgreSQL database, connect to it with Go, and execute a query.

### Task 1: The Docker Compose Setup

Create a docker-compose.yml file in the root of your project to spin up PostgreSQL:
```
version: '3.8'  
services:  
db:  
image: postgres:15-alpine  
environment:  
POSTGRES_USER: postgres  
POSTGRES_PASSWORD: password  
POSTGRES_DB: vercel_clone  
ports:  
\- "5432:5432"
```
Run ``` docker compose up -d ``` to start the database in the background.

### Task 2: The Go Driver & Schema

Install the Postgres driver: ```go get github.com/lib/pq (or github.com/jackc/pgx/v5)```

Create your connection and initialize the table:
```
CREATE TABLE IF NOT EXISTS deployments (  
id TEXT PRIMARY KEY,  
repo_url TEXT NOT NULL,  
status TEXT NOT NULL,  
artifact_url TEXT  
);
```
## Phase 2: State Tracking API

**Objective:** Create the core API endpoints to accept deployment jobs and generate unique IDs.

### Task 1: The Deploy Endpoint

When a user hits POST /deploy with a valid repo_url, the server must immediately return:

- **Status Code:** ```202 Accepted```
- **Body:** ```{"deploy_id": "&lt;unique-uuid&gt;", "status": "QUEUED"}```

### Task 2: Database Integration

Right after generating the deployID (using github.com/google/uuid), use your database connection to insert the new row into your deployments table with the initial status of "QUEUED".

### Task 3: The Async Queue

Create a Go Channel to handle background jobs without blocking the HTTP response:
```
type DeployJob struct {  
DeployID string  
RepoURL string  
}  
type AsyncQueue chan DeployJob
```
Push the DeployJob into the channel inside your DeployHandler, and have a background Worker goroutine listen to this channel to process the builds.

## Phase 3: Lifecycle & Status Endpoint

**Objective:** Track BUILDING, SUCCESS, and FAILED states, and provide an endpoint to check it.

### Task 1: Update the States

Inside your background Worker function:

- Right _before_ calling your Docker build function, execute: UPDATE deployments SET status = 'BUILDING' WHERE id = \$1
- Right _after_ the build finishes, check the error. Update the status to 'FAILED' if it crashed, or 'SUCCESS' if it compiled.

### Task 2: The Status Endpoint

Create GET /status?id=&lt;deploy_id&gt;.

Extract the ID from the URL, run a SELECT status FROM deployments WHERE id = \$1 query, and return the status to the user as JSON.

## Phase 4: The React Frontend & CORS

**Objective:** Build a modern UI to accept GitHub URLs and poll for live status updates.

### Task 1: Initialize the React App

Run``` npm create vite@latest golive-ui -- --template react``` and create a simple UI with a Text Input and a "Deploy" button.

### Task 2: Defeat CORS

In your Go backend, install github.com/rs/cors. Wrap your router to allow cross-origin requests from your React frontend (<http://localhost:5173>).

### Task 3: Polling for Updates

When the user clicks Deploy, save the returned deploy_id in React state.

Use a useEffect hook with setInterval to ping the Go GET /status endpoint every 2 seconds. Clear the interval when the status reaches SUCCESS or FAILED.

## Phase 5: Live Build Logs (WebSockets)

**Objective:** Stream the stdout from the Docker container directly to the React frontend in real-time.

### Task 1: The Go Setup

Install github.com/gorilla/websocket. Create a GET /logs handler that upgrades the HTTP connection to a WebSocket.

Store active connections in a global map protected by a sync.Mutex.

### Task 2: The Custom Writer

Create a custom io.Writer in Go that writes incoming byte streams directly to the WebSocket connection associated with that specific DeployID.
```
func (w LogWriter) Write(p \[\]byte) (int, error) {  
clientsMutex.Lock()  
ws, exists := clients\[w.DeployID\]  
clientsMutex.Unlock()  
if exists {  
ws.WriteMessage(websocket.TextMessage, p)  
}  
return len(p), nil  
}
```
Pass this custom writer to the Docker SDK's stdcopy.StdCopy function so all container logs pipe to the browser.

### Task 3: The React Terminal

Connect to ws://localhost:8080/logs?id=&lt;deploy_id&gt; in React. Append incoming messages to an array and display them in a black &lt;pre&gt; terminal window.

## Phase 6: Cloud Storage (AWS S3)

**Objective:** Upload the compiled binary to cloud storage (MinIO/S3) instead of filling up the local hard drive.

### Task 1: The AWS SDK

Install the AWS SDK v2 for Go (github.com/aws/aws-sdk-go-v2). Spin up a local MinIO container or create a real AWS S3 bucket.

### Task 2: The Uploader

Read the compiled binary from the Docker container as a tar stream. Use the AWS SDK client.PutObject function to stream the file directly into your bucket.

### Task 3: Save the Artifact

Return the public URL of the uploaded file and update the deployments database table with the artifact_url on a successful build.

## Phase 7: The Live Server & Reverse Proxy

**Objective:** Execute the cloud artifact on a dynamic port and tunnel web traffic to it securely.

### Task 1: The Port Finder & Runner

Create a function that asks the OS for a random available port using ```net.Listen("tcp", "localhost:0")```.

Download the binary from S3, save it locally, make it executable (os.Chmod), and run it in the background using exec.Command. Pass the dynamic port in the environment variables (PORT=...).

### Task 2: The Global Registry

Create a high-performance routing table using a Read-Write Mutex:
```
var (  
runningApps = make(map\[string\]string)  
proxyMutex sync.RWMutex  
)
```
When the app starts successfully, map the DeployID to the local URL (e.g., <http://localhost:8432>).

### Task 3: The Reverse Proxy Handler

Create a final endpoint:``` GET /view/{id}/```.

When a request comes in, check the runningApps map. If found, use Go's built-in httputil.NewSingleHostReverseProxy to secretly forward the client's traffic to the background application port!
