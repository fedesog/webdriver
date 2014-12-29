// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"

	//	"fmt"
	//	"net/http"
)

type WebDriver interface {
	//Start webdriver service
	Start() error
	//Stop webdriver service
	Stop() error
	//Query the server's status.
	Status() (*Status, error)
	//Create a new session.
	NewSession(desired, required Capabilities) (*Session, error)
	//Returns a list of the currently active sessions.
	Sessions() ([]Session, error)

	do(params interface{}, method, urlFormat string, urlParams ...interface{}) (string, []byte, error)
}

//typing saver
type params map[string]interface{}

//Server details.
type Status struct {
	Build Build
	OS    OS
}

//Server built details.
type Build struct {
	Version  string
	Revision string
	Time     string
}

//Server OS details
type OS struct {
	Arch    string
	Name    string
	Version string
}

//Capabilities is a map that stores capabilities of a session.
type Capabilities map[string]interface{}

//A session.
type Session struct {
	Id           string
	Capabilities Capabilities
	wd           WebDriver
}

type WindowHandle struct {
	s  *Session
	id string
}

type Size struct {
	Width  int
	Height int
}

type Position struct {
	X int
	Y int
}

type FindElementStrategy string

const (
	//Returns an element whose class name contains the search value; compound class names are not permitted.
	ClassName = FindElementStrategy("class name")
	//Returns an element matching a CSS selector.
	CSS_Selector = FindElementStrategy("css selector")
	//Returns an element whose ID attribute matches the search value.
	ID = FindElementStrategy("id")
	//Returns an element whose NAME attribute matches the search value.
	Name = FindElementStrategy("name")
	//Returns an anchor element whose visible text matches the search value.
	LinkText = FindElementStrategy("link text")
	//Returns an anchor element whose visible text partially matches the search value.
	PartialLinkText = FindElementStrategy("partial link text")
	//Returns an element whose tag name matches the search value.
	TagName = FindElementStrategy("tag name")
	//Returns an element matching an XPath expression.
	XPath = FindElementStrategy("xpath")
)

type element struct {
	ELEMENT string
}

type WebElement struct {
	s  *Session
	id string
}

type Cookie struct {
	Name   string
	Value  string
	Path   string
	Domain string
	Secure bool
	Expiry int
}

type GeoLocation struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
	Altitude  float64 `json:"altitude"`
}

type LogLevel string

const (
	LogAll     = LogLevel("ALL")
	LogDebug   = LogLevel("DEBUG")
	LogInfo    = LogLevel("INFO")
	LogWarning = LogLevel("WARNING")
	LogSevere  = LogLevel("SEVERE")
	LogOff     = LogLevel("OFF")
)

type LogEntry struct {
	TimeStamp int //TODO timestamp number type?
	Level     string
	Message   string
}

type HTML5CacheStatus int

const (
	CacheStatusUncached    = HTML5CacheStatus(0)
	CacheStatusIdle        = HTML5CacheStatus(1)
	CacheStatusChecking    = HTML5CacheStatus(2)
	CacheStatusDownloading = HTML5CacheStatus(3)
	CacheStatusUpdateReady = HTML5CacheStatus(4)
	CacheStatusObsolete    = HTML5CacheStatus(5)
)

////////////////////////////////////////////////////////////////////////////////
// COMMAND LIST
// Command descriptions are from:
// https://code.google.com/p/selenium/wiki/JsonWireProtocol
////////////////////////////////////////////////////////////////////////////////

//Retrieve the capabilities of the specified session.
func (s Session) GetCapabilities() Capabilities {
	// GET /session/:sessionId
	// I have the capabilities stored in Session already
	return s.Capabilities
}

//Delete the session.
func (s Session) Delete() error {
	_, _, err := s.wd.do(nil, "DELETE", "/session/%s", s.Id)
	return err
}

