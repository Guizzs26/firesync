package fb

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/Guizzs26/firesync/pkg/event"

	_ "github.com/nakagami/firebirdsql"
)

type OutboxReader struct {
	db           *sql.DB
	sourceNodeID string
}

func NewOutboxReader(db *sql.DB, sourceNodeID string) *OutboxReader {
	return &OutboxReader{
		db:           db,
		sourceNodeID: sourceNodeID,
	}
}

func (or *OutboxReader) Fetch() ([]event.Event, error) {
	const query = `
		SELECT
			ID,
			TABLE_NAME,
			OPERATION,
			PAYLOAD_BEFORE,
			PAYLOAD_AFTER,
			SEQUENCE_ID,
			CREATED_AT
		FROM SYNC_OUTBOX
		WHERE PROCESSED = 0
		ORDER BY SEQUENCE_ID ASC
		ROWS 100
	`

	rows, err := or.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("outbox query: %v", err)
	}

	var events []event.Event
	for rows.Next() {
		var (
			outboxID   int64
			table      string
			op         string
			before     []byte
			after      []byte
			sequenceID uint64
			createdAt  time.Time
		)

		if err := rows.Scan(
			&outboxID,
			&table,
			&op,
			&before,
			&after,
			&sequenceID,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("outbox scan: %v", err)
		}

		e := event.New(or.sourceNodeID)
		e.Metadata.OutboxID = outboxID
		e.Metadata.Operation = event.Operation(op)
		e.Metadata.Table = table
		e.Metadata.SequenceID = sequenceID
		e.Metadata.TimestampOrigin = createdAt.UnixNano()
		e.Payload.Before = before
		e.Payload.After = after

		events = append(events, e)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("outbox rows: %v", err)
	}

	return events, nil
}

func (or *OutboxReader) MarkProcessed(outboxID int64) error {
	const query = `
		UPDATE SYNC_OUTBOX SET PROCESSED = 1 WHERE ID = ?
	`

	_, err := or.db.Exec(query, outboxID)
	if err != nil {
		return fmt.Errorf("mark processed %d: %v", outboxID, err)
	}

	return nil
}
