package executer

import (
	"fmt"
	"github.com/mattn/go-pipeline"
)

func ExecCommand(containerName string) error {
	// setup.shのパス
	commands := []string{
		"docker", "exec", containerName, "bash", "-c", "source setup.sh",
	}

	fmt.Println(commands)
	_, err := pipeline.Output(commands)
	if err != nil {
		fmt.Println(err)
	}
	return err
}
