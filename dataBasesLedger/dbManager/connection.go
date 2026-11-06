package dbmanager

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/milbertk/class"
	"github.com/milbertk/databasesmng"
)

var (
	db    *sql.DB
	once  sync.Once
	dbErr error
)

// GetPostgresDB initializes or returns the existing DB connection
func GetPostgresDB() (*sql.DB, error) {
	print("entra en ps")
	once.Do(func() {
		// Load config
		reader, err := class.NewJSONReader("./connection.json")
		if err != nil {
			dbErr = fmt.Errorf("❌ Failed to read JSON config: %w", err)
			return
		}

		// Extract values
		host, ok1 := reader.GetValue("shaohost")
		port, ok2 := reader.GetValue("shaoport")
		user, ok3 := reader.GetValue("shaouser")
		pass, ok4 := reader.GetValue("shaopass")
		dbname, ok5 := reader.GetValue("shaodatabase")

		if !(ok1 && ok2 && ok3 && ok4 && ok5) {
			dbErr = fmt.Errorf("❌ Missing DB connection config")
			return
		}

		// The password is never printed (2026-10-28): logs go to journalctl on the server.
		println("Conection validate", host, port, user, dbname)
		conn, err := databasesmng.NewPostgresConnector(host, port, user, pass, dbname)
		if err != nil {
			dbErr = fmt.Errorf("❌ Failed to connect to PostgreSQL: %w", err)
			return
		}

		db = conn.DB
	})
	return db, dbErr
}
