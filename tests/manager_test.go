package tests

import (
	"github.office.opendns.com/quadra/linux-job/pkg/worker/exec"
	"testing"
	"time"
)

func TestStart(t *testing.T) {
	p, err := exec.Start("sleep", []string{"20"})
	if err != nil {
		t.Errorf("Expected nil error. Got %s", err.Error())
	}
	s := p.GetStatus()
	if  s.State != exec.Running {
		t.Errorf("Expected final state \"Running\", got \"%s\"", s.State)
	}
}

func TestStartStop(t *testing.T) {
	p, err := exec.Start("sleep", []string{"20"})
	if err != nil {
		t.Errorf("Expected nil error. Got %s", err.Error())
	}
	s := p.GetStatus()
	if  s.State != exec.Running {
		t.Errorf("Expected final state \"Running\", got \"%s\"", s.State)
	}

	time.Sleep(5 * time.Second)

	err = p.Stop()
	if err != nil {
		t.Errorf("Expected nil error. Got %s", err.Error())
	}

	st := p.Wait()

	if st.State != exec.Finished {
		t.Errorf("Expected final state \"finished\", got \"%s\"", st.State)
	}

}




