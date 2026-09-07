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
	finished := make(chan error, 1)
	go func() { finished <- cmd.Wait() }()
	disconnected := make(chan struct{})
	go func() { _, _ = io.Copy(io.Discard, os.Stdin); close(disconnected) }()
	select {
	case err := <-finished:
		killProcessGroup(cmd)
		if err != nil {
			os.Exit(1)
		}
	case <-disconnected:
		killProcessGroup(cmd)
		<-finished
	}
	os.Exit(0)
}
