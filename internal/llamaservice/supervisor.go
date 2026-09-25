package llamaservice

import (
	"io"
	"os"
	"os/exec"
)

// RunSupervisor handles the private child entrypoint before normal CLI parsing.
// The parent holds the only write end of stdin's pipe. EOF (including a parent
// crash) terminates the entire router process group, including loaded models.
// There is no shell or process-name matching.
func RunSupervisor() {
	if len(os.Args) < 3 || os.Args[1] != "--internal-llama-supervisor" {
		return
	}
	cmd := exec.Command(os.Args[2], os.Args[3:]...)
	cmd.Env = os.Environ()
	configureProcess(cmd)
	if cmd.Start() != nil {
		os.Exit(1)
	}
	// The router is reaped only after its group is killed: while it is an
	// unreaped zombie its PID — and so the group's number — cannot be handed
	// to another process, so the kill below reaches the router's own
	// children and nothing else (a router that died on its own, e.g. by the
	// OOM killer, left the kill racing PID reuse before 2026-09-25).
	exited := make(chan struct{})
	var reap func() error
	go func() { reap = waitExited(cmd); close(exited) }()
	disconnected := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, os.Stdin); close(disconnected) }()
	select {
	case <-exited:
		killProcessGroup(cmd)
		if reap() != nil {
			os.Exit(1)
		}
	case <-disconnected:
		// The kill is ours: the router's exit status says nothing here.
		killProcessGroup(cmd)
		<-exited
		_ = reap()
	}
	os.Exit(0)
}
