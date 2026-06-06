package process

import (
	"bufio"
	"io"
)

// DefaultLogStreamFactory creates logBuffer instances.
type DefaultLogStreamFactory struct{}

func (DefaultLogStreamFactory) NewLogStream() LogStream {
	return newLogBuffer()
}

// PipeOutput reads lines from stdout and feeds them into stream until EOF.
// Calls stream.Close() when stdout is exhausted.
func PipeOutput(stdout io.ReadCloser, stream LogStream) {
	defer stdout.Close()
	defer stream.Close()
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		stream.WriteLine(scanner.Text())
	}
}
