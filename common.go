// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
)

const (
	Success                    = 0
	NoSuchDriver               = 6
	NoSuchElement              = 7
	NoSuchFrame                = 8
	UnknownCommand             = 9
	StaleElementReference      = 10
	ElementNotVisible          = 11
	InvalidElementState        = 12
	UnknownError               = 13
	ElementIsNotSelectable     = 15
	JavaScriptError            = 17
	XPathLookupError           = 19
	Timeout                    = 21
	NoSuchWindow               = 23
	InvalidCookieDomain        = 24
	UnableToSetCookie          = 25
	UnexpectedAlertOpen        = 26
	NoAlertOpenError           = 27
	ScriptTimeout              = 28
	InvalidElementCoordinates  = 29
	IMENotAvailable            = 30
	IMEEngineActivationFailed  = 31
	InvalidSelector            = 32
	SessionNotCreatedException = 33
	MoveTargetOutOfBounds      = 34
)

var statusCodeStrings = map[int]string{
	0:  "The command executed successfully.",
	6:  "A session is either terminated or not started.",
	7:  "An element could not be located on the page using the given search parameters.",
	8:  "A request to switch to a frame could not be satisfied because the frame could not be found.",
	9:  "The requested resource could not be found, or a request was received using an HTTP method that is not supported by the mapped resource.",
	10: "An element command failed because the referenced element is no longer attached to the DOM.",
	11: "An element command could not be completed because the element is not visible on the page.",
	12: "An element command could not be completed because the element is in an invalid state (e.g. attempting to click a disabled element).",
	13: "An unknown server-side error occurred while processing the command.",
	15: "An attempt was made to select an element that cannot be selected.",
	17: "An error occurred while executing user supplied JavaScript.",
	19: "An error occurred while searching for an element by XPath.",
	21: "An operation did not complete before its timeout expired.",
	23: "A request to switch to a different window could not be satisfied because the window could not be found.",
	24: "An illegal attempt was made to set a cookie under a different domain than the current page.",
	25: "A request to set a cookie's value could not be satisfied.",
	26: "A modal dialog was open, blocking this operation.",
	27: "An attempt was made to operate on a modal dialog when one was not open.",
	28: "A script did not complete before its timeout expired.",
	29: "The coordinates provided to an interactions operation are invalid.",
	30: "IME was not available.",
	31: "An IME engine could not be started.",
	32: "Argument was an invalid selector (e.g. XPath/CSS).",
	33: "A new session could not be created.",
	34: "Target provided for a move action is out of bounds.",
}

//type StatusErrorCode int

type StackFrame struct {
	FileName   string
	ClassName  string
	MethodName string
	LineNumber int
}

type CommandError struct {
	StatusCode int
	ErrorType  string
	Message    string
	Screen     string
	Class      string
	StackTrace []StackFrame
}

func (e CommandError) Error() string {
	//TODO print Screen, Class, StackTrace
	m := e.ErrorType
	if m != "" {
		m += ": "
	}
	if e.StatusCode == -1 {
		m += "status code not specified"
	} else if str, found := statusCodeStrings[e.StatusCode]; found {
		m += str + ": " + e.Message
	} else {
		m += fmt.Sprintf("unknown status code (%d): %s", e.StatusCode, e.Message)
	}
	return m
}

//type matching the structure standard JSON object response.
type jsonResponse struct {
	RawSessionId json.RawMessage `json:"sessionId"`
	Status       int             `json:"status"`
	RawValue     json.RawMessage `json:"value"`
}

func parseError(c int, jr jsonResponse) error {
	var responseCodeError string
	switch c {
	// workaround: chromedriver could returns 200 code on errors
	case 200:
	case 400:
		responseCodeError = "400: Missing Command Parameters"
	case 404:
		responseCodeError = "404: Unknown command/Resource Not Found"
	case 405:
		responseCodeError = "405: Invalid Command Method"
	case 500:
		responseCodeError = "500: Failed Command"
	case 501:
		responseCodeError = "501: Unimplemented Command"
	default:
		responseCodeError = "Unknown error"
	}
	if jr.Status == 0 {
		return &CommandError{StatusCode: -1, ErrorType: responseCodeError}
	}
	commandError := &CommandError{StatusCode: jr.Status, ErrorType: responseCodeError}
	err := json.Unmarshal(jr.RawValue, commandError)
	if err != nil {
		// workaround: firefox could returns a string instead of a JSON object on errors
		commandError.Message = string(jr.RawValue)
	}
	return commandError
}

