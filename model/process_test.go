package model

import (
	"io"
	"regexp"
	"strings"
	"testing"
)

func TestStreamPipeToChan(t *testing.T) {
	ch := make(chan logEntry, 2)
	statusCh := make(chan processStatus, 1)
	r := io.NopCloser(strings.NewReader("foo\nbar\n"))

	streamPipeToChan(r, ch, nil, statusCh, logInfo)
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

func TestStreamPipeToChanSignalsReadyOnce(t *testing.T) {
	ch := make(chan logEntry, 4)
	statusCh := make(chan processStatus, 10)
	r := io.NopCloser(strings.NewReader("loaded\nloaded\nloaded\n"))

	streamPipeToChan(r, ch, regexp.MustCompile("^loaded$"), statusCh, logInfo)
	close(statusCh)

	var statuses []processStatus
	for s := range statusCh {
		statuses = append(statuses, s)
	}

	if len(statuses) != 1 {
		t.Fatalf("expected 1 status, got %d: %#v", len(statuses), statuses)
	}
	if statuses[0] != statusReady {
		t.Fatalf("expected statusReady, got %v", statuses[0])
	}
}

func TestStreamPipeToChanStripsOscSequences(t *testing.T) {
	ch := make(chan logEntry, 4)
	statusCh := make(chan processStatus, 1)
	r := io.NopCloser(strings.NewReader(
		"\x1b]0;npm run dev\afoo\n" +
			"bar\x1b]2;title\x1b\\baz\n" +
			"\x1b[31mred\x1b[0m\x1b]0;unterminated\n"))

	streamPipeToChan(r, ch, nil, statusCh, logInfo)
	close(ch)

	var entries []logEntry
	for e := range ch {
		entries = append(entries, e)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d: %#v", len(entries), entries)
	}
	if entries[0].msg != "foo" {
		t.Fatalf("expected BEL-terminated OSC to be stripped, got %q", entries[0].msg)
	}
	if entries[1].msg != "barbaz" {
		t.Fatalf("expected ST-terminated OSC to be stripped, got %q", entries[1].msg)
	}
	if entries[2].msg != "\x1b[31mred\x1b[0m" {
		t.Fatalf("expected color codes kept and unterminated OSC stripped, got %q", entries[2].msg)
	}
}

func TestStreamPipeToChanDropsWhenFull(t *testing.T) {
	ch := make(chan logEntry, 1)
	statusCh := make(chan processStatus, 1)
	r := io.NopCloser(strings.NewReader("foo\nbar\n"))

	streamPipeToChan(r, ch, nil, statusCh, logInfo)
	close(ch)

	var entries []logEntry
	for e := range ch {
		entries = append(entries, e)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].msg != "foo" {
		t.Fatalf("unexpected log entry: %#v", entries)
	}
}
