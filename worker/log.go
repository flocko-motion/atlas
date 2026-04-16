package rankedb

import "fmt"

// LogLevel controls SDK output verbosity.
type LogLevel int

const (
	LogDebug LogLevel = iota // everything — HTTP calls, payloads
	LogInfo                  // lifecycle events — init, queue, done (default)
	LogWarn                  // only warnings and errors
)

func (c *Client) log(level LogLevel, format string, args ...any) {
	if level < c.Verbosity {
		return
	}
	prefix := "rankedb"
	switch level {
	case LogDebug:
		prefix = "rankedb [debug]"
	case LogWarn:
		prefix = "rankedb [warn]"
	}
	fmt.Printf("%s: %s\n", prefix, fmt.Sprintf(format, args...))
}
