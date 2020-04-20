package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

// Todo ...
type Todo struct {
	Todo string
	Done bool
}

// TodoPageData ...
type TodoPageData struct {
	PageTitle string
	Todos     []Todo
}

type fn func(string) []byte

var db *sql.DB
var err error

func initDb() {
	db, err = sql.Open("mysql", "root:passwd1234@(127.0.0.1:3306)/db?parseTime=true")
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Ping(); err != nil {
		log.Fatal(err)
	}

	query := `SELECT 1 FROM TodoPageData LIMIT 1;`
	if _, err := db.Exec(query); err != nil {
		// Create a new table
		query = `
			CREATE TABLE TodoPageData (
				id INT AUTO_INCREMENT,
				title TEXT NOT NULL,
				PRIMARY KEY (id)
			);`
		if _, err := db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
	query = `SELECT 1 FROM Todos LIMIT 1;`
	if _, err := db.Exec(query); err != nil {
		// Create a new table
		query = `
			CREATE TABLE Todos (
				id INT AUTO_INCREMENT,
				todo TEXT NOT NULL,
				title TEXT NOT NULL,
				done BOOLEAN NOT NULL DEFAULT 0,
				PRIMARY KEY (id)
			);`
		if _, err := db.Exec(query); err != nil {
			log.Fatal(err)
		}
	}
}

func checkTitleInDb(titleStr string) bool {
	var (
		id    int
		title string
	)
	query := "SELECT id, title FROM TodoPageData WHERE title = ?"
	if err := db.QueryRow(query, titleStr).Scan(&id, &title); err != nil {
		return false
	}
	fmt.Println(id, title)
	return true
}

func insertInTodoPageData(title string) bool {
	result, err := db.Exec(`INSERT INTO TodoPageData (title) VALUES (?)`, title)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	id, err := result.LastInsertId()
	fmt.Println(id)

	return true
}

func insertInTodos(title string, todo string) bool {
	result, err := db.Exec(`INSERT INTO Todos (todo, title) VALUES (?, ?)`, todo, title)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	id, err := result.LastInsertId()
	fmt.Println(id)

	return true
}

func updateDoneInTodos(title string, todo string, done bool) bool {
	_, err := db.Exec(`UPDATE Todos SET done = ?  WHERE title = ? AND todo = ?`, done, title, todo)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	return true
}

func createTitleInDb(title string) bool {
	if checkTitleInDb(title) {
		return true
	}
	return insertInTodoPageData(title)
}

func isTodosEmpty() bool {
	rows, err := db.Query(`SELECT 1 FROM Todos LIMIT 1`)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	defer rows.Close()
	i := 0
	for rows.Next() {
		i++
		break
	}
	if i == 0 {
		return true
	}
	return false
}

func getTodos(title string) *([]Todo) {
	var todos []Todo
	if isTodosEmpty() {
		return &todos
	}
	rows, err := db.Query(`SELECT todo, done FROM Todos WHERE title='` + title + "'")
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	for rows.Next() {
		var t Todo

		err := rows.Scan(&t.Todo, &t.Done)
		if err != nil {
			log.Fatal(err)
		}
		todos = append(todos, t)
	}
	if err := rows.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%#v", todos)
	return &todos
}

func fillTodos(title string) *TodoPageData {
	createTitleInDb(title)

	data := TodoPageData{
		PageTitle: title,
		Todos: []Todo{
			{},
		},
	}
	data.Todos = *(getTodos(title))
	return &data
}

func addTaskToDb(title string, todo string) bool {
	return insertInTodos(title, todo)
}

func isDuplicateTask(titleStr string, task string) bool {
	var (
		id    int
		title string
		todo  string
	)
	query := "SELECT id, title, todo FROM TodoPageData WHERE title = ? AND todo = ?"
	if err := db.QueryRow(query, titleStr, task).Scan(&id, &title, &todo); err != nil {
		return false
	}
	fmt.Println(id, title, todo)
	return true
}

func addTask(msgStr string) []byte {
	var splits []string = strings.Split(msgStr, " ")
	var title = splits[0]
	var task = splits[1]
	//TODO: add to db
	b := []byte("Added!")
	if isDuplicateTask(title, task) {
		b = []byte("Duplicate todo found. Failed to add!")
	} else if !addTaskToDb(title, task) {
		b = []byte("Failed to add due to unknown error!")
	}
	return b
}

func updateTaskDone(msgStr string) []byte {
	var splits []string = strings.Split(msgStr, " ")
	var title = splits[0]
	var task = splits[1]
	var status, _ = strconv.ParseBool(splits[2])
	//add to db
	b := []byte("Marked Done!")
	if !updateDoneInTodos(title, task, status) {
		b = []byte("Couldn't mark as done due to unknown error!")
	}
	return b
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func writeBackMsgToClient(conn *websocket.Conn, oper fn, msgType int, msgStr string) {
	b := oper(string(msgStr))
	// Write message back to browser
	if err = conn.WriteMessage(msgType, b); err != nil {
		return
	}
}

func handleWebSocket(w http.ResponseWriter, r *http.Request, oper fn) {
	msgType := 1
	conn, err := upgrader.Upgrade(w, r, nil) // get the upgrader connection
	if err != nil {
		var msgStr string
		msgStr = "Warning: Could get the websocket connection! Cannot handle websocket traffic for " + r.URL.Path
		fmt.Println(msgStr)
		writeBackMsgToClient(conn, oper, msgType, msgStr)
		return
	}

	for {
		// Read message from browser
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			errMsg := "Failed reading message due to :" + err.Error()
			writeBackMsgToClient(conn, oper, msgType, errMsg)
			return
		}

		// Print the message to the console
		msgStr := string(msg)
		fmt.Printf("%s sent: %s\n", conn.RemoteAddr(), msgStr)

		writeBackMsgToClient(conn, oper, msgType, msgStr)

	}
}

func main() {
	initDb()
	// Use mux router
	r := mux.NewRouter()
	tmpl := template.Must(template.ParseFiles("layout.html"))

	r.HandleFunc("/{title}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		title := vars["title"]
		data := *(fillTodos(title))
		tmpl.Execute(w, data)
	})

	r.HandleFunc("/todo/addTask", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, addTask)
	})

	r.HandleFunc("/todo/updateTaskDone", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, updateTaskDone)
	})

	r.HandleFunc("/todo/updateTaskNotDone", func(w http.ResponseWriter, r *http.Request) {

	})

	r.HandleFunc("/todo/deleteTask", func(w http.ResponseWriter, r *http.Request) {

	})

	http.ListenAndServe(":80", r)
}
