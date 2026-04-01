package ioutil

import (
	"bytes"
	"errors"
	"os"
	"testing"
)

func TestProgressWriter_CountsBytes(t *testing.T) {
	var buf bytes.Buffer
	pw := &ProgressWriter{Dest: &buf, Total: 1000}

	data := make([]byte, 500)
	n, err := pw.Write(data)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != 500 {
		t.Errorf("Write() returned %d, want 500", n)
	}

	n, err = pw.Write(data[:300])
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}
	if n != 300 {
		t.Errorf("Write() returned %d, want 300", n)
	}
}

func TestProgressWriter_DelegatesToDest(t *testing.T) {
	var buf bytes.Buffer
	pw := &ProgressWriter{Dest: &buf, Total: -1}

	data := []byte("hello driftr")
	if _, err := pw.Write(data); err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if got := buf.String(); got != "hello driftr" {
		t.Errorf("dest received %q, want %q", got, "hello driftr")
	}
}

type errWriter struct{}

func (errWriter) Write([]byte) (int, error) {
	return 0, errors.New("disk full")
}

func TestProgressWriter_PropagatesErrors(t *testing.T) {
	pw := &ProgressWriter{Dest: errWriter{}, Total: 100}

	_, err := pw.Write([]byte("data"))
	if err == nil {
		t.Fatal("expected error from Write(), got nil")
	}
	if err.Error() != "disk full" {
		t.Errorf("error = %q, want %q", err.Error(), "disk full")
	}
}

func TestIsTerminal_Pipe(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("os.Pipe() error: %v", err)
	}
	defer r.Close()
	defer w.Close()

	if IsTerminal(r) {
		t.Error("IsTerminal(pipe reader) = true, want false")
	}
	if IsTerminal(w) {
		t.Error("IsTerminal(pipe writer) = true, want false")
	}
}
