// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// The package implementation a WebDriver that communicate with a browser
// using the JSON Wire Protocol.
//
// See https://code.google.com/p/selenium/wiki/JsonWireProtocol
//
// Example:
//	chromeDriver := webdriver.NewChromeDriver("/path/to/chromedriver")
//	err := chromeDriver.Start()
//	if err != nil {
//		log.Println(err)
//	}
//	desired := webdriver.Capabilities{"Platform": "Linux"}
//	required := webdriver.Capabilities{}
//	session, err := chromeDriver.NewSession(desired, required)
//	if err != nil {
//		log.Println(err)
//	}
//	err = session.Url("http://golang.org")
//	if err != nil {
//		log.Println(err)
//	}
//	time.Sleep(60 * time.Second)
//	session.Delete()
//	chromeDriver.Stop()
//
package webdriver
