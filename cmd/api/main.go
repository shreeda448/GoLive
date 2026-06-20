package main

import (
	"log"
	"net/http"
)

func main() {
	myAsyncQ := NewAsyncQueue()
	go func() {
		FeedDeployJobs(myAsyncQ)
	}()
	mux := http.NewServeMux()
	mq := &MyAsyncQ{
		myAsyncQ,
	}
	mux.HandleFunc("POST /deploy", mq.DeployHandler)

	log.Fatal(http.ListenAndServe(":8080", mux))
}
