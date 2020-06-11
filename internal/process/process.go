package process

import (
	"context"
	"fmt"
	"io"
	"log"
	"os/exec"
	"sync"
	"time"
)

// Process is a long-running process manager that will restart processes if they stop.
type Process struct {
	Cmd          *exec.Cmd
	Stdin        io.WriteCloser
	Stdout       io.ReadCloser
	Stderr       io.ReadCloser
	ReadStdoutFn func(io.Reader)
	ReadStderrFn func(io.Reader)
	RestartDelay time.Duration

	cancel     context.CancelFunc
	mainLoopWg sync.WaitGroup
}

// New creates a new process wrapper
func New(command []string) (*Process, error) {
	p := &Process{
		RestartDelay: 5 * time.Second,
	}
	if len(command) > 1 {
		p.Cmd = exec.Command(command[0], command[1:]...)
	} else {
		p.Cmd = exec.Command(command[0])
	}
	var err error
	p.Stdin, err = p.Cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("Error opening stdin pipe: %s", err)
	}

	p.Stdout, err = p.Cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("Error opening stdout pipe: %s", err)
	}

	p.Stderr, err = p.Cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("Error opening stderr pipe: %s", err)
	}

	return p, nil
}

// Start the process
func (p *Process) Start() error {
	p.mainLoopWg.Add(1)

	ctx, cancel := context.WithCancel(context.Background())
	p.cancel = cancel

	if err := p.cmdStart(); err != nil {
		return err
	}

	go func() {
		if err := p.cmdLoop(ctx); err != nil {
			log.Printf("Process quit with message: %s", err.Error())
		}
		p.mainLoopWg.Done()
	}()

	return nil
}

func (p *Process) Stop() {
	if p.cancel != nil {
		p.cancel()
	}
	p.mainLoopWg.Wait()
}

func (p *Process) cmdStart() error {
	log.Printf("Starting process: %s %s", p.Cmd.Path, p.Cmd.Args)

	if err := p.Cmd.Start(); err != nil {
		return fmt.Errorf("Error starting process: %s", err)
	}

	return nil
}

// cmdLoop watches an already running process, restarting it when appropriate.
func (p *Process) cmdLoop(ctx context.Context) error {
	for {
		// Use a buffered channel to ensure goroutine below can exit
		// if `ctx.Done` is selected and nothing reads on `done` anymore
		done := make(chan error, 1)
		go func() {
			done <- p.cmdWait()
		}()

		select {
		case <-ctx.Done():
			if p.Stdin != nil {
				p.Stdin.Close()
				gracefulStop(p.Cmd, 5*time.Second)
			}
			return nil
		case err := <-done:
			log.Printf("Process %s terminated: %s", p.Cmd.Path, err)
			if isQuitting(ctx) {
				return err
			}
		}

		log.Printf("Restarting in %s...", time.Duration(p.RestartDelay))

		select {
		case <-ctx.Done():
			return nil
		case <-time.After(time.Duration(p.RestartDelay)):
			// Continue the loop and restart the process
			if err := p.cmdStart(); err != nil {
				return err
			}
		}
	}
}

func (p *Process) cmdWait() error {
	var wg sync.WaitGroup

	if p.ReadStdoutFn != nil {
		wg.Add(1)
		go func() {
			p.ReadStdoutFn(p.Stdout)
			wg.Done()
		}()
	}

	if p.ReadStderrFn != nil {
		wg.Add(1)
		go func() {
			p.ReadStderrFn(p.Stderr)
			wg.Done()
		}()
	}

	wg.Wait()
	return p.Cmd.Wait()
}

func isQuitting(ctx context.Context) bool {
	select {
	case <-ctx.Done():
		return true
	default:
		return false
	}
}
