package parser

import (
	"io"
	"log"
	"os/exec"
)

func CollectFromCommand(command string, args []string, parser func(io.Reader)) {
	cmd := exec.Command(command, args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Printf("Error creating stdout pipe for %s: %v", command, err)
		return
	}

	if err := cmd.Start(); err != nil {
		log.Printf("Error starting %s: %v", command, err)
		return
	}

	parser(stdout)

	cmd.Wait()
}
