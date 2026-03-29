package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/gofrs/uuid"
	_ "github.com/jackc/pgx/v5/stdlib"  // Driver Postgres
	_ "github.com/nakagami/firebirdsql" // Driver Firebird
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
	fmt.Println("🚀 Iniciando teste de conectividade nos bancos...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pgDSN := "postgres://test-user:test-pass@localhost:7856/test-db?sslmode=disable"
	testDB(ctx, "pgx", pgDSN, "Postgres 16")

	fbDSN := "SYSDBA:masterkey@localhost:7854//firebird/data/pax.fdb"
	testDB(ctx, "firebirdsql", fbDSN, "Firebird 2.5")

}

func testDB(ctx context.Context, driver, dsn, name string) {
	db, err := sql.Open(driver, dsn)
	if err != nil {
		log.Printf("❌ [%s] Erro no driver: %v", name, err)
		return
	}
	defer db.Close()

	if err := db.PingContext(ctx); err != nil {
		log.Printf("❌ [%s] Falha na conexão: %v", name, err)
		return
	}

	events, _ := FetchEvents(db)
	for _, ev := range events {
		pretty, _ := json.MarshalIndent(ev, "", "  ")
		fmt.Printf("📦 Evento Capturado:\n%s\n", string(pretty))
	}

	fmt.Printf("✅ [%s] Conectado com sucesso!\n", name)
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
		var outboxID int64
		var before, after []byte

		if err := rows.Scan(
			&outboxID,
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
		e.Metadata.TimestampOrigin = time.Now().UnixNano()

		events = append(events, e)
	}

	return events, nil
}
