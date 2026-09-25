package llamaservice

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// RunSupervisor handles the private child entrypoint before normal CLI parsing.
// The parent holds the only write end of stdin's pipe. EOF (including a parent
// crash) terminates the entire router process group, including loaded models.
// There is no shell or process-name matching.
// routerFDEnv names the descriptor the daemon passes for the router's PID —
// its process group — so the daemon can clear the group if this supervisor
// dies without doing it (ADR-0090 amendment 2026-09-25). Unset, nothing is
// written: fd 3 may be anything in a process that did not ask.
const routerFDEnv = "PICODE_LLAMA_ROUTER_FD"

func reportRouter(pid int) {
	fd, err := strconv.Atoi(os.Getenv(routerFDEnv))
	if err != nil || fd < 3 {
		return
	}
	f := os.NewFile(uintptr(fd), "router")
	_, _ = fmt.Fprintf(f, "%d\n", pid)
	_ = f.Close()
}

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
	reportRouter(cmd.Process.Pid)
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

// readRouterGroup reads the PID the supervisor reports, bounded: a supervisor
// that cannot start its router closes the pipe without writing.
func readRouterGroup(r *os.File) int {
	defer r.Close()
	_ = r.SetReadDeadline(time.Now().Add(5 * time.Second))
	line, _ := bufio.NewReader(io.LimitReader(r, 32)).ReadString('\n')
	pid, err := strconv.Atoi(strings.TrimSpace(line))
	if err != nil || pid <= 1 {
		return 0
	}
	return pid
}
