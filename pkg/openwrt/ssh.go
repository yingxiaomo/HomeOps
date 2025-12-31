package openwrt

import (
	"fmt"
	"github.com/yingxiaomo/homeops/config"
	"time"

	"golang.org/x/crypto/ssh"
)

func SSHExec(cmd string) (string, error) {
	conf := &ssh.ClientConfig{
		User: config.AppConfig.OpenWrtUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.AppConfig.OpenWrtPass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.AppConfig.OpenWrtHost, config.AppConfig.OpenWrtPort)
	client, err := ssh.Dial("tcp", addr, conf)
	if err != nil {
		return "", err
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	output, err := session.CombinedOutput(cmd)
	if err != nil {
		return string(output), err
	}

	return string(output), nil
}

func GetSystemStatus() string {
	cmd := "uptime && echo '---' && free -h"
	out, err := SSHExec(cmd)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	return out
}