//Configure the amount of time that a particular type of operation can execute for before they are aborted and a |Timeout| error is returned to the client.  Valid values are: "script" for script timeouts, "implicit" for modifying the implicit wait timeout and "page load" for setting a page load timeout.
func (s Session) SetTimeouts(typ string, ms int) error {
	p := params{"type": typ, "ms": ms}
	_, _, err := s.wd.do(p, "POST", "/session/%s/timeouts", s.Id)
	return err
}

//Set the amount of time, in milliseconds, that asynchronous scripts executed by ExecuteScriptAsync() are permitted to run before they are aborted and a |Timeout| error is returned to the client.
func (s Session) SetTimeoutsAsyncScript(ms int) error {
	p := params{"ms": ms}
	_, _, err := s.wd.do(p, "POST", "/session/%s/timeouts/async_script", s.Id)
	return err
}

//Set the amount of time the driver should wait when searching for elements. When searching for a single element, the driver should poll the page until an element is found or the timeout expires, whichever occurs first. When searching for multiple elements, the driver should poll the page until at least one element is found or the timeout expires, at which point it should return an empty list.
//If this command is never sent, the driver should default to an implicit wait of 0ms.
func (s Session) SetTimeoutsImplicitWait(ms int) error {
	p := params{"ms": ms}
	_, _, err := s.wd.do(p, "POST", "/session/%s/timeouts/implicit_wait", s.Id)
	return err
}

func (s Session) GetCurrentWindowHandle() WindowHandle {
	return WindowHandle{&s, "current"}
}

//Retrieve the current window handle.
func (s Session) WindowHandle() (WindowHandle, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/window_handle", s.Id)
	if err != nil {
		return WindowHandle{}, err
	}
	var handle string
	err = json.Unmarshal(data, &handle)
	return WindowHandle{&s, handle}, err
}

//Retrieve the list of all window handles available to the session.
func (s Session) WindowHandles() ([]WindowHandle, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/window_handles", s.Id)
	if err != nil {
		return nil, err
	}
	var hv []string
	err = json.Unmarshal(data, &hv)
	if err != nil {
		return nil, err
	}
	var handles = make([]WindowHandle, len(hv))
	for i, h := range hv {
		handles[i] = WindowHandle{&s, h}
	}
	return handles, nil
}

//Retrieve the URL of the current page.
func (s Session) GetUrl() (string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/url", s.Id)
	if err != nil {
		return "", err
	}
	var url string
	err = json.Unmarshal(data, &url)
	return url, err
}

//Navigate to a new URL.
func (s Session) Url(url string) error {
	p := params{"url": url}
	_, _, err := s.wd.do(p, "POST", "/session/%s/url", s.Id)
	return err
}

//Navigate forwards in the browser history, if possible.
func (s Session) Forward() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/forward", s.Id)
	return err
}

//Navigate backwards in the browser history, if possible.
func (s Session) Back() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/back", s.Id)
	return err
}

//Refresh the current page.
func (s Session) Refresh() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/refresh", s.Id)
	return err
}

// Inject a snippet of JavaScript into the page for execution in the context of the currently selected frame. The executed script is assumed to be synchronous and the result of evaluating the script is returned to the client.
// The script argument defines the script to execute in the form of a function body. The value returned by that function will be returned to the client. The function will be invoked with the provided args array and the values may be accessed via the arguments object in the order specified.
// Arguments may be any JSON-primitive, array, or JSON object. JSON objects that define a WebElement reference will be converted to the corresponding DOM element. Likewise, any WebElements in the script result will be returned to the client as WebElement JSON objects.
func (s Session) ExecuteScript(script string, args []interface{}) ([]byte, error) {
	p := params{"script": script, "args": args}
	_, data, err := s.wd.do(p, "POST", "/session/%s/execute", s.Id)
	return data, err
}

