// +build !windows

package process

import (
	"os/exec"
	"syscall"
	"time"
)

func gracefulStop(cmd *exec.Cmd, timeout time.Duration) {
	go func() {
		<-time.NewTimer(timeout).C
		if !cmd.ProcessState.Exited() {
			cmd.Process.Signal(syscall.SIGTERM)
			go func() {
				<-time.NewTimer(timeout).C
				if !cmd.ProcessState.Exited() {
					cmd.Process.Kill()
				}
			}()
		}
	}()
}
