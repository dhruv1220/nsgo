package restart

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"syscall"
)

// restartAfter names the process a restarted app waits for.
const restartAfter = "NSGO_RESTART_AFTER"

// Start starts the program again with args, naming this process for Await to wait for.
func Start(args []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, args...)
	cmd.Env = append(os.Environ(), fmt.Sprintf("%s=%d", restartAfter, os.Getpid()), guardEnv+"=1")
	return cmd.Start()
}

// Await waits, at most 10 seconds, for the process that restarted the app to exit, so the two never run side by side.
func Await() {
	markRestarted()
	pid, err := strconv.ParseUint(os.Getenv(restartAfter), 10, 32)
	os.Unsetenv(restartAfter)
	if err != nil {
		return
	}
	h, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		return
	}
	defer syscall.CloseHandle(h)
	syscall.WaitForSingleObject(h, 10_000)
}
