package model

import (
	"io"
	"strings"
	"testing"
)

func TestStreamPipeToChan(t *testing.T) {
	ch := make(chan logEntry, 2)
	r := io.NopCloser(strings.NewReader("foo\nbar\n"))

	streamPipeToChan(r, ch, logInfo)
	close(ch)

	var entries []logEntry
	for e := range ch {
		entries = append(entries, e)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].msg != "foo" || entries[1].msg != "bar" {
		t.Fatalf("unexpected log entries: %#v", entries)
	}
	if entries[0].level != logInfo || entries[1].level != logInfo {
		t.Fatalf("unexpected log level: %#v", entries)
	}
}