// Inject a snippet of JavaScript into the page for execution in the context of the currently selected frame. The executed script is assumed to be asynchronous and must signal that is done by invoking the provided callback, which is always provided as the final argument to the function. The value to this callback will be returned to the client.
// Asynchronous script commands may not span page loads. If an unload event is fired while waiting for a script result, an error should be returned to the client.
// The script argument defines the script to execute in teh form of a function body. The function will be invoked with the provided args array and the values may be accessed via the arguments object in the order specified. The final argument will always be a callback function that must be invoked to signal that the script has finished.
// Arguments may be any JSON-primitive, array, or JSON object. JSON objects that define a WebElement reference will be converted to the corresponding DOM element. Likewise, any WebElements in the script result will be returned to the client as WebElement JSON objects.
func (s Session) ExecuteScriptAsync(script string, args []interface{}) ([]byte, error) {
	p := params{"script": script, "args": args}
	_, data, err := s.wd.do(p, "POST", "/session/%s/execute_async", s.Id)
	return data, err
}

//Take a screenshot of the current page.
func (s Session) Screenshot() ([]byte, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/screenshot", s.Id)
	if err != nil {
		return nil, err
	}
	reader := bytes.NewBuffer(data[1 : len(data)-1])
	decoder := base64.NewDecoder(base64.StdEncoding, reader)
	return ioutil.ReadAll(decoder)
}

//List all available engines on the machine.
func (s Session) IMEAvailableEngines() ([]string, error) {
	_, data, err := s.wd.do(nil, "GET", "session/%s/ime/available_engines", s.Id)
	if err != nil {
		return nil, err
	}
	var engines []string
	err = json.Unmarshal(data, &engines)
	return engines, err
}

//Get the name of the active IME engine.
func (s Session) IMEActiveEngine() (string, error) {
	_, data, err := s.wd.do(nil, "GET", "session/%s/ime/active_engine", s.Id)
	if err != nil {
		return "", err
	}
	var engine string
	err = json.Unmarshal(data, &engine)
	return engine, err
}

//Indicates whether IME input is active at the moment (not if it's available).
func (s Session) IsIMEActivated() (bool, error) {
	_, data, err := s.wd.do(nil, "GET", "session/%s/ime/activated", s.Id)
	if err != nil {
		return false, err
	}
	var activated bool
	err = json.Unmarshal(data, &activated)
	return activated, err
}

//De-activates the currently-active IME engine.
func (s Session) IMEDeactivate() error {
	_, _, err := s.wd.do(nil, "GET", "session/%s/ime/deactivate", s.Id)
	return err
}

//Make an engines that is available (appears on the list returned by getAvailableEngines) active.
func (s Session) IMEActivate(engine string) error {
	p := params{"engine": engine}
	_, _, err := s.wd.do(p, "POST", "/session/%s/ime/activate", s.Id)
	return err
}

//Change focus to another frame on the page.
func (s Session) FocusOnFrame(frameId interface{}) error {
	if frameId != nil {
		switch frameId.(type) {
		case string:
		case int:
		case WebElement:
		default:
			return errors.New("invalid frame, must be string|int|nil|WebElement")
		}
	}
	p := params{"id": frameId}
	_, _, err := s.wd.do(p, "POST", "/session/%s/frame", s.Id)
	return err
}

// Change focus back to parent frame
func (s Session) FocusParentFrame() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/frame/parent", s.Id)
	return err
}

//Change focus to another window. The window to change focus to may be specified by its server assigned window handle, or by the value of its name attribute.
func (s Session) FocusOnWindow(name string) error {
	p := params{"name": name}
	_, _, err := s.wd.do(p, "POST", "/session/%s/window", s.Id)
	return err
}

//Close the current window.
func (s Session) CloseCurrentWindow() error {
	_, _, err := s.wd.do(nil, "DELETE", "/session/%s/window", s.Id)
	return err
}

//Change the size of the specified window.
func (w WindowHandle) SetSize(size Size) error {
	p := params{"width": size.Width, "height": size.Height}
	_, _, err := w.s.wd.do(p, "POST", "/session/%s/window/%s/size", w.s.Id, w.id)
	return err
}

