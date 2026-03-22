package cmd

import "fmt"

// ExitError signals that a command completed with a specific non-zero exit
// code that does not represent a failure (e.g. diff exit 1 = differences found).
type ExitError struct{ Code int }

func (e *ExitError) Error() string {
	return fmt.Sprintf("exit code %d", e.Code)
}
