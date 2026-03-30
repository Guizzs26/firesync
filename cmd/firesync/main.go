package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "github.com/nakagami/firebirdsql"
)

/*

Event structure exemple:

{
  "metadata": {
    "event_id": "018e7b2a-8c1d-7f3e-a1b2-c3d4e5f6a7b8",
    "source_node_id": "fb-instance-01",
    "sequence_id": 100234,
    "operation": "UPDATE",
    "table": "orders",
    "timestamp_origin": 1711554000000000000,
    "checksum": "sha256-e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "schema_version": 1
  },
  "payload": {
    "before": { "id": 50, "status": "OPEN", "total": 100.50 },
    "after":  { "id": 50, "status": "PAID", "total": 100.50 }
  },
  "control": {
    "is_sync_event": true,
    "retry_count": 0
  }
}

*/

type Operation string

const (
	OpInsert Operation = "INSERT"
	OpUpdate Operation = "UPDATE"
	OpDelete Operation = "DELETE"
)

type Event struct {
	Metadata Metadata `json:"metadata"`
	Payload  Payload  `json:"payload"`
	Control  Control  `json:"control"`
}

type Metadata struct {
	EventID         uuid.UUID `json:"event_id"`
	OutboxID        int64     `json:"-"`
	Operation       Operation `json:"operation"`
	Table           string    `json:"table"`
	SourceNodeID    string    `json:"source_node_id"`
	Checksum        string    `json:"checksum"`
	SequenceID      uint64    `json:"sequence_id"`      // Chronological ordering by table/node
	TimestampOrigin int64     `json:"timestamp_origin"` // Unix Nano
	SchemaVersion   int       `json:"schema_version"`
}

type Payload struct {
	Before json.RawMessage `json:"before"`
	After  json.RawMessage `json:"after"`
}

type Control struct {
	RetryCount  int  `json:"retry_count"`
	IsSyncEvent bool `json:"is_sync_event"`
}

type Config struct {
	Source      DBConfig `yaml:"source"`
	Target      DBConfig `yaml:"target"`
	SourceTable string   `yaml:"source_table"`
	TargetTable string   `yaml:"target_table"`
}

type DBConfig struct {
	Driver   string `yaml:"driver"`
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	Database string `yaml:"database"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fbDSN := "SYSDBA:masterkey@localhost:7854//firebird/data/pax.fdb"
	db, err := Connect(ctx, "firebird", fbDSN)
	if err != nil {
		os.Exit(1)
	}
	defer db.Close()

	go StartWorker(db)
}

func Connect(ctx context.Context, driver, dsn string) (*sql.DB, error) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Printf("open: %v", err)
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		log.Printf("ping: %v", err)
		return nil, err
	}

	fmt.Println("sucessfully connected!")

	return db, nil
}

func FetchEvents(db *sql.DB) ([]Event, error) {
	const query = `
    SELECT ID, TABLE_NAME, OPERATION, PAYLOAD_BEFORE, PAYLOAD_AFTER, SEQUENCE_ID
    FROM SYNC_OUTBOX
    WHERE PROCESSED = 0
    ORDER BY SEQUENCE_ID ASC
  `

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var events []Event
	for rows.Next() {
		var e Event
		var before, after []byte

		if err := rows.Scan(
			&e.Metadata.OutboxID,
			&e.Metadata.Table,
			&e.Metadata.Operation,
			&before,
			&after,
			&e.Metadata.SequenceID,
		); err != nil {
			return nil, err
		}

		e.Metadata.EventID, _ = uuid.NewV7()
		e.Payload.Before = before
		e.Payload.After = after
		e.Metadata.SchemaVersion = 1
		// e.Metadata.TimestampOrigin = time.Now().UnixNano()

		events = append(events, e)
	}

	return events, nil
}

func MarkAsProcessed(db *sql.DB, outboxID int64) error {
	const query = `
		UPDATE SYNC_OUTBOX SET PROCESSED = 1 WHERE ID = ?
	`

	_, err := db.Exec(query, outboxID)
	if err != nil {
		return err
	}

	return nil
}

func StartWorker(db *sql.DB) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	fmt.Println("initialized worker")

	for range ticker.C {
		events, err := FetchEvents(db)
		if err != nil {
			log.Printf("fetch events: %v", err)
			continue
		}

		if len(events) == 0 {
			continue
		}

		for _, ev := range events {
			fmt.Printf("syncronized: [%s] ID: %d\n", ev.Metadata.Table, ev.Metadata.SequenceID)

			err := MarkAsProcessed(db, ev.Metadata.OutboxID)
			if err != nil {
				log.Printf("mark as processed %d: %v", ev.Metadata.OutboxID, err)
				continue
			}

			fmt.Printf("event %d processed and confirmed!\n", ev.Metadata.OutboxID)
		}
	}
}