//Get the size of the specified window.
func (w WindowHandle) GetSize() (Size, error) {
	_, data, err := w.s.wd.do(nil, "GET", "/session/%s/window/%s/size", w.s.Id, w.id)
	if err != nil {
		return Size{}, err
	}
	var outSize Size
	err = json.Unmarshal(data, &outSize)
	return outSize, err
}

//Change the position of the specified window.
func (w WindowHandle) SetPosition(position Position) error {
	p := params{"x": position.X, "y": position.Y}
	_, _, err := w.s.wd.do(p, "POST", "/session/%s/window/%s/position", w.s.Id, w.id)
	return err
}

//Get the position of the specified window.
func (w WindowHandle) GetPosition() (Position, error) {
	_, data, err := w.s.wd.do(nil, "GET", "/session/%s/window/%s/position", w.s.Id, w.id)
	if err != nil {
		return Position{}, err
	}
	var position Position
	err = json.Unmarshal(data, &position)
	return position, err
}

//Maximize the specified window if not already maximized.
func (w WindowHandle) MaximizeWindow() error {
	_, _, err := w.s.wd.do(nil, "POST", "/session/%s/window/%s/maximize", w.s.Id, w.id)
	return err
}

//Retrieve all cookies visible to the current page.
func (s Session) GetCookies() ([]Cookie, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/cookie", s.Id)
	if err != nil {
		return nil, err
	}
	var cookies []Cookie
	err = json.Unmarshal(data, &cookies)
	return cookies, err
}

//Set a cookie.
func (s Session) SetCookie(cookie Cookie) error {
	p := params{"cookie": cookie}
	_, _, err := s.wd.do(p, "POST", "/session/%s/cookie", s.Id)
	return err
}

//Delete all cookies visible to the current page.
func (s Session) DeleteCookies() error {
	_, _, err := s.wd.do(nil, "DELETE", "/session/%s/cookie", s.Id)
	return err
}

//Delete the cookie with the given name.
func (s Session) DeleteCookieByName(name string) error {
	_, _, err := s.wd.do(nil, "DELETE", "/session/%s/cookie/%s", s.Id, name)
	return err
}

//Get the current page source.
func (s Session) Source() (string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/source", s.Id)
	if err != nil {
		return "", err
	}
	var source string
	err = json.Unmarshal(data, &source)
	return source, err
}

//Get the current page title.
func (s Session) Title() (string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/title", s.Id)
	if err != nil {
		return "", err
	}
	var title string
	err = json.Unmarshal(data, &title)
	return title, err
}

func (s Session) WebElementFromId(id string) WebElement {
	return WebElement{&s, id}
}

//Search for an element on the page, starting from the document root.
func (s Session) FindElement(using FindElementStrategy, value string) (WebElement, error) {
	p := params{"using": using, "value": value}
	_, data, err := s.wd.do(p, "POST", "/session/%s/element", s.Id)
	if err != nil {
		return WebElement{}, err
	}
	var elem element
	err = json.Unmarshal(data, &elem)
	return WebElement{&s, elem.ELEMENT}, err
}

//Search for multiple elements on the page, starting from the document root.
func (s Session) FindElements(using FindElementStrategy, value string) ([]WebElement, error) {
	p := params{"using": using, "value": value}
	_, data, err := s.wd.do(p, "POST", "/session/%s/elements", s.Id)
	if err != nil {
		return nil, err
	}
	var v []element
	err = json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	elements := make([]WebElement, len(v))
	for i, elem := range v {
		elements[i] = WebElement{&s, elem.ELEMENT}
	}
	return elements, err
}

//Get the element on the page that currently has focus.
func (s Session) GetActiveElement() (WebElement, error) {
	_, data, err := s.wd.do(nil, "POST", "/session/%s/element/active", s.Id)
	if err != nil {
		return WebElement{}, err
	}
	var elem element
	err = json.Unmarshal(data, &elem)
	return WebElement{&s, elem.ELEMENT}, err
}

