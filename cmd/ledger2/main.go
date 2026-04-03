package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/stockyard-dev/stockyard-ledger2/internal/server"
	"github.com/stockyard-dev/stockyard-ledger2/internal/store"
)

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8555"
	}
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = "./ledger2-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("ledger2: open database: %v", err)
	}
	defer db.Close()

	srv := server.New(db, server.DefaultLimits())

	fmt.Printf("\n  Ledger II — Self-hosted personal finance tracker\n")
	fmt.Printf("  ─────────────────────────────────\n")
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  ─────────────────────────────────\n\n")

	log.Printf("ledger2: listening on :%s", port)
	if err := http.ListenAndServe(":"+port, srv); err != nil {
		log.Fatalf("ledger2: %v", err)
	}
}
