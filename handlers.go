package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const maxFilterTasks = 20

func handleTask(w http.ResponseWriter, req *http.Request) {

	switch req.Method {
	case http.MethodPost:
		writeTask(w, req, true)
	case http.MethodGet:
		getTask(w, req)
	case http.MethodPut:
		writeTask(w, req, false)
	case http.MethodDelete:
		deleteTask(w, req)
	}
}

func handleTaskDone(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	if id == "" {
		setError(w, "Не указан идентификатор", http.StatusBadRequest)
		return
	}

	task, err := getTaskByID(id)
	if err != nil {
		status := http.StatusBadRequest
		if err == sql.ErrNoRows {
			status = http.StatusNotFound
		}
		setError(w, err.Error(), status)
		return
	}

	if task.Repeat == "" {
		deleteTaskDB(w, id)
		return
	}

	if err := task.setTime(true); err != nil {
		setError(w, err.Error(), http.StatusBadRequest)
		return
	}
	writeTaskDB(w, task, false)
}

func handleTasks(w http.ResponseWriter, req *http.Request) {
	query := `SELECT
				id as id,
				date as date,
				title as title,
				comment as comment,
				repeat as repeat
			FROM
				scheduler
				%s
			ORDER BY
				date
			LIMIT $maxTasks`

	filter := ""
	searchWord := req.URL.Query().Get("search")
	if searchWord != "" {
		if matched, _ := regexp.MatchString(`\d\d\.\d\d\.\d\d\d\d`, searchWord); matched {
			searchWord = searchWord[6:] + searchWord[3:5] + searchWord[0:2] //02.01.2006
			filter = `WHERE date LIKE $par`
		} else {
			searchWord = "%" + searchWord + "%"
			filter = `WHERE title LIKE $par
			OR comment LIKE $par`
		}
	}

	query = fmt.Sprintf(query, filter)

	rows, err := DB.db.Query(query, sql.Named("maxTasks", maxFilterTasks), sql.Named("par", searchWord))

	if err != nil {
		setError(w, err.Error(), http.StatusBadRequest)
		return
	}
	defer rows.Close()

	tasks := []Task{}
	for rows.Next() {
		t := Task{}
		err := rows.Scan(&t.ID, &t.Date, &t.Title, &t.Comment, &t.Repeat)
		if err != nil {
			setError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		tasks = append(tasks, t)
	}

	mapTasks := map[string][]Task{
		"tasks": tasks,
	}
	res, err := json.Marshal(mapTasks)
	if err != nil {
		setError(w, err.Error(), http.StatusInternalServerError)
	}

	w.Write(res)
}

func handleNextDate(w http.ResponseWriter, req *http.Request) {

	now, err := time.Parse(dateLayout, req.URL.Query().Get("now"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	date := req.URL.Query().Get("date")
	repeat := req.URL.Query().Get("repeat")

	nextDate, err := NextDate(now, date, repeat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprint(w, nextDate)
}

func writeTask(w http.ResponseWriter, req *http.Request, adding bool) {
	w.Header().Set("Content-Type", "application/json; charset=UTF-8")

	var task Task
	var buf bytes.Buffer

	_, err := buf.ReadFrom(req.Body)
	if err != nil {
		setError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := json.Unmarshal(buf.Bytes(), &task); err != nil {
		setError(w, err.Error(), http.StatusBadRequest)
		return
	}

	if task.Title == "" {
		setError(w, "не указан заголовок задачи", http.StatusBadRequest)
		return
	}

	if !adding && task.ID == "" {
		setError(w, "Не указан идентификатор", http.StatusBadRequest)
		return
	}
	if err := task.setTime(false); err != nil {
		setError(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeTaskDB(w, task, adding)
}

func deleteTask(w http.ResponseWriter, req *http.Request) {
	id := req.URL.Query().Get("id")
	if id == "" {
		setError(w, "Не указан идентификатор", http.StatusBadRequest)
		return
	}
	_, err := getTaskByID(id)
	if err != nil {
		status := http.StatusBadRequest
		if err == sql.ErrNoRows {
			status = http.StatusNotFound
		}
		setError(w, err.Error(), status)
		return
	}
	deleteTaskDB(w, id)
}

func getTask(w http.ResponseWriter, req *http.Request) {

	id := req.URL.Query().Get("id")
	if id == "" {
		setError(w, "Не указан идентификатор", http.StatusBadRequest)
		return
	}

	task, err := getTaskByID(id)
	if err != nil {
		status := http.StatusBadRequest
		if err == sql.ErrNoRows {
			status = http.StatusNotFound
		}
		setError(w, err.Error(), status)
		return
	}

	res, err := json.Marshal(task)
	if err != nil {
		setError(w, err.Error(), http.StatusBadGateway)
	}
	w.Write(res)
}

func setError(w http.ResponseWriter, error string, status int) {
	err := map[string]string{
		"error": error,
	}
	errJSON, _ := json.Marshal(err)
	w.WriteHeader(status)
	w.Write(errJSON)
}

func writeTaskDB(w http.ResponseWriter, task Task, adding bool) {
	if !adding {
		_, err := getTaskByID(task.ID)
		if err != nil {
			setError(w, "Задача не найдена", http.StatusBadRequest)
			return
		}
	}

	var query string
	if adding {
		query = "INSERT INTO scheduler (date, title, comment, repeat) values ($date, $title, $comment, $repeat)"
	} else {
		query = `UPDATE
		scheduler
	SET
		id = $id,
		date = $date,
		title = $title,
		comment = $comment,
		repeat = $repeat
	WHERE
		id = $id`
	}

	res, err := DB.db.Exec(query,
		sql.Named("date", task.Date),
		sql.Named("title", task.Title),
		sql.Named("comment", task.Comment),
		sql.Named("repeat", task.Repeat),
		sql.Named("id", task.ID),
	)
	if err != nil {
		setError(w, err.Error(), http.StatusBadRequest)
		return
	}

	var response string
	if adding {
		id, err := res.LastInsertId()
		if err != nil {
			setError(w, err.Error(), http.StatusInternalServerError)
			return
		}
		response = fmt.Sprintf(`{"id":"%d"}`, id)
	} else {
		response = `{}`
	}
	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, response)
}

func deleteTaskDB(w http.ResponseWriter, id string) {
	query := `DELETE FROM scheduler WHERE id = $id`
	_, err := DB.db.Exec(query, sql.Named("id", id))
	if err != nil {
		setError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprint(w, `{}`)
}

func getTaskByID(id string) (Task, error) {
	query := `SELECT
				id as id,
				date as date,
				title as title,
				comment as comment,
				repeat as repeat
			FROM
				scheduler
			WHERE
				id = $id`

	var task Task
	row := DB.db.QueryRow(query, sql.Named("id", id))
	err := row.Scan(&task.ID, &task.Date, &task.Title, &task.Comment, &task.Repeat)
	return task, err
}
