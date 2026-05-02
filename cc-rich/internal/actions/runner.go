// cc-rich/internal/actions/runner.go
// Runner is the test seam for subprocess dispatch.
package actions

import "os/exec"

// Runner abstracts subprocess execution so tests can swap a mock in.
type Runner interface {
	Cmd(name string, args ...string) error
}

// DefaultRunner shells out via os/exec.
type DefaultRunner struct{}

// Cmd runs name with args, inheriting stderr but discarding stdout.
func (DefaultRunner) Cmd(name string, args ...string) error {
	c := exec.Command(name, args...)
	return c.Run()
}
