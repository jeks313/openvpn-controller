package main

import (
	"bufio"
	"bytes"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strings"
)

// Credentials are used to connect to the VPN. Username and password must go in "filename", OTP is
// provided at run time
type Credentials struct {
	Username    string
	Password    string
	OneTimeCode string
}

func (c Credentials) Store(filename string) error {
	f, err := os.Create(filename)
	if err != nil {
		slog.Error("failed to create credentials file", "error", err)
		return err
	}
	defer func() {
		err = f.Close()
	}()
	err = f.Chmod(0600)
	if err != nil {
		slog.Error("failed to chmod file", "error", err)
		return err
	}
	f.Write([]byte(c.Username))
	f.Write([]byte("\n"))
	f.Write([]byte(c.Password))
	f.Write([]byte("\n"))
	return err
}

// OpenVPN is the main openvpn controller
type OpenVPN struct {
	configFile string
	running    bool
	cmd        *exec.Cmd
	wait       error
}

// NewOpenVPN creates the struct
func NewOpenVPN(configFile string) *OpenVPN {
	return &OpenVPN{
		configFile: configFile,
	}
}

// Wait waits for the process to exit and sets the error code
func (o *OpenVPN) Wait() {
	err := o.cmd.Wait()
	o.wait = err
	o.cmd = nil
	o.running = false
}

// Start starts the opepnvpn process and provides the authentication and otp code
func (o *OpenVPN) Start(otp string) error {
	if o.cmd != nil || o.running {
		slog.Info("process already running")
		return nil
	}
	cmd := exec.Command("./expect.sh")
	o.cmd = cmd
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		slog.Error("error obtaining stdout", "error", err.Error())
		return err
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		slog.Error("error obtaining stdout", "error", err.Error())
		return err
	}

	err = cmd.Start()
	if err != nil {
		slog.Error("error starting program", "path", cmd.Path, "error", err.Error())
		return err
	}

	reader := bufio.NewReader(stdout)

	go func(reader io.Reader, waitFor string, otp string) {
		var buf []byte
		var err error
		var n int

		buf = make([]byte, 1024)

		var challenge bool

		for {
			slog.Info("waiting for stdout output ...")
			n, err = reader.Read(buf)
			if err != nil {
				break
			}

			slog.Info("read bytes from subprocess", "bytes", n)

			lines := buf[:n]
			for {
				var before []byte
				var found bool

				before, lines, found = bytes.Cut(lines, []byte("\r\n"))

				if len(before) > 0 {
					slog.Info("read line from subprocess", "line", strings.TrimSpace(string(before)))
					slog.Info("found line ending", "line", found)

					if !challenge {
						slog.Debug("comparing", "line", strings.TrimSpace(string(before)), "expect", waitFor)

						if string(before[:len(waitFor)]) == waitFor {
							slog.Info("sending otp code...")
							stdin.Write([]byte(otp))
							stdin.Write([]byte("\n"))
							challenge = true
							o.running = true
						}
					}
				}

				if !found {
					break
				}
			}
		}

	}(reader, "CHALLENGE: One Time Password ", otp)

	go func() {
		o.Wait()
	}()

	return err
}
