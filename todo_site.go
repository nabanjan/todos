package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
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

type todoState string

const (
	todoDone    todoState = "Done"
	todoNotDone           = "NotDone"
	todoDelete            = "Delete"
)

type fn func(string) []byte

func checkTitleInDb(titleStr string) bool {
	var (
		id    int
		title string
	)
	query := "SELECT id, title FROM TodoPageData WHERE title = ?"
	if err := Db.QueryRow(query, titleStr).Scan(&id, &title); err != nil {
		return false
	}
	fmt.Println(id, title)
	return true
}

func insertInTodoPageData(title string) bool {
	result, err := Db.Exec(`INSERT INTO TodoPageData (title) VALUES (?)`, title)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	id, err := result.LastInsertId()
	fmt.Println(id)

	return true
}

func insertInTodos(title string, todo string) bool {
	result, err := Db.Exec(`INSERT INTO Todos (todo, title) VALUES (?, ?)`, todo, title)
	if err != nil {
		fmt.Println(err.Error())
		return false
	}
	id, err := result.LastInsertId()
	fmt.Println(id)

	return true
}

func updateDoneInTodos(title string, todo string, state todoState) bool {
	var done bool
	if state == todoDone || state == todoNotDone {
		if state == todoDone {
			done = true
		}
		if state == todoNotDone {
			done = false
		}
		_, err := Db.Exec(`UPDATE Todos SET done = ?  WHERE title = ? AND todo = ?`, done, title, todo)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
		return true
	}
	if state == todoDelete {
		_, err := Db.Exec(`DELETE FROM Todos WHERE title = ? AND todo = ?`, title, todo)
		if err != nil {
			fmt.Println(err.Error())
			return false
		}
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
	rows, err := Db.Query(`SELECT 1 FROM Todos LIMIT 1`)
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
	rows, err := Db.Query(`SELECT todo, done FROM Todos WHERE title='` + title + "'")
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
	query := "SELECT id, title, todo FROM Todos WHERE title = ? AND todo = ?"
	err := Db.QueryRow(query, titleStr, task).Scan(&id, &title, &todo)
	if err == sql.ErrNoRows {
		return false
	}
	fmt.Println(id, title, todo)
	return true
}

func addTask(msgStr string) []byte {
	var splits []string = strings.Split(msgStr, " ")
	var title = splits[0]
	var task = splits[1]
	//TODO: add to Db
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
	var state = getTodoState(splits[2])
	//delete from Db
	b := []byte("Marked " + splits[2])
	if !updateDoneInTodos(title, task, state) {
		b = []byte("Couldn't mark as " + splits[2] + " due to unknown error!")
	}
	return b
}

func getTodoState(state string) todoState {
	switch state {
	case "done":
		return todoDone
	case "notDone":
		return todoNotDone
	case "delete":
		return todoDelete
	}
	return todoDone
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
	defer conn.Close()

	if err != nil {
		var msgStr string
		msgStr = "Warning: Couldn't get the websocket connection! Cannot handle websocket traffic for " + r.URL.Path
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
	//initRMQ()
	// Use mux router
	r := mux.NewRouter()
	tmpl := template.Must(template.ParseFiles("layout.html"))

	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := *(fillTodos("home"))
		tmpl.Execute(w, data)
	})

	r.HandleFunc("/{title}", func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		title := vars["title"]
		data := *(fillTodos(title))
		err = tmpl.Execute(w, data)
		handleError(err, "Couldn't read the layout.html")
	})

	r.HandleFunc("/todo/addTask", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, addTask)
	})

	r.HandleFunc("/todo/updateTaskDone", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, updateTaskDone)
	})

	r.HandleFunc("/todo/updateTaskNotDone", func(w http.ResponseWriter, r *http.Request) {

	})

	r.HandleFunc("/todo/deleteTaskDone", func(w http.ResponseWriter, r *http.Request) {
		handleWebSocket(w, r, updateTaskDone)
	})

	fmt.Println("Starting server...")
	http.ListenAndServe(":80", r)
}
