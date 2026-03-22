package cmd

import "fmt"

// ExitError signals that a command exited with a specific non-zero exit
// code that carries semantic meaning rather than indicating failure
// (e.g., diff exits 1 when differences are found).
type ExitError struct{ Code int }

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}
