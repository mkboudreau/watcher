package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
)

type ChangeHandler struct {
	Dir     string
	Command string
}

func NewChangeHandler() *ChangeHandler {
	return &ChangeHandler{}
}

func (handler *ChangeHandler) StartChangeHandler(changes <-chan string) {
	go func() {
		var foundChange bool
		for entry := range changes {
			if entry == IntervalStartToken {
				if debug {
					log.Println(entry)
				}
				foundChange = false
			} else if entry == IntervalEndToken {
				if debug {
					log.Println(entry)
				}
				if foundChange {
					handler.handleChangeFound()
				} else {
					handler.handleNoChangeFound()
				}
			} else {
				log.Println(entry)
				foundChange = true
			}
		}
	}()
}

func (handler *ChangeHandler) handleChangeFound() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatalln("Could not retrieve current directory:", err)
	}

	if handler.Dir != "" {
		err = os.Chdir(handler.Dir)
		if err != nil {
			log.Fatalln("Could not change directory to run command:", err)
		}
	}

	cmdParts := strings.Split(handler.Command, " ")
	cmdActual := cmdParts[0]
	cmd := exec.Command(cmdActual)
	cmd.Args = cmdParts

	var out bytes.Buffer
	cmd.Stdout = &out

	log.Println("Executing command:", cmdActual, "with args", cmdParts)
	err = cmd.Run()
	log.Println(out.String())
	if err != nil {
		log.Fatalln("Could not run command:", err)
	}

	if handler.Dir != "" {
		err = os.Chdir(cwd)
		if err != nil {
			log.Fatalln("Could not change directory back to monitor:", err)
		}
	}
}
func (handler *ChangeHandler) handleNoChangeFound() {
	if debug {
		log.Println("No changes found")
	}
}

func (handler *ChangeHandler) String() string {
	return fmt.Sprintf("Directory [%v]; Command [%v]", handler.Dir, handler.Command)
}
