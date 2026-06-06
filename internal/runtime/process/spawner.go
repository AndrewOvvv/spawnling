package process

import (
	"io"
	"os/exec"
)

// ExecSpawner implements ProcessSpawner using os/exec.
type ExecSpawner struct{}

func (ExecSpawner) Spawn(javaPath string, args []string, dir string) (ProcessHandle, error) {
	cmd := exec.Command(javaPath, args...)
	cmd.Dir = dir

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &execHandle{cmd: cmd, stdin: stdin, stdout: stdout}, nil
}

type execHandle struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
}

func (h *execHandle) Stdin() io.WriteCloser { return h.stdin }
func (h *execHandle) Stdout() io.ReadCloser { return h.stdout }
func (h *execHandle) Wait() error           { return h.cmd.Wait() }
func (h *execHandle) Kill() error           { return h.cmd.Process.Kill() }
func (h *execHandle) Pid() int              { return h.cmd.Process.Pid }
