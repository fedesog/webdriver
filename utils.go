// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"runtime"
	"time"
)

var debug = false

func debugprint(message interface{}) {
	if debug {
		pc, _, line, ok := runtime.Caller(1)
		if ok {
			f := runtime.FuncForPC(pc)
			fmt.Printf("%s:%d: %v\n", f.Name(), line, message)
		} else {
			fmt.Printf("?:?: %s\n", message)
		}
	}
}

//probe d.Port until get a reply or timeout is up
func probePort(port int, timeout time.Duration) error {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	now := time.Now()
	for {
		if conn, err := net.Dial("tcp", address); err == nil {
			if err = conn.Close(); err != nil {
				return err
			}
			break
		}
		if time.Since(now) > timeout {
			return errors.New("start failed: timeout expired")
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}

// starts the browser and file logging.
func runBrowser(exePath string, switches []string, env map[string]string, logFilePath string) (*exec.Cmd, *os.File, error) {
	var logFile *os.File

	cmd := exec.Command(exePath, switches...)
	cmd.Env = os.Environ()
	if len(env) > 0 {
		for k, v := range env {
			cmd.Env = append(cmd.Env, []string{k + "=" + v}...)
		}
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}

	if logFilePath != "" {
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		logFile, err = os.OpenFile(logFilePath, flags, 0640)
		if err != nil {
			return nil, nil, err
		}
		go io.Copy(logFile, stdout)
		go io.Copy(logFile, stderr)
	} else {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	return cmd, logFile, nil
}
