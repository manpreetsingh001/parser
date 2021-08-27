package tests

import (
	"github.office.opendns.com/quadra/linux-job/pkg/worker"
	"github.office.opendns.com/quadra/linux-job/pkg/worker/exec"
	"strings"
	"testing"
	"time"
)

func hold(finished chan bool) {
	time.Sleep(10 * time.Second)
	finished <- true
}

func TestGetOutput(t *testing.T) {

	c := worker.NewController()

	msg := "some testing message "
	st, err := c.Start("echo", []string{msg})
	if err != nil {
		t.Fatalf("expected nil error, Got %s", err.Error())
	}

	time.Sleep(5 * time.Second)

	o, err := c.GetOutput(st.Id)
	if err != nil {
		t.Fatalf("expected nil error, Got %s", err.Error())
	}

	var msg2 string
	for _, i := range o {
		msg2 = string(i.Bytes)

	}
	if ! strings.Contains(msg2, msg) {
		t.Errorf("Expected output message : %s got  %s", msg, msg2)
	}
}

func TestControllerStartStop(t *testing.T) {

	c := worker.NewController()

	p, err := c.Start("ping", []string{"localhost"})
	if err != nil {
		t.Fatalf("expected nil error, Got %s", err.Error())
	}

	st, err := c.Stop(p.Id)
	if err != nil {
		t.Fatalf("stop failed with error %s", err.Error())
	}

	time.Sleep(5 * time.Second)
	st, err = c.Status(st.Id)
	if err != nil {
		t.Fatalf("expected nil error, Got %s", err.Error())
	}

	if st.Status.State != exec.Finished {
		t.Errorf("Expected \"Finished\" for job. Got \"%s\"", st.Status.State)
	}
}






