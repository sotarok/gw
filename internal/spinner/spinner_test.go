package spinner

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestNew(t *testing.T) {
	t.Run("creates spinner with non-TTY writer", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("loading...", &buf)

		if sp == nil {
			t.Fatal("expected non-nil spinner")
		}
		if sp.enabled {
			t.Error("expected enabled to be false for non-TTY writer")
		}
		if sp.message != "loading..." {
			t.Errorf("expected message 'loading...', got %q", sp.message)
		}
		if sp.writer != &buf {
			t.Error("expected writer to be the provided buffer")
		}
		if sp.s == nil {
			t.Error("expected internal spinner to be non-nil")
		}
	})

	t.Run("creates spinner with nil writer", func(t *testing.T) {
		sp := New("test", nil)

		if sp == nil {
			t.Fatal("expected non-nil spinner")
		}
		if sp.enabled {
			t.Error("expected enabled to be false for nil writer")
		}
		if sp.writer != nil {
			t.Error("expected writer to be nil")
		}
	})

	t.Run("NO_COLOR disables spinner even for os.File", func(t *testing.T) {
		// Set NO_COLOR environment variable
		t.Setenv("NO_COLOR", "1")

		// Use os.Stderr which is an *os.File, but NO_COLOR should keep it disabled
		// Note: In CI/test environments, stderr may not be a terminal anyway,
		// but NO_COLOR should ensure it's disabled regardless
		sp := New("test", os.Stderr)

		if sp == nil {
			t.Fatal("expected non-nil spinner")
		}
		// enabled should be false because NO_COLOR is set
		if sp.enabled {
			t.Error("expected enabled to be false when NO_COLOR is set")
		}
	})

	t.Run("empty NO_COLOR does not disable spinner", func(t *testing.T) {
		// Ensure NO_COLOR is empty
		t.Setenv("NO_COLOR", "")

		var buf bytes.Buffer
		sp := New("test", &buf)

		// Still disabled because bytes.Buffer is not a TTY
		if sp.enabled {
			t.Error("expected enabled to be false for non-TTY writer")
		}
	})

	t.Run("sets suffix on internal spinner", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("my message", &buf)

		if sp.s.Suffix != " my message" {
			t.Errorf("expected suffix ' my message', got %q", sp.s.Suffix)
		}
	})
}

func TestStart(t *testing.T) {
	t.Run("writes fallback message for non-TTY writer", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("installing dependencies", &buf)

		sp.Start()

		output := buf.String()
		if !strings.Contains(output, "installing dependencies") {
			t.Errorf("expected output to contain 'installing dependencies', got %q", output)
		}
		if !strings.HasSuffix(output, "\n") {
			t.Error("expected output to end with newline")
		}
	})

	t.Run("does not panic with nil writer", func(t *testing.T) {
		sp := New("test", nil)
		sp.writer = nil

		// Should not panic
		sp.Start()
	})

	t.Run("writes message only once per start call", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("processing", &buf)

		sp.Start()
		output := buf.String()
		count := strings.Count(output, "processing")
		if count != 1 {
			t.Errorf("expected message to appear once, appeared %d times", count)
		}
	})

	t.Run("can start multiple times", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("loading", &buf)

		sp.Start()
		sp.Start()

		output := buf.String()
		count := strings.Count(output, "loading")
		if count != 2 {
			t.Errorf("expected message to appear twice, appeared %d times", count)
		}
	})
}

func TestStop(t *testing.T) {
	t.Run("does not panic when not enabled", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("test", &buf)

		// Should not panic even though spinner was never started
		sp.Stop()
	})

	t.Run("does not panic when called multiple times", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("test", &buf)

		sp.Start()
		sp.Stop()
		sp.Stop() // Should not panic on double stop
	})
}

func TestUpdateMessage(t *testing.T) {
	t.Run("updates message and suffix", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("initial message", &buf)

		sp.UpdateMessage("updated message")

		if sp.message != "updated message" {
			t.Errorf("expected message 'updated message', got %q", sp.message)
		}
		if sp.s.Suffix != " updated message" {
			t.Errorf("expected suffix ' updated message', got %q", sp.s.Suffix)
		}
	})

	t.Run("updates message multiple times", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("first", &buf)

		sp.UpdateMessage("second")
		sp.UpdateMessage("third")

		if sp.message != "third" {
			t.Errorf("expected message 'third', got %q", sp.message)
		}
		if sp.s.Suffix != " third" {
			t.Errorf("expected suffix ' third', got %q", sp.s.Suffix)
		}
	})

	t.Run("updated message is used by Start fallback", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("original", &buf)

		sp.UpdateMessage("new message")
		sp.Start()

		output := buf.String()
		if !strings.Contains(output, "new message") {
			t.Errorf("expected output to contain 'new message', got %q", output)
		}
		if strings.Contains(output, "original") {
			t.Error("expected output to not contain 'original' after update")
		}
	})
}

func TestSpinnerIntegration(t *testing.T) {
	t.Run("full lifecycle with non-TTY writer", func(t *testing.T) {
		var buf bytes.Buffer
		sp := New("step 1", &buf)

		// Start writes fallback
		sp.Start()
		if !strings.Contains(buf.String(), "step 1") {
			t.Error("expected 'step 1' in output after Start")
		}

		// Update changes message
		sp.UpdateMessage("step 2")
		if sp.message != "step 2" {
			t.Error("expected message to be 'step 2'")
		}

		// Stop should not panic
		sp.Stop()
	})
}
