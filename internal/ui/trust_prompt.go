package ui

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// TrustPrompt asks the user whether to trust and run the hook values shown
// in hookLines, sourced from the project config file at projectPath.
//
// Unlike ConfirmPrompt (which defaults to "yes" and renders through
// bubbletea to the process's real stdout), TrustPrompt is a plain-text
// prompt that defaults to "no" and writes only to stderr. This keeps the
// prompt out of `gw start`/`gw checkout`'s stdout, which shell integration
// parses mechanically to `cd` into the new worktree — a bubbletea program
// targeting stdout would corrupt that output.
func (u *DefaultUI) TrustPrompt(projectPath string, hookLines []string) (bool, error) {
	// projectPath and hookLines both come from an untrusted project .gwrc (or
	// its containing path) and are rendered to a real terminal here — the
	// last line of defense before a hook value is approved to run. Quote
	// them so embedded ANSI escapes, carriage returns, or other control
	// characters can't visually spoof what the user is approving.
	fmt.Fprintf(os.Stderr, "\n%s Untrusted project configuration at %s\n", symbolWarningPrefix, strconv.Quote(projectPath))
	fmt.Fprintln(os.Stderr, "The following hook value(s) require approval before they will run:")
	for _, line := range hookLines {
		fmt.Fprintf(os.Stderr, "  %s\n", strconv.Quote(line))
	}
	fmt.Fprint(os.Stderr, "Trust and run these hooks? (y/N): ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		// Fail closed on any read error (including EOF without a trailing
		// newline) rather than parsing whatever partial input was read.
		return false, nil
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}

// symbolWarningPrefix mirrors the "⚠" warning glyph used elsewhere in the
// CLI's output, without importing the cmd package (which would create an
// import cycle since cmd imports ui).
const symbolWarningPrefix = "⚠"
