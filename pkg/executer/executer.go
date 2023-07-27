package executer

import (
	"os/exec"
)

func ExecCommand(containerName string) error {
	netns := "vSIX"
	scriptPath := "/etc/nftables/fw-template.rule"

	cmd := exec.Command("ip", "netns", "exec", netns, "nft", "-f", scriptPath)
	err := cmd.Run()

	return err
}