func isRedirect(response *http.Response) bool {
	r := response.StatusCode
	return r == 302 || r == 303
}

func newRequest(method, url string, data []byte) (*http.Request, error) {
	request, err := http.NewRequest(method, url, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	if method == "POST" {
		request.Header.Add("Content-Type", "application/json;charset=utf-8")
	}
	//TODO add png format for screenshots
	request.Header.Set("Accept", "application/json")
	request.Header.Set("Accept-charset", "utf-8")
	return request, nil
}

type WebDriverCore struct {
	url string
}

func (d *WebDriverCore) SetUrl(u *url.URL) {
	d.url = u.String()
}

func (w WebDriverCore) Start() error { return nil }
func (w WebDriverCore) Stop() error  { return nil }

func (w WebDriverCore) do(params interface{}, method, urlFormat string, urlParams ...interface{}) (string, []byte, error) {
	if method != "GET" && method != "POST" && method != "DELETE" {
		return "", nil, errors.New("invalid method: " + method)
	}
	url := w.url + fmt.Sprintf(urlFormat, urlParams...)
	return w.doInternal(params, method, url)
}

//communicate with the server.
func (w WebDriverCore) doInternal(params interface{}, method, url string) (string, []byte, error) {
	debugprint(">> " + method + " " + url)
	var jsonParams []byte
	var err error
	if method == "POST" {
		if params == nil {
			params = map[string]interface{}{}
		}
		jsonParams, err = json.Marshal(params)
		if err != nil {
			return "", nil, err
		}
	}
	request, err := newRequest(method, url, jsonParams)
	if err != nil {
		return "", nil, err
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		return "", nil, err
	}
	debugprint("StatusCode: " + strconv.Itoa(response.StatusCode))
	//http.Client doesn't follow POST redirected (/session command)
	if method == "POST" && isRedirect(response) {
		debugprint("redirected")
		url, err := response.Location()
		if err != nil {
			return "", nil, err
		}
		return w.doInternal(nil, "GET", url.String())
	}

	buf, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return "", nil, err
	}
	head := string(buf)
	if len(buf) > 1024 {
		head = fmt.Sprintf("%s ...%d more bytes", string(buf[0:1024]), len(buf)-1024)
	}
	debugprint("<< " + head)

	jr := &jsonResponse{}
	err = json.Unmarshal(buf, jr)
	if err != nil && response.StatusCode == 200 {
		return "", nil, errors.New("error: response must be a JSON object")
	}
	//if err = json.Unmarshal(buf, jr); err != nil {
	//	return "", nil, errors.New("error: response must be a JSON object: "+err.Error())
	//}
	if response.StatusCode >= 400 || jr.Status != 0 {
		return "", nil, parseError(response.StatusCode, *jr)
	}
	sessionId := string(bytes.Trim(jr.RawSessionId, "{}\""))
	return sessionId, []byte(jr.RawValue), nil
}

//Query the server's status.
func (w WebDriverCore) Status() (*Status, error) {
	_, data, err := w.do(nil, "GET", "/status")
	if err != nil {
		return nil, err
	}
	status := &Status{}
	err = json.Unmarshal(data, status)
	return status, err
}

//Create a new session.
//The server should attempt to create a session that most closely matches the desired and required capabilities. Required capabilities have higher priority than desired capabilities and must be set for the session to be created.
func (w WebDriverCore) newSession(desired, required Capabilities) (*Session, error) {
	if desired == nil {
		desired = map[string]interface{}{}
	}
	p := params{"desiredCapabilities": desired, "requiredCapabilities": required}
	sessionId, data, err := w.do(p, "POST", "/session")
	if err != nil {
		return nil, err
	}
	var capabilities Capabilities
	err = json.Unmarshal(data, &capabilities)
	return &Session{Id: sessionId, Capabilities: capabilities}, err
}

//Returns a list of the currently active sessions.
func (w WebDriverCore) sessions() ([]Session, error) {
	_, data, err := w.do(nil, "GET", "/sessions")
	if err != nil {
		return nil, err
	}
	var sessions []Session
	err = json.Unmarshal(data, &sessions)
	return sessions, err
	//return nil, errors.New("unsupported")
}
