package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	// The official time zone database inside the binary (2026-11-04), so the
	// company time zones work on any server or container, with or without
	// tzdata installed.
	_ "time/tzdata"

	"github.com/joho/godotenv"
	"github.com/milbertk/shao/router"
	emailnotifier "github.com/milbertk/shao/shared"
)

func main() {
	// .env is optional (2026-10-30): locally it gives JWT_SECRET, the AlephK
	// default SMTP (SMTP_*), etc.; on a server the real environment wins
	// (godotenv never overwrites variables that are already set).
	if err := godotenv.Load(); err == nil {
		log.Println("Loaded settings from .env")
	}

	// Logs with levels; JSON in production (LOG_FORMAT, 2026-11-01). The old
	// log.Println lines go through the same logger.
	emailnotifier.SetupLogger()

	r := router.NewRouter()

	// PORT (default 8081) and CORS_ALLOWED_ORIGINS come from the environment
	// (2026-10-24), so a production deploy needs no code change.
	port := ":8081"
	if value := strings.TrimSpace(os.Getenv("PORT")); value != "" {
		port = ":" + strings.TrimPrefix(value, ":")
	}

	log.Println("ServiceHub backend running on http://localhost" + port)

	err := http.ListenAndServe(port, r)
	if err != nil {
		log.Fatal("Server error:", err)
	}
}
