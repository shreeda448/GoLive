package main

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/shreeda448/goLive/internal/executor"
)

type DeployRequest struct {
	DeployID string `json:"deploy_id"`
	RepoURL  string `json:"repo_url"`
}

type AsyncQueue chan DeployRequest

func NewAsyncQueue() AsyncQueue {
	return make(chan DeployRequest, 50)
}

type MyAsyncQ struct {
	asyncQ AsyncQueue
}

func (q *MyAsyncQ) DeployHandler(w http.ResponseWriter, r *http.Request) {
	var deployRequest DeployRequest
	err := json.NewDecoder(r.Body).Decode(&deployRequest)
	if err != nil {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	q.asyncQ <- deployRequest
	w.WriteHeader(http.StatusAccepted)
	w.Write([]byte("Deploy job accepted"))
}

func FeedDeployJobs(asyncQ AsyncQueue) {
	for {
		curDeployJob := <-asyncQ
		Worker(curDeployJob)
	}
}

func Worker(d DeployRequest) {
	outputDir := d.RepoURL

	err := executor.RunBuild(outputDir)
	if err != nil {
		log.Printf("error occured in building : %v", err)
	}
}
