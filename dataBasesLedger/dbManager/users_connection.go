package dbmanager

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/milbertk/class"
	"github.com/milbertk/databasesmng"
)

// Second connection (2026-10-22): the usersAK database of usersAppAK, where
// public.companies lives (company plan and plan dates). Same server, other
// database. It reads the keys without the "shao" prefix from connection.json:
// host, port, user, pass, database. The password is never printed.
var (
	usersDB     *sql.DB
	usersDBOnce sync.Once
	usersDBErr  error
)

// GetUsersDB initializes or returns the connection to the usersAK database.
func GetUsersDB() (*sql.DB, error) {
	usersDBOnce.Do(func() {
		reader, err := class.NewJSONReader("./connection.json")
		if err != nil {
			usersDBErr = fmt.Errorf("failed to read JSON config: %w", err)
			return
		}

		host, ok1 := reader.GetValue("host")
		port, ok2 := reader.GetValue("port")
		user, ok3 := reader.GetValue("user")
		pass, ok4 := reader.GetValue("pass")
		dbname, ok5 := reader.GetValue("database")
		if !(ok1 && ok2 && ok3 && ok4 && ok5) {
			usersDBErr = fmt.Errorf("missing usersAK connection config (host, port, user, pass, database)")
			return
		}

		println("Users DB connection:", host, port, user, dbname)
		conn, err := databasesmng.NewPostgresConnector(host, port, user, pass, dbname)
		if err != nil {
			usersDBErr = fmt.Errorf("failed to connect to the usersAK database: %w", err)
			return
		}
		usersDB = conn.DB
	})
	return usersDB, usersDBErr
}
