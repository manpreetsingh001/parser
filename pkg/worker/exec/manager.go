package exec

import (
	"errors"
	"fmt"
	"github.office.opendns.com/quadra/linux-job/pkg/worker/log"
	"os/exec"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	// TODO: determine what SIZE would fit here.This value is used for testing purposes
	DefaultStreamChanSize = 16384
    //TODO may be use enum
	Running  ProcessState = "running"
	Finished ProcessState = "finished"
	Fatal    ProcessState = "failed"
)

type (
	ProcessState string

	Executor struct {
		lock         *sync.RWMutex
		cmd          *exec.Cmd
		streamStdOut chan []byte
		streamStdErr chan []byte
		stdOut       *log.Stream
		stdErr       *log.Stream
		status       *Status
		done         chan struct{}
	}

	Status struct {
		Cmd       string
		PID       int
		State     ProcessState
		Error     error
		ExitCode  int
		StartTime time.Time
		StopTime  time.Time
	}
)

// ErrJobNotActive occurs when termination is attempted on a process that
// is no longer active.
type ErrJobNotActive struct{ msg string }

func (err *ErrJobNotActive) Error() string { return err.msg }

// Start creates a new process and starts it immediately
func Start(name string, args []string) (*Executor, error) {
	// TODO check the separator
	arguments := strings.Join(args, " ")

	status := &Status{
		Cmd:      name + " " + arguments,
		ExitCode: -1,
	}

	cmd := exec.Command(name, args...)
	stdOutChan := make(chan []byte, DefaultStreamChanSize)
	stdErrChan := make(chan []byte, DefaultStreamChanSize)
	stdOut := log.NewStreamOutput(stdOutChan)
	stdErr := log.NewStreamOutput(stdErrChan)
	cmd.Stdout = stdOut
	cmd.Stderr = stdErr

	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}

	m := &Executor{
		cmd:          cmd,
		status:       status,
		done:         make(chan struct{}),
		lock:         &sync.RWMutex{},
		streamStdOut: stdOutChan,
		streamStdErr: stdErrChan,
		stdOut:       stdOut,
		stdErr:       stdErr,
	}

	m.status.StartTime = time.Now()
	if err := m.cmd.Start(); err != nil {
		m.status.StopTime = time.Now()
		m.status.Error = err
		m.status.State = Fatal
		close(m.done)
		return m, err
	}

	m.status.PID = m.cmd.Process.Pid
	m.status.State = Running
	go m.wait()
	return m, nil
}

// Wait waits for the process to finish running and returns the final status
func (e *Executor) Wait() Status {
	<-e.Done()

	return e.GetStatus()
}


// wait waits for a process to finish running and sets it's final status
func (e *Executor) wait() {
	// send the final status and close done channel to signal to
	// goroutines the process has finished running
	defer func() {
		close(e.done)
	}()
	err := e.cmd.Wait()
	now := time.Now()
	exitCode := 0
	e.lock.Lock()
	defer e.lock.Unlock()
	// get the exit code of the process
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			err = nil
			exitCode = exitErr.ExitCode()

			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				if status.Signaled() {
					err = errors.New(exitErr.Error())
				}
			}
		}
	}

	// set final status of the process
	e.status.Error = err
	e.status.ExitCode = exitCode
	e.status.StopTime = now
	e.status.State = Finished
}


// GetStatus returns the current status of the process
func (e *Executor) GetStatus() Status {
	e.lock.RLock()
	defer e.lock.RUnlock()

	return *e.status
}

//TODO how to determine if process is stopped by user or killed by some other reason.
//Stop stops a process by sending it's group a SIGKILL signal
func (e *Executor) Stop() error {
	e.lock.Lock()
	defer e.lock.Unlock()

	if e.status.State != Running {
		return &ErrJobNotActive{fmt.Sprintf("Job is not running")}
	}
	return syscall.Kill(-e.cmd.Process.Pid, syscall.SIGKILL)
}

// StdOut returns the standard output streaming channel
func (e *Executor) StdOut() <-chan []byte {
	return e.streamStdOut
}

// StdErr returns the standard error streaming channel
func (e *Executor) StdErr() <-chan []byte {
	return e.streamStdErr
}

// Done returns a channel that is closed when the process finishes running
func (e *Executor) Done() chan struct{} {
	return e.done
}
