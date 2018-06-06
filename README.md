This project is not maintained and it is effectively abandoned, if you have interest in this code you should probably clone it.

webdriver
=========

The package implements a WebDriver that communicate with a browser using the JSON Wire Protocol (See https://code.google.com/p/selenium/wiki/JsonWireProtocol).
This is a pure go library and doesn't require a running Selenium driver. It currently supports Firefox (using the WebDriver extension) and Chrome (using the standalone server chromedriver). It should be fairly easy to add other browser that directly implement the wire protocol.

**Version: 0.1**  
Tests are partial and have been run only on Linux (with firefox webdriver 2.32.0 and chromedriver 2.1).

**Install:**  
$ go get github.com/fedesog/webdriver

**Requires:**
* chromedriver (for chrome):  
https://code.google.com/p/chromedriver/  
* webdriver.xpi (for firefox): that is founds in the selenium-server-standalone file  
https://code.google.com/p/selenium/


Example:
--------

    chromeDriver := webdriver.NewChromeDriver("/path/to/chromedriver")
    err := chromeDriver.Start()
    if err != nil {
    	log.Println(err)
    }
    desired := webdriver.Capabilities{"Platform": "Linux"}
    required := webdriver.Capabilities{}
    session, err := chromeDriver.NewSession(desired, required)
    if err != nil {
    	log.Println(err)
    }
    err = session.Url("http://golang.org")
    if err != nil {
    	log.Println(err)
    }
    time.Sleep(10 * time.Second)
    session.Delete()
    chromeDriver.Stop()

