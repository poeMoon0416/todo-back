package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
)

// Todoテーブルのモデル
type Todo struct {
	Id     int64  `json:"id"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Point  int64  `json:"point"`
	Done   bool   `json:"done"`
}

// DBへの接続、どの関数からでもアクセスできるようにグローバル変数
var db *sql.DB

func main() {
	// DSNの定義
	cfg := mysql.Config{
		User:   os.Getenv("DB_USER"),
		Passwd: os.Getenv("DB_PASS"),
		Net:    "tcp",
		Addr:   fmt.Sprintf("%v:%v", os.Getenv("DB_HOST"), os.Getenv("DB_PORT")),
		DBName: os.Getenv("DB_NAME"),
	}

	// DB接続(接続できなくなった場合再接続を試み続ける)
	var err error
	db, err = sql.Open("mysql", cfg.FormatDSN())
	if err != nil {
		log.Fatalf("fail to connect MySQL server: %v", err)
	}

	// 最初のDB接続の失敗時にエラーを出す用(sql.Open()だけだと接続時エラーでない)
	if err := db.Ping(); err != nil {
		log.Fatalf("fail to ping MySQL server: %v", err)
	}

	// エントリーポイントでルータを実行
	router := gin.Default()
	// 以下テストコマンド
	/*
		curl --request 'POST' \
		--url "http://${AP_HOST}:${AP_PORT}/todos" \
		--header 'Content-Type: application/json' \
		--data '{"title": "アプリの完成", "detail": "Denoを頑張って学ぶ必要がある。", "point": 1, "done": true}' \
		--include
	*/
	router.POST("/todos", createTodo)
	/*
		curl --request 'GET' \
		--url "http://${AP_HOST}:${AP_PORT}/todos" \
		--header 'Content-Type: application/json' \
		--include
	*/
	router.GET("/todos", listTodos)
	/*
		curl --request 'GET' \
		--url "http://${AP_HOST}:${AP_PORT}/todos/2" \
		--header 'Content-Type: application/json' \
		--include
	*/
	router.GET("/todos/:id", getTodo)
	/*
		curl --request 'PUT' \
		--url "http://${AP_HOST}:${AP_PORT}/todos/2" \
		--header 'Content-Type: application/json' \
		--data '{"title": "アプリの完成", "detail": "Node.jsとGoとMySQLを頑張って学ぶ必要がある。", "point": 3, "done": true}' \
		--include
	*/
	router.PUT("/todos/:id", updateTodo)
	/*
		curl --request 'DELETE' \
		--url "http://${AP_HOST}:${AP_PORT}/todos/2" \
		--header 'Content-Type: application/json' \
		--include
	*/
	router.DELETE("/todos/:id", deleteTodo)
	// APサーバのipアドレス(自身以外が可能) or localhost(127.0.0.1)
	router.Run(fmt.Sprintf("%v:%v", os.Getenv("AP_HOST"), os.Getenv("AP_PORT")))
}

// Todoを作成
func createTodo(ctx *gin.Context) {
	// bodyのチェック(JSON形式で型があっているかチェックしている, 余計なフィールド足りないフィールドは無視される)
	var newTodo Todo
	if err := ctx.BindJSON(&newTodo); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "body must be todo's json"})
		return
	}

	// DBへの挿入
	res, err := db.Exec("INSERT INTO todos(title, detail, point, done) VALUES(?, ?, ?, ?)", newTodo.Title, newTodo.Detail, newTodo.Point, newTodo.Done)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to create todo"})
		return
	}

	// int64でAUTO_INCREMENTのIDを取得
	newTodo.Id, err = res.LastInsertId()
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to get last insert id"})
		return
	}

	// 正常系
	ctx.IndentedJSON(http.StatusCreated, newTodo)
}

// Todoを一覧表示
func listTodos(ctx *gin.Context) {
	// クエリ実行
	rows, err := db.Query("SELECT * FROM todos")
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to exec query"})
		return
	}
	defer rows.Close()

	// 1行ずつ読み出し
	todos := make([]Todo, 0)
	for rows.Next() {
		var todo Todo
		if err := rows.Scan(&todo.Id, &todo.Title, &todo.Detail, &todo.Point, &todo.Done); err != nil {
			ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to scan columns to struct"})
			return
		}
		todos = append(todos, todo)
	}

	// rows.Next()がエラーで抜けてきた場合
	if rows.Err() != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to read rows"})
		return
	}

	// 正常系
	ctx.IndentedJSON(http.StatusOK, todos)
}

// Todoをid指定で単一取得
func getTodo(ctx *gin.Context) {
	// slugのチェック
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "id must can parse string to int64"})
		return
	}

	// クエリ実行
	var todo Todo
	row := db.QueryRow("SELECT * FROM todos WHERE id = ?", id)
	if err := row.Scan(&todo.Id, &todo.Title, &todo.Detail, &todo.Point, &todo.Done); err != nil {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "not exists id"})
		return
	}

	// 正常系
	ctx.IndentedJSON(http.StatusOK, todo)
}

// Todoをid指定で単一更新(PUTなので指定がないフィールドは初期化される)
func updateTodo(ctx *gin.Context) {
	// slugのチェック
	var newTodo Todo
	var err error
	newTodo.Id, err = strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "id must can parse string to int64"})
		return
	}

	// bodyのチェック(JSON形式で型があっているかチェックしている, 余計なフィールド足りないフィールドは無視される)
	if err := ctx.BindJSON(&newTodo); err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "body must be todo's json"})
		return
	}

	// 更新クエリ実行
	var res sql.Result
	res, err = db.Exec("UPDATE todos SET title = ?, detail = ?, point = ?, done = ? WHERE id = ?", newTodo.Title, newTodo.Detail, newTodo.Point, newTodo.Done, newTodo.Id)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to exec query"})
		return
	}

	// 更新行数の取得
	var cnt int64
	cnt, err = res.RowsAffected()
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to get affected row count"})
		return
	}

	// 1行も消していない場合
	if cnt == 0 {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "not exists id"})
		return
	}

	// 正常系
	ctx.IndentedJSON(http.StatusOK, newTodo)
}

// Todoをid指定で単一削除
func deleteTodo(ctx *gin.Context) {
	// slugのチェック
	id, err := strconv.ParseInt(ctx.Param("id"), 10, 64)
	if err != nil {
		ctx.IndentedJSON(http.StatusBadRequest, gin.H{"message": "id must can parse string to int64"})
		return
	}

	// 削除クエリ実行
	var res sql.Result
	res, err = db.Exec("DELETE FROM todos WHERE id = ?", id)
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to exec query"})
		return
	}

	// 削除行数の取得
	var cnt int64
	cnt, err = res.RowsAffected()
	if err != nil {
		ctx.IndentedJSON(http.StatusInternalServerError, gin.H{"message": "fail to get affected row count"})
		return
	}

	// 1行も消していない場合
	if cnt == 0 {
		ctx.IndentedJSON(http.StatusNotFound, gin.H{"message": "not exists id"})
		return
	}

	// 正常系
	ctx.IndentedJSON(http.StatusOK, gin.H{"id": id})
}
