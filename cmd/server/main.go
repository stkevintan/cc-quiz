package main

import (
	"log"

	"classical-chinese-quiz/internal/config"
	"classical-chinese-quiz/internal/db"
	"classical-chinese-quiz/internal/routes"
)

func main() {
	cfg := config.Load()

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		log.Fatalf("database setup failed: %v", err)
	}
	defer database.Close()

	migrationInfo, err := db.Migrate(database)
	if err != nil {
		log.Fatalf("database migration failed: %v", err)
	}
	log.Printf("database ready at %s (%d migrations applied)", cfg.DBPath, migrationInfo.AppliedCount)

	router := routes.NewRouter(cfg, database)
	log.Printf("starting server on %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("server stopped: %v", err)
	}
}
