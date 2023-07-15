package main

import (
	"bufio"
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
)

func main() {
}

func startvpn() {
	cmd := exec.Command("./expect.sh")
	cmd.Stderr = os.Stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Error obtaining stdout: %s", err.Error())
	}

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatalf("Error obtaining stdout: %s", err.Error())
	}

	err = cmd.Start()
	if err != nil {
		log.Fatalf("Error starting program: %s, %s", cmd.Path, err.Error())
	}

	reader := bufio.NewReader(stdout)

	go func(reader io.Reader, waitFor string, send string) {
		var buf []byte
		var err error
		var n int

		buf = make([]byte, 1024)

		var challenge bool

		for {
			log.Printf("Waiting for stdout output ...")
			n, err = reader.Read(buf)
			if err != nil {
				break
			}

			log.Printf("Reading bytes from subprocess [%d]", n)

			lines := buf[:n]
			for {
				var before []byte
				var found bool

				before, lines, found = bytes.Cut(lines, []byte("\r\n"))

				if len(before) > 0 {
					log.Printf("Reading line from subprocess: '%s'", strings.TrimSpace(string(before)))
					log.Printf("Found Line Ending: %v", found)

					if !challenge {
						log.Printf("Comparing '%s' and '%s'", strings.TrimSpace(string(before)), waitFor)

						if string(before[:len(waitFor)]) == waitFor {
							log.Printf("Sending OTP code...")
							stdin.Write([]byte(send))
							challenge = true
						}
					}
				}

				if !found {
					break
				}
			}
		}

	}(reader, "CHALLENGE: One Time Password ", "123456\n")

	time.Sleep(60 * time.Second)
	cmd.Wait()
}

// @Title
// @Description
// @Author
// @Update
