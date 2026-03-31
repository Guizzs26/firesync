package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	fb "github.com/Guizzs26/firesync/internal/cdc/firebird"
	"github.com/Guizzs26/firesync/internal/config"
	"github.com/Guizzs26/firesync/internal/worker"
	"github.com/Guizzs26/firesync/internal/writer"

	_ "github.com/lib/pq"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatalf("error load config: %v", err)
	}

	fbAddr := fmt.Sprintf("%s:%s@%s:%d/%s",
		cfg.Source.User,
		cfg.Source.Password,
		cfg.Source.Host,
		cfg.Source.Port,
		cfg.Source.Database,
	)

	fbDB, err := sql.Open("firebirdsql", fbAddr)
	if err != nil {
		log.Fatalf("error open firebird: %v", err)
	}
	defer fbDB.Close()

	pgDB, err := sql.Open("postgres", cfg.Destination.URL)
	if err != nil {
		log.Fatalf("error open postgres: %v", err)
	}
	defer pgDB.Close()

	reader := fb.NewOutboxReader(fbDB, cfg.NodeID)
	writer := writer.NewWriter(pgDB)

	syncWorker := worker.NewWorker(reader, writer, cfg.PollInterval)

	fmt.Printf("Firesync [%s] started. Monitoring %v...\n", cfg.NodeID, cfg.PollInterval)
	syncWorker.Start(ctx)
}
