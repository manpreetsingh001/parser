package worker

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.office.opendns.com/quadra/linux-job/pkg/worker/exec"
	"github.office.opendns.com/quadra/linux-job/pkg/worker/log"
	logging "log"
	stdexec "os/exec"
	"sync"
)

// ErrJobNotFound occurs when a process cannot be found in the worker log.
type ErrJobNotFound struct{ msg string }

func (err *ErrJobNotFound) Error() string { return err.msg }



const (
	StdOut = iota
	StdErr
)

type (
	Controller struct {
		access         *sync.RWMutex
		jobs           *sync.Map
		ouputBuffers   map[string][]log.LogOutput
		outputChannels map[string]*log.StreamWatchers
		wg             *sync.WaitGroup
    }

	ProcessStatus struct {
		Id     string
		Status exec.Status
   }
	ProcessStartStatus struct {
		Id string
	}

	Jobtest map[int]*stdexec.Cmd
)

func NewController() *Controller {
	return &Controller{
		access:         &sync.RWMutex{},
		jobs:           &sync.Map{},
		ouputBuffers:   make(map[string][]log.LogOutput),
		outputChannels: make(map[string]*log.StreamWatchers),
		wg:             &sync.WaitGroup{},
	}
}


func (c *Controller) Start(cmd string, args []string) (*ProcessStartStatus, error) {
	if cmd == "" {
		msg := "cannot start job, no executable defined"
        logging.Println(msg)
		return nil, errors.New(msg)
	}

	jobId := uuid.New().String()
	p, err := exec.Start(cmd, args)
	if err != nil {
		logging.Printf("could not start process %s", err)
		return nil, err
	}
	c.wg.Add(1)
	go func(id string, e *exec.Executor) {
		defer c.wg.Done()
		c.run(id, p)
	}(jobId, p)

	logging.Printf("Added Process: %s name: %s args: %s", jobId, cmd, args)

	c.access.Lock()
	c.jobs.Store(jobId, p)
	c.outputChannels[jobId] = log.NewStreamWatcher()
	c.ouputBuffers[jobId] = []log.LogOutput{}
	c.access.Unlock()

	return &ProcessStartStatus{jobId}, nil
}

func (c *Controller) run(id string, proc *exec.Executor) {
	go func(id string, p *exec.Executor) {
		for {
			select {
			case l := <-p.StdOut():
				line := log.LogOutput{Type: StdOut, Bytes: l}
				c.access.Lock()
				c.ouputBuffers[id] = append(c.ouputBuffers[id], line)
				watchers := c.outputChannels[id]
				c.access.Unlock()

                watchers.SendLog(line)
			case l := <-p.StdErr():
				line := log.LogOutput{Type: StdErr, Bytes: l}
				c.access.Lock()
				c.ouputBuffers[id] = append(c.ouputBuffers[id], line)
				watchers := c.outputChannels[id]
				c.access.Unlock()

				watchers.SendLog(line)
			case <-p.Done():
				return
			}

		}
	}(id, proc)

	fs := proc.Wait()

	if fs.ExitCode != 0 || fs.Error != nil {
		// process didn't exit successfully. Log the error
		logging.Printf("Process didn't exit successfully id: %s, exit_code;" +
			" %d, error: %s", id, fs.ExitCode, fs.Error)
	}
}

// Status returns the current status of a job
func (c *Controller) Status(id string) (*ProcessStatus, error) {
	logging.Printf("map %v",c.jobs)
	p, ok := c.jobs.Load(id)
	job, ok := p.(*exec.Executor)
	if ! ok {
		logging.Printf("Cannot find job with the given id %s", id)
		return nil, &ErrJobNotFound{fmt.Sprintf("job not found ID: %s", id)}
	}

	if !ok {
		return nil,fmt.Errorf("does not implement interface")
	}

	st := job.GetStatus()
	logging.Printf("i am status %v" ,st)
	return &ProcessStatus{id, st}, nil
}

// Stop stops a job by sending it's process group a SIGTERM signal
func (c *Controller) Stop(id string) (*ProcessStatus, error) {
	c.access.Lock()
	defer c.access.Unlock()

	p, ok := c.jobs.Load(id)
	if !ok {
		logging.Printf("Cannot find job with the given id %s", id)
		return nil, &ErrJobNotFound{fmt.Sprintf("job not found ID: %s", id)}
	}

	job, ok := p.(*exec.Executor)
	if !ok {
		return nil,fmt.Errorf("does not implement interface")
	}

	if err := job.Stop(); err != nil {
		logging.Printf("Cannot stop process")
		return nil, errors.New(fmt.Sprintf("cannot stop process %v" ,err))
	}

	// return current status to caller
	st := job.GetStatus()
	return &ProcessStatus{id, st}, nil
}

func (c *Controller) WatchOutput(ctx context.Context, id string) (<-chan log.LogOutput, error) {
	out := make(chan log.LogOutput)

	c.access.Lock()
	defer c.access.Unlock()

	j, ok := c.jobs.Load(id)
	if !ok {
		logging.Printf("cannot find job with the given id: %s", id)
		return nil, &ErrJobNotFound{fmt.Sprintf("job not found ID: %s", id)}
	}

	process, ok := j.(*exec.Executor)
	if ! ok {
		return nil,fmt.Errorf("does not implement interface")
	}


	// TODO check existence
	m, ok := c.outputChannels[id]
	if !ok {
		logging.Printf("Cannot find job with the given id: %s", id)
		return nil, &ErrJobNotFound{fmt.Sprintf("job not found ID: %s", id)}
	}

	// TODO check existence
	buffs, ok := c.ouputBuffers[id]
	if !ok {
		logging.Printf("Cannot find job with the given id: %s", id)
		return nil, &ErrJobNotFound{fmt.Sprintf("job not found ID: %s", id)}
	}

	ch := make(chan log.LogOutput, len(buffs))
	m.AddWatcher(ch)

	go func() {
		defer func() {
			c.unWatchOutput(id, out)
			close(out)
		}()

		for _, b := range buffs {
			select {
			case out <- b:
			case <-ctx.Done():
				return
			}
		}

		for {
			select {
			case buf := <-ch:
				out <- buf
			case <-ctx.Done():
				return
			case <-process.Done():
				return
			}
		}
	}()

	return out, nil
}


func (c *Controller) unWatchOutput(id string, ch chan log.LogOutput) {
	c.access.Lock()
	defer c.access.Unlock()

	m, ok := c.outputChannels[id]

	if !ok {
		return
	}

	m.RemoveLogWatcher(ch)
}

// GetOutput returns the byte buffer for the job
func (s *Controller) GetOutput(id string) ([]log.LogOutput, error) {
	s.access.Lock()
	defer s.access.Unlock()

	bytes, ok := s.ouputBuffers[id]

	if !ok {
		logging.Printf("Cannot find job with the given id: %s", id)
		return nil, &ErrJobNotFound{fmt.Sprintf("job not found ID: %s", id)}
	}
	return bytes, nil
}



