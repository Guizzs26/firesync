package main

import (
	"encoding/json"

	"github.com/gofrs/uuid"
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

func main() {

}
