// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"errors"
	"fmt"
	"net"
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
