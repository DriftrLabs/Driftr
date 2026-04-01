package ioutil

import (
	"fmt"
	"io"
	"os"
	"time"
)

// ProgressWriter wraps an io.Writer to report download progress to stderr.
type ProgressWriter struct {
	Dest      io.Writer
	Total     int64 // from Content-Length; -1 if unknown
	written   int64
	lastPrint time.Time
}

func (pw *ProgressWriter) Write(p []byte) (int, error) {
	n, err := pw.Dest.Write(p)
	pw.written += int64(n)

	if time.Since(pw.lastPrint) >= 100*time.Millisecond {
		pw.printProgress()
		pw.lastPrint = time.Now()
	}

	return n, err
}

func (pw *ProgressWriter) printProgress() {
	downloadedMB := float64(pw.written) / (1024 * 1024)
	if pw.Total > 0 {
		totalMB := float64(pw.Total) / (1024 * 1024)
		fmt.Fprintf(os.Stderr, "\r  Downloading: %.1f MB / %.1f MB", downloadedMB, totalMB)
	} else {
		fmt.Fprintf(os.Stderr, "\r  Downloading: %.1f MB", downloadedMB)
	}
}

// Finish prints the final progress line and a newline.
func (pw *ProgressWriter) Finish() {
	pw.printProgress()
	fmt.Fprint(os.Stderr, "\n")
}
