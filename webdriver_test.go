// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"bytes"
	"flag"
	"fmt"
	"image/png"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

var (
	target = flag.String("target", "", "target driver (chrome|firefox)")
	wdpath = flag.String("wdpath", "", "path to chromedriver (chrome) or webdriver.xpi (firefox)")
	wdlog  = flag.String("wdlogdir", "", "dir where to dump log files")
)

func init() {
	debug = true
}

var (
	wd      WebDriver
	session *Session
	addr    string
)

var pages = [][]string{
	{"simple", `<!DOCTYPE html><html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"><title>webdriver simple</title></head><body>Simple page</body></html>`},

	{"simple2", `<!DOCTYPE html><html><head><meta http-equiv="Content-Type" content="text/html; charset=utf-8"><title>webdriver simple 2</title></head><body>Simple page 2</body></html>`},

	{"elements", `<!DOCTYPE html><html><body><form name="input" action="" method="get">
<input type="checkbox" name="check1" value="Check1">Check 1<br>
<input type="checkbox" name="check2" value="Check2">Check 2<br><br>
<input type="submit" value="Submit">
</form> 
<div id="foo" style="color:#0000FF">
  <h3>This is a heading</h3>
  <p>This is a <a href="http://golang.com">longwordlinktogolang</a> to a page served by a go server.</p>
</div>
</body></html>`},
}

func handler(w http.ResponseWriter, r *http.Request) {
	for _, page := range pages {
		if page[0] == r.URL.Path[1:] {
			fmt.Fprint(w, page[1])
		}
	}
}

func checkServer(t *testing.T) {
	if addr != "" {
		return
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal("init error: " + err.Error())
	}
	http.HandleFunc("/", handler)
	addr = ln.Addr().String()
	go http.Serve(ln, nil)
}

func checkWebDriver(t *testing.T) {
	if wd == nil {
		switch *target {
		case "":
			t.Fatal(`specify a target browser:
			chrome: go test webdriver -target="chrome" -wdpath="/path/to/chromedriver"
			firefox: go test webdriver -target="firefox" -wdpath="/path/to/webdriver.xpi"`)
		case "chrome":
			wd = startChromedriver(t)
		case "firefox":
			wd = startFirefoxdriver(t)
		default:
			t.Fatal("Unknown target browser: " + *target)
		}
	}
}

func checkSession(t *testing.T) {
	checkServer(t)
	checkWebDriver(t)
	if session == nil {
		desiredCapabilities := Capabilities{"Platform": "Linux"}
		var err error
		session, err = wd.NewSession(desiredCapabilities, Capabilities{})
		if err != nil {
			t.Fatal(err)
		}
	}
}

func startChromedriver(t *testing.T) WebDriver {
	chromedriver := NewChromeDriver(*wdpath)
	if *wdlog != "" {
		chromedriver.LogPath = filepath.Join(*wdlog, "chromedriver.log")
	}
	err := chromedriver.Start()
	if err != nil {
		t.Fatal(err)
	}
	return chromedriver
}

func startFirefoxdriver(t *testing.T) WebDriver {
	firefoxdriver := NewFirefoxDriver("firefox", *wdpath)
	if *wdlog != "" {
		dir := filepath.Dir(*wdlog)
		logfile := filepath.Join(dir, "firefoxdriver.log")
		file, err := os.Open(logfile)
		if err != nil {
			t.Fatal(err)
		}
		err = file.Close()
		if err != nil {
			t.Fatal(err)
		}
		firefoxdriver.SetLogPath(logfile)
	}
	err := firefoxdriver.Start()
	if err != nil {
		t.Fatal(err)
	}
	return firefoxdriver
}

func TestStatus(t *testing.T) {
	checkWebDriver(t)
	_, err := wd.Status()
	if err != nil {
		t.Fatal(err)
	}
}

func TestCreateSession(t *testing.T) {
	checkSession(t)
}