//Describe the identified element. This command is reserved for future use; its return type is currently undefined.
/*func (e WebElement) Id() error {
	// GET /session/:sessionId/element/:id
}*/

//Search for an element on the page, starting from the identified element.
func (e WebElement) FindElement(using FindElementStrategy, value string) (WebElement, error) {
	p := params{"using": using, "value": value}
	_, data, err := e.s.wd.do(p, "POST", "/session/%s/element/%s/element", e.s.Id, e.id)
	if err != nil {
		return WebElement{}, err
	}
	var elem element
	err = json.Unmarshal(data, &elem)
	return WebElement{e.s, elem.ELEMENT}, err
}

//Search for multiple elements on the page, starting from the identified element.
func (e WebElement) FindElements(using FindElementStrategy, value string) ([]WebElement, error) {
	p := params{"using": using, "value": value}
	_, data, err := e.s.wd.do(p, "POST", "/session/%s/element/%s/elements", e.s.Id, e.id)
	if err != nil {
		return nil, err
	}
	var v []element
	err = json.Unmarshal(data, &v)
	if err != nil {
		return nil, err
	}
	elements := make([]WebElement, len(v))
	for i, z := range v {
		elements[i] = WebElement{e.s, z.ELEMENT}
	}
	return elements, err
}

//Click on an element.
func (e WebElement) Click() error {
	_, _, err := e.s.wd.do(nil, "POST", "/session/%s/element/%s/click", e.s.Id, e.id)
	return err
}

//Submit a FORM element.
func (e WebElement) Submit() error {
	_, _, err := e.s.wd.do(nil, "POST", "/session/%s/element/%s/submit", e.s.Id, e.id)
	return err
}

//Returns the visible text for the element.
func (e WebElement) Text() (string, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/text", e.s.Id, e.id)
	if err != nil {
		return "", err
	}
	var text string
	err = json.Unmarshal(data, &text)
	return text, err
}

//Send a sequence of key strokes to an element.
func (e WebElement) SendKeys(sequence string) error {
	keys := make([]string, len(sequence))
	for i, k := range sequence {
		keys[i] = string(k)
	}
	p := params{"value": keys}
	_, _, err := e.s.wd.do(p, "POST", "/session/%s/element/%s/value", e.s.Id, e.id)
	return err
}

//Send a sequence of key strokes to the active element.
func (s Session) SendKeysOnActiveElement(sequence string) error {
	keys := make([]string, len(sequence))
	for i, k := range sequence {
		keys[i] = string(k)
	}
	p := params{"value": keys}
	_, _, err := s.wd.do(p, "POST", "/session/%s/keys", s.Id)
	return err
}

//Query for an element's tag name.
func (e WebElement) Name() (string, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/name", e.s.Id, e.id)
	if err != nil {
		return "", err
	}
	var name string
	err = json.Unmarshal(data, &name)
	return name, err
}

//Clear a TEXTAREA or text INPUT element's value.
func (e WebElement) Clear() error {
	_, _, err := e.s.wd.do(nil, "POST", "/session/%s/element/%s/clear", e.s.Id, e.id)
	return err
}

//Determine if an OPTION element, or an INPUT element of type checkbox or radiobutton is currently selected.
func (e WebElement) IsSelected() (bool, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/value", e.s.Id, e.id)
	if err != nil {
		return false, err
	}
	var isSelected bool
	err = json.Unmarshal(data, &isSelected)
	return isSelected, err
}

//Determine if an element is currently enabled.
func (e WebElement) IsEnabled() (bool, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/enabled", e.s.Id, e.id)
	if err != nil {
		return false, err
	}
	var isEnabled bool
	err = json.Unmarshal(data, &isEnabled)
	return isEnabled, err
}

//Get the value of an element's attribute.
func (e WebElement) GetAttribute(name string) (string, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/attribute/%s", e.s.Id, e.id, name)
	if err != nil {
		return "", err
	}
	var attribute string
	err = json.Unmarshal(data, &attribute)
	return attribute, err
	//return z, e.do("GET", u, nil, &z)
}

