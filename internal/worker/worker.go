package worker

import (
	"context"
	"fmt"
	"log"
	"time"

	fb "github.com/Guizzs26/firesync/internal/cdc/firebird"
	"github.com/Guizzs26/firesync/internal/writer"
)

type Worker struct {
	reader       *fb.OutboxReader
	writer       *writer.Writer
	pollInterval time.Duration
	maxBackoff   time.Duration
}

func NewWorker(r *fb.OutboxReader, w *writer.Writer, interval time.Duration) *Worker {
	return &Worker{
		reader:       r,
		writer:       w,
		pollInterval: interval,
		maxBackoff:   30 * time.Second,
	}
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	currentBackoff := 1 * time.Second
	fmt.Printf("Initilizing worker (interval: %v)\n", w.pollInterval)
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Ending worker gracefully...")
			return

		case <-ticker.C:
			err := w.processBatch()
			if err != nil {
				log.Printf("Synchronization cycle failure: %v", err)

				fmt.Printf("Waiting %v before trying again...\n", currentBackoff)

				select {
				case <-time.After(currentBackoff):
				case <-ctx.Done():
					fmt.Println("Ending worker gracefully...")
					return
				}
				currentBackoff = min(currentBackoff*2, w.maxBackoff)
				continue
			}
			currentBackoff = 1 * time.Second
		}
	}
}

func (w *Worker) processBatch() error {
	events, err := w.reader.Fetch()
	if err != nil {
		return fmt.Errorf("fetch: %v", err)
	}

	if len(events) == 0 {
		return nil
	}

	for _, ev := range events {
		if err := w.writer.Write(ev, ev.Metadata.Table); err != nil {
			return fmt.Errorf("write error on table %s: %w", ev.Metadata.Table, err)
		}

		if err := w.reader.MarkProcessed(ev.Metadata.OutboxID); err != nil {
			return fmt.Errorf("ack error for outbox %d: %w", ev.Metadata.OutboxID, err)
		}
		fmt.Printf("Event [%s] %d synchronized successfully\n", ev.Metadata.Table, ev.Metadata.SequenceID)
	}

	return nil
}
