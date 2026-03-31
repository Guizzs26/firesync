package event

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

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
	SequenceID      uint64    `json:"sequence_id"`
	TimestampOrigin int64     `json:"timestamp_origin"` // unix nano
	SchemaVersion   int       `json:"schema_version"`
}

type Payload struct {
	Before json.RawMessage `json:"before,omitempty"`
	After  json.RawMessage `json:"after,omitempty"`
}

type Control struct {
	RetryCount  int  `json:"retry_count"`
	IsSyncEvent bool `json:"is_sync_event"`
}

func New(sourceNodeID string) Event {
	id, _ := uuid.NewV7()
	return Event{
		Metadata: Metadata{
			EventID:         id,
			SourceNodeID:    sourceNodeID,
			SchemaVersion:   1,
			TimestampOrigin: time.Now().UnixNano(),
		},
		Control: Control{
			IsSyncEvent: true,
			RetryCount:  0,
		},
	}
}
