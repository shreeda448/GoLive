package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/rs/cors"
)

func main() {
	db, err := sql.Open("pgx", "postgres://shreeda:iuytrewa@127.0.0.1:5433/go_live?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS deployments (
		id TEXT PRIMARY KEY,
		repo_url TEXT,
		status TEXT,
		artifact_url TEXT
	)`)
	if err != nil {
		log.Fatal("failed to create table: ", err)
	}
	myAsyncQ := NewAsyncQueue()
	mq := &MyAsyncQ{
		asyncQ: myAsyncQ,
		db:     db,
	}
	go func() {
		FeedDeployJobs(*mq)
	}()

	log.Print("connection made successfully")
	mux := http.NewServeMux()
	mux.HandleFunc("POST /deploy", mq.DeployHandler)
	mux.HandleFunc("GET /status", mq.StatusHandler)
	mux.HandleFunc("GET /logs", mq.LogsHandler)
	mux.HandleFunc("GET /view/{id}", mq.ProxyHandler)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:5173"},
		AllowedMethods: []string{"GET", "POST"},
		AllowedHeaders: []string{"Content-Type"},
	})
	corsEnabledMux := c.Handler(mux)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%s", port), corsEnabledMux))
}