//Test if two element IDs refer to the same DOM element.
func (e WebElement) Equal(element WebElement) (bool, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/equal/%s", e.s.Id, e.id, element.id)
	if err != nil {
		return false, err
	}
	var equal bool
	err = json.Unmarshal(data, &equal)
	return equal, err
}

//Determine if an element is currently displayed.
func (e WebElement) IsDisplayed() (bool, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/displayed", e.s.Id, e.id)
	if err != nil {
		return false, err
	}
	var isDisplayed bool
	err = json.Unmarshal(data, &isDisplayed)
	return isDisplayed, err
}

//Determine an element's location on the page.
//The point (0, 0) refers to the upper-left corner of the page. The element's coordinates are returned as a JSON object with x and y properties.
func (e WebElement) GetLocation() (Position, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/location", e.s.Id, e.id)
	if err != nil {
		return Position{}, err
	}
	var position Position
	err = json.Unmarshal(data, &position)
	return position, err
}

//Determine an element's location on the screen once it has been scrolled into view.
//
//Note: This is considered an internal command and should only be used to determine an element's location for correctly generating native events.
func (e WebElement) GetLocationInView() (Position, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/location_in_view", e.s.Id, e.id)
	if err != nil {
		return Position{}, err
	}
	var position Position
	err = json.Unmarshal(data, &position)
	return position, err
}

//Determine an element's size in pixels.
func (e WebElement) Size() (Size, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/size", e.s.Id, e.id)
	if err != nil {
		return Size{}, err
	}
	var size Size
	err = json.Unmarshal(data, &size)
	return size, err
}

//Query the value of an element's computed CSS property.
func (e WebElement) GetCssProperty(name string) (string, error) {
	_, data, err := e.s.wd.do(nil, "GET", "/session/%s/element/%s/css/%s", e.s.Id, e.id, name)
	if err != nil {
		return "", err
	}
	var cssProperty string
	err = json.Unmarshal(data, &cssProperty)
	return cssProperty, err
}

type ScreenOrientation string

const (
	//TODO what is actually returned?
	LANDSCAPE = iota
	PORTRAIT
)

//Get the current browser orientation.
func (s Session) GetOrientation() (ScreenOrientation, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/orientation", s.Id)
	if err != nil {
		return "", err
	}
	var orientation ScreenOrientation
	err = json.Unmarshal(data, &orientation)
	return orientation, err
}

//Set the browser orientation.
func (s Session) SetOrientation(orientation ScreenOrientation) error {
	p := params{"orientation": orientation}
	_, _, err := s.wd.do(p, "POST", "/session/%s/orientation", s.Id)
	return err
}

//Gets the text of the currently displayed JavaScript alert(), confirm(), or prompt() dialog.
func (s Session) GetAlertText() (string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/alert_text", s.Id)
	if err != nil {
		return "", err
	}
	var alertText string
	err = json.Unmarshal(data, &alertText)
	return alertText, err
}

//Sends keystrokes to a JavaScript prompt() dialog.
func (s Session) SetAlertText(text string) error {
	p := params{"text": text}
	_, _, err := s.wd.do(p, "POST", "/session/%s/alert_text", s.Id)
	return err
}

//Accepts the currently displayed alert dialog.
func (s Session) AcceptAlert() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/accept_alert", s.Id)
	return err
}

//Dismisses the currently displayed alert dialog.
func (s Session) DismissAlert() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/dismiss_alert", s.Id)
	return err
}

//Move the mouse by an offset of the specificed element.
//If no element is specified, the move is relative to the current mouse cursor. If an element is provided but no offset, the mouse will be moved to the center of the element. If the element is not visible, it will be scrolled into view.
func (s Session) MoveTo(element WebElement, xoffset, yoffset int) error {
	p := params{"element": element.id, "xoffset": xoffset, "yoffset": yoffset}
	_, _, err := s.wd.do(p, "POST", "/session/%s/moveto", s.Id)
	return err
}

