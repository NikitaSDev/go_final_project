package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	_ "modernc.org/sqlite"
)

const (
	dbName      = "scheduler.db"
	defaultPort = ":7540"
)

type Task struct {
	ID      string `json:"id"`
	Date    string `json:"date"`
	Title   string `json:"title"`
	Comment string `json:"comment"`
	Repeat  string `json:"repeat"`
}

var (
	DB     TaskStore
	dbPath string
)

func (t *Task) setTime(done bool) error {
	now := time.Now()
	now = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	if t.Date == "" {
		t.Date = now.Format(dateLayout)
	} else {
		date, err := time.Parse(dateLayout, t.Date)
		if err != nil {
			return err
		}
		if done {
			now = date
		}
		if done || date.Before(now) {
			if t.Repeat == "" {
				t.Date = now.Format(dateLayout)
			} else {
				nextDate, err := NextDate(now, t.Date, t.Repeat)
				if err != nil {
					return err
				}
				t.Date = nextDate
			}
		}
	}
	return nil
}

func main() {
	if err := checkDB(); err != nil {
		log.Fatal(err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		fmt.Println(err)
		return
	}

	DB = NewTaskStore(db)

	port := os.Getenv("TODO_PORT")
	if port != "" {
		port = fmt.Sprintf(":%s", port)
	} else {
		port = defaultPort
	}

	webDir := "web"

	mux := http.NewServeMux()
	mux.Handle("/", http.FileServer(http.Dir(webDir)))
	mux.HandleFunc("/api/nextdate", handleNextDate)
	mux.HandleFunc("/api/task", handleTask)
	mux.HandleFunc("/api/task/done", handleTaskDone)
	mux.HandleFunc("/api/tasks", handleTasks)

	err = http.ListenAndServe(port, mux)
	if err != nil {
		fmt.Printf("starting server error %s", err.Error())
	}
}
