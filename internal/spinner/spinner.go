package spinner

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/briandowns/spinner"
	"golang.org/x/term"
)

// Spinner wraps the briandowns/spinner for use in gw
type Spinner struct {
	s       *spinner.Spinner
	enabled bool
	writer  io.Writer
	message string
}

// New creates a new spinner with a message
func New(message string, w io.Writer) *Spinner {
	// TTY check + NO_COLOR environment variable support
	// Only enable spinner for TTY file descriptors
	enabled := false
	if f, ok := w.(*os.File); ok {
		enabled = term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""
	}

	// CharSets[14] is a clean dot spinner: ⣾⣽⣻⢿⡿⣟⣯⣷
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond, spinner.WithWriter(w))
	s.Suffix = " " + message

	return &Spinner{s: s, enabled: enabled, writer: w, message: message}
}

// Start begins the spinner animation
func (sp *Spinner) Start() {
	if sp.enabled {
		sp.s.Start()
	} else if sp.writer != nil {
		// Fallback for non-TTY: print message
		fmt.Fprintf(sp.writer, "%s\n", sp.message)
	}
}

// Stop ends the spinner animation
func (sp *Spinner) Stop() {
	if sp.enabled && sp.s.Active() {
		sp.s.Stop()
	}
}

// UpdateMessage changes the spinner message while running
func (sp *Spinner) UpdateMessage(message string) {
	sp.message = message
	sp.s.Suffix = " " + message
}