type MouseButton int

const (
	LeftButton   = MouseButton(0)
	MiddleButton = MouseButton(1)
	RightButton  = MouseButton(2)
)

//Click any mouse button (at the coordinates set by the last moveto command).
//
//Note that calling this command after calling buttondown and before calling button up (or any out-of-order interactions sequence) will yield undefined behaviour).
func (s Session) Click(button MouseButton) error {
	p := params{"button": button}
	_, _, err := s.wd.do(p, "POST", "/session/%s/click", s.Id)
	return err
}

//Click and hold the left mouse button (at the coordinates set by the last moveto command).
func (s Session) ButtonDown(button MouseButton) error {
	p := params{"button": button}
	_, _, err := s.wd.do(p, "POST", "/session/%s/buttondown", s.Id)
	return err
}

//Releases the mouse button previously held (where the mouse is currently at).
func (s Session) ButtonUp(button MouseButton) error {
	p := params{"button": button}
	_, _, err := s.wd.do(p, "POST", "/session/%s/buttonup", s.Id)
	return err
}

//Double-clicks at the current mouse coordinates (set by moveto).
func (s Session) DoubleClick() error {
	_, _, err := s.wd.do(nil, "POST", "/session/%s/doubleclick", s.Id)
	return err
}

//Single tap on the touch enabled device.
func (s Session) TouchClick(element WebElement) error {
	p := params{"element": element.id}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/click", s.Id)
	return err
}

//Finger down on the screen.
func (s Session) TouchDown(x, y int) error {
	p := params{"x": x, "y": y}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/down", s.Id)
	return err
}

//Finger up on the screen.
func (s Session) TouchUp(x, y int) error {
	p := params{"x": x, "y": y}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/up", s.Id)
	return err
}

//Finger move on the screen.
func (s Session) TouchMove(x, y int) error {
	p := params{"x": x, "y": y}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/move", s.Id)
	return err
}

//Scroll on the touch screen using finger based motion events.
func (s Session) TouchScroll(element WebElement, xoffset, yoffset int) error {
	p := params{"element": element.id, "xoffset": xoffset, "yoffset": yoffset}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/scroll", s.Id)
	return err
}

//Double tap on the touch screen using finger motion events.
func (s Session) TouchDoubleClick(element WebElement) error {
	p := params{"element": element.id}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/doubleclick", s.Id)
	return err
}

//Long press on the touch screen using finger motion events.
func (s Session) TouchLongClick(element WebElement) error {
	p := params{"element": element.id}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/longclick", s.Id)
	return err
}

//Flick on the touch screen using finger motion events.
//This flickcommand starts at a particulat screen location.
func (s Session) TouchFlick(element WebElement, xoffset, yoffset, speed int) error {
	p := params{"element": element.id, "xoffset": xoffset, "yoffset": yoffset, "speed": speed}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/flick", s.Id)
	return err
}

//Flick on the touch screen using finger motion events.
//Use this flick command if you don't care where the flick starts on the screen.
func (s Session) TouchFlickAnywhere(xspeed, yspeed int) error {
	p := params{"xspeed": xspeed, "yspeed": yspeed}
	_, _, err := s.wd.do(p, "POST", "/session/%s/touch/flick", s.Id)
	return err
}

//Get the current geo location.
func (s Session) GetGeoLocation() (GeoLocation, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/location", s.Id)
	if err != nil {
		return GeoLocation{}, err
	}
	var location GeoLocation
	err = json.Unmarshal(data, &location)
	return location, err
}

//Set the current geo location.
func (s Session) SetGeoLocation(location GeoLocation) error {
	p := params{"location": location}
	_, _, err := s.wd.do(p, "POST", "/session/%s/location", s.Id)
	return err
}

//helper functions, storageType can be "local_storage" or "session_storage"
func (s Session) storageGetKeys(storageType string) ([]string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/%s", s.Id, storageType)
	if err != nil {
		return nil, err
	}
	var keys []string
	err = json.Unmarshal(data, &keys)
	return keys, err
}

