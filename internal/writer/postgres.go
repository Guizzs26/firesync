package writer

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/Guizzs26/firesync/pkg/event"
)

var validIdentifier = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

type Writer struct {
	db *sql.DB
}

func NewWriter(db *sql.DB) *Writer {
	return &Writer{db: db}
}

func (w *Writer) Write(e event.Event, targetTable string) error {
	tx, err := w.db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %v", err)
	}
	defer tx.Rollback()

	var exists bool
	err = tx.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM sync_processed_events WHERE event_id = $1)`,
		e.Metadata.EventID,
	).Scan(&exists)
	if err != nil {
		return fmt.Errorf("idempotency check: %v", err)
	}
	if exists {
		return nil
	}

	switch e.Metadata.Operation {
	case event.OpDelete:
		if err := w.handleDelete(tx, e, targetTable); err != nil {
			return err
		}

	default:
		if err := w.handleUpsert(tx, e, targetTable); err != nil {
			return err
		}
	}

	_, err = tx.Exec(
		`INSERT INTO sync_processed_events (event_id, source_node_id, table_name)
         VALUES ($1, $2, $3)`,
		e.Metadata.EventID,
		e.Metadata.SourceNodeID,
		e.Metadata.Table,
	)
	if err != nil {
		return fmt.Errorf("register processed event: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %v", err)
	}

	return nil
}

func (w *Writer) handleUpsert(tx *sql.Tx, e event.Event, targetTable string) error {
	table, err := sanitize(targetTable)
	if err != nil {
		return err
	}

	var data map[string]any
	if err := json.Unmarshal(e.Payload.After, &data); err != nil {
		return fmt.Errorf("unsmarshal after payload: %v", err)
	}

	if len(data) == 0 {
		return fmt.Errorf("empty payload for table %s", table)
	}

	cols := make([]string, 0, len(data))
	vals := make([]any, 0, len(data))
	for col, val := range data {
		clean, err := sanitize(col)
		if err != nil {
			return err
		}
		cols = append(cols, clean)
		vals = append(vals, val)
	}

	placeholders := make([]string, len(cols))
	setClauses := make([]string, len(cols))
	for i, col := range cols {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		setClauses[i] = fmt.Sprintf("%s = EXCLUDED.%s", col, col)
	}

	query := fmt.Sprintf(
		`INSERT INTO %s (%s) VALUES (%s) ON CONFLICT (id) DO UPDATE SET %s`,
		table,
		strings.Join(cols, ", "),
		strings.Join(placeholders, ", "),
		strings.Join(setClauses, ", "),
	)

	_, err = tx.Exec(query, vals...)
	if err != nil {
		return fmt.Errorf("execute dynamic upsert on %s: %v", targetTable, err)
	}

	return nil
}

func (w *Writer) handleDelete(tx *sql.Tx, e event.Event, targetTable string) error {
	table, err := sanitize(targetTable)
	if err != nil {
		return err
	}

	var data map[string]any
	src := e.Payload.Before
	if err := json.Unmarshal(src, &data); err != nil {
		return fmt.Errorf("unmarshal before payload: %v", err)
	}

	id, ok := data["id"]
	if !ok {
		return fmt.Errorf("missing 'id' in before payload for delete on %s", table)
	}

	query := fmt.Sprintf("DELETE FROM %s WHERE id = $1", table)

	_, err = tx.Exec(query, id)
	if err != nil {
		return fmt.Errorf("execute delete on %s: %v", targetTable, err)
	}

	return err
}

func sanitize(identifier string) (string, error) {
	if !validIdentifier.MatchString(identifier) {
		return "", fmt.Errorf("invalid identifier: %q", identifier)
	}
	return identifier, nil
}