func TestSessions(t *testing.T) {
	switch *target {
	case "chrome", "firefox":
		t.Skip("Not implemented on", *target)
	}
	checkSession(t)
	_, err := wd.Sessions()
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimeouts(t *testing.T) {
	checkSession(t)
	err := session.SetTimeouts("page load", 10000)
	if err != nil {
		t.Error(err)
	}
	err = session.SetTimeoutsAsyncScript(1000)
	if err != nil {
		t.Error(err)
	}
	err = session.SetTimeoutsImplicitWait(1000)
	if err != nil {
		t.Error(err)
	}
}

func TestWindowHandle(t *testing.T) {
	//TODO GetCurrentWindowHandle
	checkSession(t)
	h, err := session.WindowHandle()
	if err != nil {
		t.Fatal(err)
	}
	hv, err := session.WindowHandles()
	if err != nil {
		t.Fatal(err)
	}
	if h.id != hv[0].id {
		t.Fatal("mismatching Window handles")
	}
}

func getUrl(page string) string {
	return "http://" + addr + "/" + page
}

func TestUrl(t *testing.T) {
	checkSession(t)
	url1, url2 := getUrl("simple"), getUrl("simple2")
	err := session.Url(url1)
	if err != nil {
		t.Fatal("simple1: ", err)
	}
	url1b, err := session.GetUrl()
	if err != nil {
		t.Fatal("get url simple1:", err)
	}
	if url1b != url1 {
		t.Fatal("urls differs :" + url1 + " != " + url1b)
	}
	err = session.Url(url2)
	if err != nil {
		t.Fatal("simple2:", err)
	}
	err = session.Refresh()
	if err != nil {
		t.Fatal("refresh:", err)
	}
	err = session.Back()
	if err != nil {
		t.Fatal("back:", err)
	}
	url1b, err = session.GetUrl()
	if err != nil {
		t.Fatal("get url simple 1 after Back:", err)
	}
	if url1b != url1 {
		t.Fatal("back url:" + url1 + " != " + url1b)
	}
	err = session.Forward()
	if err != nil {
		t.Fatal("forward:", err)
	}
	url2b, err := session.GetUrl()
	if err != nil {
		t.Fatal("get url simple 2 after Forward:", err)
	}
	if url2b != url2 {
		t.Fatal("forward url:" + url1 + " != " + url1b)
	}
	title, err := session.Title()
	if err != nil {
		t.Fatal("title simple 2", err)
	}
	if title != "webdriver simple 2" {
		t.Fatal("title \"webdriver simple 2\" not matching: " + title)
	}
	source, err := session.Source()
	if err != nil {
		t.Fatal("source simple 2: ", err)
	}
	if !strings.Contains(source, "<body>Simple page 2</body>") {
		t.Fatalf("source of simple 2 page not matching:\n%s", source)
	}
}

func TestExecuteScript(t *testing.T) {
	checkSession(t)
	value1, value2 := 4, 7
	v := value1 + value2
	script := "return arguments[0] + arguments[1]"
	res, err := session.ExecuteScript(script, []interface{}{value1, value2})
	if err != nil {
		t.Fatal(err)
	}
	x, err := strconv.Atoi(string(res))
	if err != nil || x != v {
		t.Fatal("script returned the wrong result: " + string(res) + " instead of " + strconv.Itoa(v))
	}
}

func TestExecuteScriptAsync(t *testing.T) {
	checkSession(t)
	value1, value2 := 5, 8
	v := value1 + value2
	script := "var cb = arguments[arguments.length - 1];" +
		"cb(arguments[0] + arguments[1]);"
	res, err := session.ExecuteScriptAsync(script, []interface{}{value1, value2})
	if err != nil {
		t.Fatal(err)
	}
	x, err := strconv.Atoi(string(res))
	if err != nil || x != v {
		t.Fatal("script returned the wrong result: " + string(res) + " instead of " + strconv.Itoa(v))
	}
}

func TestExecuteScriptAsyncTimeout(t *testing.T) {
	checkSession(t)
	err := session.SetTimeoutsAsyncScript(1000)
	if err != nil {
		t.Fatal(err)
	}
	script := "window.setTimeout(arguments[arguments.length - 1], 10000)"
	_, err = session.ExecuteScriptAsync(script, []interface{}{})
	if err == nil {
		t.Fatal(err)
	}
	if cerr, ok := err.(CommandError); ok && cerr.StatusCode != 28 {
		t.Fatal(err)
	}
}

func TestScreenshot(t *testing.T) {
	checkSession(t)
	err := session.Url("http://" + addr + "/simple")
	if err != nil {
		t.Fatal(err) //tested already
	}
	buf, err := session.Screenshot()
	if err != nil {
		t.Fatal(err)
	}
	_, err = png.Decode(bytes.NewBuffer(buf))
	if err != nil {
		t.Fatal("returned data is not a png image: " + err.Error())
	}
}

func xTestIME(t *testing.T) {
	checkSession(t)
	// TODO IMEAvailableEngines
	// TODO IMEActiveEngine
	// TODO IsIMEActivated
	// TODO IMEDeactivate
	// TODO IMEActivate
}

func xTestFocusOnFrame(t *testing.T) {
	checkSession(t)
	// TODO FocusOnFrame
}

func TestWindow(t *testing.T) {
	checkSession(t)
	// TODO session.FocusOnWindow
	// TODO session.CloseCurrentWindow
	// TODO wh.SetPosition
	// TODO wh.GetPosition
	// TODO wh.Maximize
	wh := session.GetCurrentWindowHandle()
	size, err := wh.GetSize()
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("width: %d, height: %d", size.Width, size.Height)
	err = wh.SetSize(Size{size.Width / 2, size.Height / 2})
	if err != nil {
		t.Fatal(err)
	}
	time.Sleep(1 * time.Second)
	size2, err := wh.GetSize()
	if err != nil {
		t.Fatal(err)
	}
	//t.Logf("width2: %d, height2: %d", size2.Width, size2.Height)
	time.Sleep(3 * time.Second)
	if size2.Width != size.Width/2 || size2.Height != size.Height/2 {
		t.Log(size2.Width, size.Width/2, size2.Height, size.Height/2)
		t.Fatal("size got with GetSize different from size set with SetSize")
	}
	err = wh.SetSize(Size{size.Width, size.Height})
	if err != nil {
		t.Fatal(err)
	}
}

func xTestCookie(t *testing.T) {
	checkSession(t)
	// TODO GetCookies
	// TODO SetCookie
	// TODO DeleteCookies
	// TODO DeleteCookieByName
}

func TestElements(t *testing.T) {
	checkSession(t)
	err := session.Url(getUrl("elements"))
	if err != nil {
		t.Fatal(err)
	}
	we, err := session.FindElement(ID, "foo")
	if err != nil {
		t.Fatal(err)
	}
	wev, err := session.FindElements(ID, "foo")
	if err != nil {
		t.Fatal(err)
	}
	if we.id != wev[0].id {
		t.Fatal("ids of same element differ")
	}
	we2, err := we.FindElement(PartialLinkText, "linktogo")
	if err != nil {
		t.Fatal(err)
	}
	text, err := we2.Text()
	if err != nil {
		t.Fatal(err)
	}
	if text != "longwordlinktogolang" {
		t.Fatal("got wrong text")
	}
	// TODO FindElement
	// TODO FindElements
	// TODO GetActiveElement
	// TODO element.FindElement
	// TODO element.FindElements
	// TODO element.Click
	// TODO element.Submit
	// TODO element.Text
	// TODO element.SendKeys
	// TODO SendKeysOnActiveElement
	// TODO element.Name
	// TODO element.Clear
	// TODO element.IsSelected
	// TODO element.IsEnabled
	// TODO element.GetAttribute
	// TODO element.Equals
	// TODO element.IsDisplayed
	// TODO element.GetLocation
	// TODO element.GetLocationInView
	// TODO element.Size
}

func xTestCssProperty(t *testing.T) {
	checkSession(t)
	// TODO GetCssProperty
}

func xTestOrientation(t *testing.T) {
	checkSession(t)
	// TODO GetOrientation
	// TODO SetOrientation
}

func xTestAlert(t *testing.T) {
	checkSession(t)
	// TODO GetAlertText
	// TODO SetAlertText
	// TODO AcceptAlert
	// TODO DismissAlert
}

func xTestMouseEvents(t *testing.T) {
	checkSession(t)
	// TODO MoveTo
	// TODO Click
	// TODO ButtonDown
	// TODO ButtonUp
	// TODO DoubleClick
}

func xTestTouchEvents(t *testing.T) {
	checkSession(t)
	// TODO TouchClick
	// TODO TouchDown
	// TODO TouchUp
	// TODO TouchMove
	// TODO TouchScroll
	// TODO TouchDoubleClick
	// TODO TouchLongClick
	// TODO TouchFlick
	// TODO TouchFlickAnywhere
}

func xTestGeoLocation(t *testing.T) {
	checkSession(t)
	// TODO GetGeoLocation
	// TODO SetGeoLocation
}

func xTestStorage(t *testing.T) {
	checkSession(t)
	// TODO LocalStorageGetKeys
	// TODO LocalStorageSetKey
	// TODO LocalStorageClear
	// TODO LocalStorageGetKey
	// TODO LocalStorageRemoveKey
	// TODO LocalStorageSize
	// TODO SessionStorageGetKeys
	// TODO SessionStorageSetKey
	// TODO SessionStorageClear
	// TODO SessionStorageGetKey
	// TODO SessionStorageRemoveKey
	// TODO SessionStorageSize
}

func xTestLog(t *testing.T) {
	checkSession(t)
	// TODO Log
	// TODO LogTypes
}

func xTestHTML5Cache(t *testing.T) {
	checkSession(t)
	// TODO GetHTML5CacheStatus
}

func TestClose(t *testing.T) {
	checkSession(t)
	err := session.Delete()
	if err != nil {
		t.Error(err)
	}
	err = wd.Stop()
	if err != nil {
		t.Error(err)
	}
}
