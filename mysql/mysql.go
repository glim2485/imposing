package mysql

import (
	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

var Db *sql.DB

func Init() {
	db_conn, err := sql.Open("mysql", "goapi:goapipass@tcp(127.0.0.1:3306)/myapi")
	if err != nil {
		panic(err)
	}
	Db = db_conn
}

func UpdateNodeLatency(node string, latency int) {
	queryStr := "UPDATE nodes SET latency = ? WHERE name = ?"
	_, err := Db.Exec(queryStr, latency, node)
	if err != nil {
		panic(err)
	}
}
