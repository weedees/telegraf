// +build windows

package process

import (
	"os/exec"
	"time"
)

func gracefulStop(cmd *exec.Cmd, timeout time.Duration) {
	go func() {
		<-time.NewTimer(timeout).C
		if !cmd.ProcessState.Exited() {
			cmd.Process.Kill()
		}
	}()
}