func (s Session) storageSetKey(storageType, key, value string) error {
	p := params{"key": key, "value": value}
	_, _, err := s.wd.do(p, "POST", "/session/%s/%s", s.Id, storageType)
	return err
}

func (s Session) storageClear(storageType string) error {
	_, _, err := s.wd.do(nil, "DELETE", "/session/%s/%s", s.Id, storageType)
	return err
}

//TODO protocol specification doesn't specify what is returned, I guess a string
func (s Session) storageGetKey(storageType, key string) (string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/%s/key/%s", s.Id, storageType, key)
	if err != nil {
		return "", err
	}
	var value string
	err = json.Unmarshal(data, &value)
	return value, err
}

func (s Session) storageRemoveKey(storageType string, key string) error {
	_, _, err := s.wd.do(nil, "DELETE", "/session/%s/%s/key/%s", s.Id, storageType, key)
	return err
}

//Get the number of items in the storage.
func (s Session) storageSize(storageType string) (int, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/%s/size", s.Id, storageType)
	if err != nil {
		return -1, err
	}
	var size int
	err = json.Unmarshal(data, &size)
	return size, err
}

//Get all keys of the storage.
func (s Session) LocalStorageGetKeys() ([]string, error) {
	return s.storageGetKeys("local_storage")
}

//Set the storage item for the given key.
func (s Session) LocalStorageSetKey(key, value string) error {
	return s.storageSetKey("local_storage", key, value)
}

//Clear the storage.
func (s Session) LocalStorageClear() error {
	return s.storageClear("local_storage")
}

//Get the storage item for the given key.
func (s Session) LocalStorageGetKey(key string) (string, error) {
	return s.storageGetKey("local_storage", key)
}

//Remove the storage item for the given key.
func (s Session) LocalStorageRemoveKey(key string) error {
	return s.storageRemoveKey("local_storage", key)
}

//Get the number of items in the storage.
func (s Session) LocalStorageSize() (int, error) {
	return s.storageSize("local_storage")
}

//Get all keys of the storage.
func (s Session) SessionStorageGetKeys() ([]string, error) {
	return s.storageGetKeys("session_storage")
}

//Set the storage item for the given key.
func (s Session) SessionStorageSetKey(key, value string) error {
	return s.storageSetKey("session_storage", key, value)
}

//Clear the storage.
func (s Session) SessionStorageClear() error {
	return s.storageClear("session_storage")
}

//Get the storage item for the given key.
func (s Session) SessionStorageGetKey(key string) (string, error) {
	return s.storageGetKey("session_storage", key)
}

//Remove the storage item for the given key.
func (s Session) SessionStorageRemoveKey(key string) error {
	return s.storageRemoveKey("session_storage", key)
}

//Get the number of items in the storage.
func (s Session) SessionStorageSize() (int, error) {
	return s.storageSize("session_storage")
}

//Get the log for a given log type.
func (s Session) Log(logType string) ([]LogEntry, error) {
	p := params{"type": logType}
	_, data, err := s.wd.do(p, "POST", "/session/%s/log", s.Id)
	if err != nil {
		return nil, err
	}
	var log []LogEntry
	err = json.Unmarshal(data, &log)
	return log, err
}

//Get available log types.
func (s Session) LogTypes() ([]string, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/log/types", s.Id)
	if err != nil {
		return nil, err
	}
	var logTypes []string
	err = json.Unmarshal(data, &logTypes)
	return logTypes, err
}

//Get the status of the html5 application cache.
func (s Session) GetHTML5CacheStatus() (HTML5CacheStatus, error) {
	_, data, err := s.wd.do(nil, "GET", "/session/%s/application_cache/status", s.Id)
	if err != nil {
		return 0, err
	}
	var cacheStatus HTML5CacheStatus
	err = json.Unmarshal(data, &cacheStatus)
	return cacheStatus, err
}
