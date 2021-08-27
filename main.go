package main

import (
	"fmt"
	"github.office.opendns.com/quadra/linux-job/pkg/worker"
	logging "log"
	"time"
)

func hold(finished chan bool) {
	time.Sleep(10 * time.Second)
	finished <- true
}

func  main() {
	fmt.Println("staring")
	c := worker.NewController()
	args := []string{"a.sh"}
	p, _ := c.Start("bash", args)


	st, err:= c.Stop(p.Id)
	if err != nil {
		logging.Printf("error %s", err)
	}
	logging.Printf("stop %v", st)

	time.Sleep(10 * time.Second)

	s, err := c.Status(p.Id)
	if err != nil {
		logging.Printf("Error occured %s",err)
	}

	logging.Printf("I ma s %v", s.Status)



}