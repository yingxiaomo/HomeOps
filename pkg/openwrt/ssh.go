package openwrt

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/yingxiaomo/homeops/config"
	"golang.org/x/crypto/ssh"
)

var (
	sshClient *ssh.Client
	sshMutex  sync.Mutex
)

func getClient(forceReconnect bool) (*ssh.Client, error) {
	sshMutex.Lock()
	defer sshMutex.Unlock()

	if sshClient != nil {
		if !forceReconnect {
			return sshClient, nil
		}
		sshClient.Close()
		sshClient = nil
	}

	authMethods := []ssh.AuthMethod{}

	if config.AppConfig.OpenWrtKeyFile != "" {
		key, err := os.ReadFile(config.AppConfig.OpenWrtKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %v", err)
		}

		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, fmt.Errorf("failed to parse SSH key: %v", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		authMethods = append(authMethods, ssh.Password(config.AppConfig.OpenWrtPass))
	}

	conf := &ssh.ClientConfig{
		User:            config.AppConfig.OpenWrtUser,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         5 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", config.AppConfig.OpenWrtHost, config.AppConfig.OpenWrtPort)
	client, err := ssh.Dial("tcp", addr, conf)
	if err != nil {
		return nil, err
	}

	sshClient = client
	return client, nil
}

func SSHExec(cmd string) (string, error) {
	client, err := getClient(false)
	if err != nil {
		return "", err
	}

	session, err := client.NewSession()
	if err != nil {
		client, err = getClient(true)
		if err != nil {
			return "", fmt.Errorf("failed to reconnect: %v", err)
		}
		session, err = client.NewSession()
		if err != nil {
			return "", fmt.Errorf("failed to create session after reconnect: %v", err)
		}
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
