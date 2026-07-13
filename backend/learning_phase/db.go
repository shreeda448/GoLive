package main

import (
	"database/sql"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func main() {
	db, err := sql.Open("pgx", "postgres://shreeda:iuytrewa@127.0.0.1:5432/go_live?sslmode=disable")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}
	log.Print("connection made successfully")
	q := `CREATE TABLE	IF NOT EXISTS deployments  (
	id TEXT PRIMARY KEY,
	repo_url  TEXT,
	status TEXT
	)`
	res, err := db.Exec(q)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("successfully created  the table  deployments :%v", res)
}
