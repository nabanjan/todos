package main

import (
	"database/sql"
	"log"
)

// Db is global db var
var Db *sql.DB
var err error

func initDb() {
	Db, err = sql.Open("mysql", "root:passwd1234@(127.0.0.1:3306)/Db?parseTime=true")
	handleError(err, "")

	err := Db.Ping()
	handleError(err, "")

	query := `SELECT 1 FROM TodoPageData LIMIT 1;`
	if _, err := Db.Exec(query); err != nil {
		// Create a new table
		query = `
			CREATE TABLE TodoPageData (
				id INT AUTO_INCREMENT,
				title TEXT NOT NULL,
				PRIMARY KEY (id)
			);`
		if _, err := Db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
	query = `SELECT 1 FROM Todos LIMIT 1;`
	if _, err := Db.Exec(query); err != nil {
		// Create a new table
		query = `
			CREATE TABLE Todos (
				id INT AUTO_INCREMENT,
				todo TEXT NOT NULL,
				title TEXT NOT NULL,
				done BOOLEAN NOT NULL DEFAULT 0,
				PRIMARY KEY (id)
			);`
		if _, err := Db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
}
