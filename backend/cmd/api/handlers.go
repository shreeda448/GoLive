package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"sync"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/shreeda448/goLive/internal/executor"
)

type LogWriter struct {
	DeployID string
}

type DeployResponse struct {
	DeployID string `json:"deploy_id"`
	RepoURL  string `json:"repo_url"`
	Status   State  `json:"status"`
}

type StatusResponse struct {
	Status State `json:"status"`
}

type DeployJob struct {
	DeployID string `json:"deploy_id"`
	RepoURL  string `json:"repo_url"`
}

type DeployRequest struct {
	RepoURL string `json:"repo_url"`
}

type AsyncQueue chan DeployJob

type State string

const (
	StateQueued   State = "QUEUED"
	StateSuccess  State = "SUCCESS"
	StateFailed   State = "FAILED"
	StateBuilding State = "BUILDING"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

var (
	clients      = make(map[string]*websocket.Conn)
	clientsMutex sync.Mutex
)

var (
	runningApps = make(map[string]string)
	proxyMutex  sync.RWMutex
)

func NewAsyncQueue() AsyncQueue {
	return make(chan DeployJob, 50)
}

type MyAsyncQ struct {
	asyncQ AsyncQueue
	db     *sql.DB
}

func (q *MyAsyncQ) DeployHandler(w http.ResponseWriter, r *http.Request) {
	var deployRequest DeployRequest
	err := json.NewDecoder(r.Body).Decode(&deployRequest)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	deployID := uuid.New().String()
	query := `INSERT INTO deployments (id,repo_url,status)  VALUES  ($1,$2,$3)`
	_, err = q.db.Exec(query, deployID, deployRequest.RepoURL, StateQueued)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	q.asyncQ <- DeployJob{
		deployID,
		deployRequest.RepoURL,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(DeployResponse{
		deployID,
		deployRequest.RepoURL,
		StateQueued,
	})
}

func FeedDeployJobs(asyncQ MyAsyncQ) {
	for curDeployJob := range asyncQ.asyncQ {
		go func(curJob DeployJob) {
			Worker(curJob, asyncQ.db)
		}(curDeployJob)
	}
}

func Worker(d DeployJob, db *sql.DB) {
	outputDir := d.RepoURL
	query := `UPDATE deployments SET status = $1 WHERE id = $2`
	_, err := db.Exec(query, "BUILDING", d.DeployID)
	if err != nil {
		log.Printf("db query execution error : %v", err.Error())
		return
	}
	myWriter := LogWriter{DeployID: d.DeployID}
	artifactURL, err := executor.RunBuild(outputDir, myWriter, d.DeployID)
	if err != nil {
		log.Printf("error occured in building : %v", err.Error())
		_, err := db.Exec(query, "FAILED", d.DeployID)
		if err != nil {
			log.Printf("db query execution error : %v", err.Error())
			return
		}
		return
	}

	freePort, err := getFreePort()
	if err != nil {
		log.Printf("error acquiring free port : %v", err.Error())
		return
	}
	err = executor.RunApplication(artifactURL, freePort, d.DeployID)
	if err != nil {
		log.Printf("error running the application : %v", err.Error())
		return
	}
	proxyMutex.Lock()
	runningApps[d.DeployID] = fmt.Sprintf("http://localhost:%d", freePort)
	proxyMutex.Unlock()

	successQuery := `UPDATE deployments SET status = $1, artifact_url = $2 WHERE id=$3`
	_, err = db.Exec(successQuery, "SUCCESS", artifactURL, d.DeployID)
	if err != nil {
		log.Printf("db query execution error : %v", err.Error())
		return
	}
}

func (q *MyAsyncQ) StatusHandler(w http.ResponseWriter, r *http.Request) {
	deployID := r.URL.Query().Get("id")
	query := `SELECT status FROM deployments WHERE id = $1`
	var status State
	err := q.db.QueryRow(query, deployID).Scan(&status)
	if err != nil {
		http.Error(w, "db error", http.StatusInternalServerError)
		return
	}
	jsonBody := StatusResponse{
		status,
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(jsonBody)
}

func (q *MyAsyncQ) LogsHandler(w http.ResponseWriter, r *http.Request) {
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		http.Error(w, "logging error", http.StatusInternalServerError)
		return
	}
	defer ws.Close()
	deployID := r.URL.Query().Get("id")
	clientsMutex.Lock()
	clients[deployID] = ws
	clientsMutex.Unlock()
	for {
		if _, _, err := ws.ReadMessage(); err != nil {
			break
		}
	}
	clientsMutex.Lock()
	delete(clients, deployID)
	clientsMutex.Unlock()
}

func (w LogWriter) Write(p []byte) (int, error) {
	clientsMutex.Lock()
	ws, exists := clients[w.DeployID]
	clientsMutex.Unlock()
	if exists {
		ws.WriteMessage(websocket.TextMessage, p)
	}
	return len(p), nil
}

func (q *MyAsyncQ) ProxyHandler(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	proxyMutex.RLock()
	urlString, ok := runningApps[id]
	proxyMutex.RUnlock()
	if !ok {
		http.Error(w, "404 Not Found", http.StatusNotFound)
		return
	}
	curAppURL, err := url.Parse(urlString)
	if err != nil {
		log.Printf("url parsing error : %v", err.Error())
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(curAppURL)
	prefix := fmt.Sprintf("/view/%s", id)
	http.StripPrefix(prefix, proxy).ServeHTTP(w, r)
}

func getFreePort() (int, error) {
	addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
	if err != nil {
		return 0, err
	}
	listener, err := net.ListenTCP("tcp", addr)
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
