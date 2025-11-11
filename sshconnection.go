package main

import (
	"errors"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/ssh"
)

type SSHConnectionInfo struct {
	User string `json:"user"`
	Pass string `json:"pass"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`

	Host string `json:"-"`
	Port int    `json:"-"`
}

type SSHConnection struct {
	Session *ssh.Session
	Stdin   io.WriteCloser
	Stdout  io.Reader
	Stderr  io.Reader
	Client  *ssh.Client
}

func tryConnectToSsh(sshConnInfo SSHConnectionInfo) (*SSHConnection, error) {
	config := &ssh.ClientConfig{
		User: sshConnInfo.User,
		Auth: []ssh.AuthMethod{
			ssh.Password(sshConnInfo.Pass),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshClient, err := ssh.Dial(
		"tcp",
		fmt.Sprintf("%s:%d", sshConnInfo.Host, sshConnInfo.Port),
		config,
	)
	if err != nil {
		log.Println("SSH dial error:", err)
		return nil, errors.New("SSH dial error") // auth failed?
	}

	session, err := sshClient.NewSession()
	if err != nil {
		_ = sshClient.Close()
		log.Println("SSH session error:", err)
		return nil, errors.New("SSH session error")
	}

	stdIn, err := session.StdinPipe()
	if err != nil {
		_ = session.Close()
		_ = sshClient.Close()
		log.Println("STDIN pipe error:", err)
		return nil, errors.New("STDIN pipe error")
	}

	stdOut, err := session.StdoutPipe()
	if err != nil {
		_ = session.Close()
		_ = sshClient.Close()
		log.Println("STDOUT pipe error:", err)
		return nil, errors.New("STDOUT pipe error")
	}

	stdErr, err := session.StderrPipe()
	if err != nil {
		_ = session.Close()
		_ = sshClient.Close()
		return nil, fmt.Errorf("stderr pipe error: %w", err)
	}

	if err := session.RequestPty("xterm", sshConnInfo.Rows, sshConnInfo.Cols, ssh.TerminalModes{}); err != nil {
		_ = session.Close()
		_ = sshClient.Close()
		log.Println("Request PTY error:", err)
		return nil, errors.New("PTY error")
	}

	if err := session.Shell(); err != nil {
		_ = session.Close()
		_ = sshClient.Close()
		log.Println("Start shell error:", err)
		return nil, errors.New("SHELL error")
	}

	return &SSHConnection{
		Session: session,
		Stdin:   stdIn,
		Stdout:  stdOut,
		Stderr:  stdErr,
		Client:  sshClient,
	}, nil
}
