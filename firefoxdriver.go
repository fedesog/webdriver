// Copyright 2013 Federico Sogaro. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package webdriver

import (
	"archive/zip"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"time"
)

type FirefoxDriver struct {
	WebDriverCore
	// The port firefox webdriver listens on. This port - 1 will be used as a mutex to avoid starting multiple firefox instances listening to the same port. Default: 7055
	Port int
	// Start method fails if lock (see Port) is not acquired before LockPortTimeout. Default 60s
	LockPortTimeout time.Duration
	// Start method fails if Firefox doesn't start in less than StartTimeout. Default 20s.
	StartTimeout time.Duration
	// Log file to dump firefox stdout/stderr. If "" send to terminal. Default: ""
	LogFile string
	// Firefox preferences. Default: see method GetDefaultPrefs
	Prefs map[string]interface{}
	// If temporary profile has to be deleted when closing. Default: true
	DeleteProfileOnClose bool

	firefoxPath string
	xpiPath     string
	profilePath string
	cmd         *exec.Cmd
	logFile     *os.File
}

func NewFirefoxDriver(firefoxPath string, xpiPath string) *FirefoxDriver {
	d := &FirefoxDriver{}
	d.firefoxPath = firefoxPath
	d.xpiPath = xpiPath
	d.Port = 0
	d.LockPortTimeout = 60 * time.Second
	d.StartTimeout = 20 * time.Second
	d.LogFile = ""
	d.Prefs = GetDefaultPrefs()
	d.DeleteProfileOnClose = true
	return d
}

// Equivalent to setting the following firefox preferences to:
// "webdriver.log.file": path/jsconsole.log
// "webdriver.log.driver.file": path/driver.log
// "webdriver.log.profiler.file": path/profiler.log
// "webdriver.log.browser.file": path/browser.log
func (d *FirefoxDriver) SetLogPath(path string) {
	d.Prefs["webdriver.log.file"] = filepath.Join(path, "jsconsole.log")
	d.Prefs["webdriver.log.driver.file"] = filepath.Join(path, "driver.log")
	d.Prefs["webdriver.log.profiler.file"] = filepath.Join(path, "profiler.log")
	d.Prefs["webdriver.log.browser.file"] = filepath.Join(path, "browser.log")
}

func (d *FirefoxDriver) Start() error {
	if d.Port == 0 { //otherwise try to use that port
		d.Port = 7055
		lockPortAddress := fmt.Sprintf("127.0.0.1:%d", d.Port-1)
		now := time.Now()
		//try to lock port d.Port - 1
		for {
			if ln, err := net.Listen("tcp", lockPortAddress); err == nil {
				defer ln.Close()
				break
			}
			if time.Since(now) > d.LockPortTimeout {
				return errors.New("timeout expired trying to lock mutex port")
			}
			time.Sleep(1 * time.Second)
		}
		//find the first available port starting with d.Port
		for i := d.Port; i < 65535; i++ {
			address := fmt.Sprintf("127.0.0.1:%d", i)
			if ln, err := net.Listen("tcp", address); err == nil {
				if err = ln.Close(); err != nil {
					return err
				}
				d.Port = i
				break
			}
		}
	}
	//start firefox with custom profile
	//TODO it should be possible to use an existing profile
	d.Prefs["webdriver_firefox_port"] = d.Port
	var err error
	d.profilePath, err = createTempProfile(d.xpiPath, d.Prefs)
	if err != nil {
		return err
	}
	debugprint(d.profilePath)
	d.cmd = exec.Command(d.firefoxPath, "-no-remote", "-profile", d.profilePath)
	stdout, err := d.cmd.StdoutPipe()
	if err != nil {
		fmt.Println(err)
	}
	stderr, err := d.cmd.StderrPipe()
	if err != nil {
		fmt.Println(err)
	}
	if err := d.cmd.Start(); err != nil {
		return errors.New("unable to start firefox: " + err.Error())
	}
	if d.LogFile != "" {
		flags := os.O_WRONLY | os.O_CREATE | os.O_TRUNC
		d.logFile, err = os.OpenFile(d.LogFile, flags, 0640)
		if err != nil {
			return err
		}
		go io.Copy(d.logFile, stdout)
		go io.Copy(d.logFile, stderr)
	} else {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}
	//probe d.Port until firefox replies or StartTimeout is up
	if err = probePort(d.Port, d.StartTimeout); err != nil {
		return err
	}

	d.url = fmt.Sprintf("http://127.0.0.1:%d/hub", d.Port)
	return nil
}

// Populate a map with default firefox preferences
func GetDefaultPrefs() map[string]interface{} {
	prefs := map[string]interface{}{
		// Disable cache
		"browser.cache.disk.enable":   false,
		"browser.cache.disk.capacity": 0,
		"browser.cache.memory.enable": true,
		//Allow extensions to be installed into the profile and still work
		"extensions.autoDisableScopes": 10,
		//Disable "do you want to remember this password?"
		"signon.rememberSignons": false,
		//Disable re-asking for license agreement
		"browser.EULA.3.accepted": true,
		"browser.EULA.override":   true,
		//set blank homepage, no welcome page
		"browser.startup.homepage":                 "about:blank",
		"browser.startup.page":                     0,
		"browser.startup.homepage_override.mstone": "ignore",
		//browser mode online
		"browser.offline": false,
		// Don't ask if we want to switch default browsers
		"browser.shell.checkDefaultBrowser": false,
		//TODO configure proxy if needed ("network.proxy.type", "network.proxy.autoconfig_url"
		//enable pop-ups
		"dom.disable_open_during_load": false,
		//disable dialog for long username/password in url
		"network.http.phishy-userpass-length": 255,
		//Disable security warnings
		"security.warn_entering_secure":           false,
		"security.warn_entering_secure.show_once": false,
		"security.warn_entering_weak":             false,
		"security.warn_entering_weak.show_once":   false,
		"security.warn_leaving_secure":            false,
		"security.warn_leaving_secure.show_once":  false,
		"security.warn_submit_insecure":           false,
		"security.warn_submit_insecure.show_once": false,
		"security.warn_viewing_mixed":             false,
		"security.warn_viewing_mixed.show_once":   false,
		//Do not use NetworkManager to detect offline/online status.
		"toolkit.networkmanager.disable": true,
		//TODO disable script timeout (should be same as server timeout)
		//"dom.max_script_run_time"
		//("dom.max_chrome_script_run_time")
		// Disable various autostuff
		"app.update.auto":                           false,
		"app.update.enabled":                        false,
		"extensions.update.enabled":                 false,
		"browser.search.update":                     false,
		"extensions.blocklist.enabled":              false,
		"browser.safebrowsing.enabled":              false,
		"browser.safebrowsing.malware.enabled":      false,
		"browser.download.manager.showWhenStarting": false,
		"browser.sessionstore.resume_from_crash":    false,
		"browser.tabs.warnOnClose":                  false,
		"browser.tabs.warnOnOpen":                   false,
		"devtools.errorconsole.enabled":             true,
		"extensions.logging.enabled":                true,
		"extensions.update.notifyUser":              false,
		"network.manage-offline-status":             false,
		"offline-apps.allow_by_default":             true,
		"prompts.tab_modal.enabled":                 false,
		"security.fileuri.origin_policy":            3,
		"security.fileuri.strict_origin_policy":     false,
		"toolkit.telemetry.prompted":                2,
		"toolkit.telemetry.enabled":                 false,
		"toolkit.telemetry.rejected":                true,
		"browser.dom.window.dump.enabled":           true,
		"dom.report_all_js_exceptions":              true,
		"javascript.options.showInConsole":          true,
		"network.http.max-connections-per-server":   10,
		// Webdriver settings
		"webdriver_accept_untrusted_certs":     true,
		"webdriver_assume_untrusted_issuer":    true,
		"webdriver_enable_native_events":       false,
		"webdriver_unexpected_alert_behaviour": "dismiss",
	}
	return prefs
}

type InstallRDF struct {
	Description InstallRDFDescription
}

type InstallRDFDescription struct {
	Id string `xml:"id"`
}

func createTempProfile(xpiPath string, prefs map[string]interface{}) (string, error) {
	cpferr := "create profile failed: "
	profilePath, err := ioutil.TempDir(os.TempDir(), "webdriver")
	if err != nil {
		return "", errors.New(cpferr + err.Error())
	}
	extsPath := filepath.Join(profilePath, "extensions")
	err = os.Mkdir(extsPath, 0770)
	if err != nil {
		return "", errors.New(cpferr + err.Error())
	}
	zr, err := zip.OpenReader(xpiPath)
	if err != nil {
		return "", errors.New(cpferr + err.Error())
	}
	defer zr.Close()
	var extName string
	for _, f := range zr.File {
		if f.Name == "install.rdf" {
			rc, err := f.Open()
			if err != nil {
				return "", errors.New(cpferr + err.Error())
			}
			buf, err := ioutil.ReadAll(rc)
			if err != nil {
				return "", errors.New(cpferr + err.Error())
			}
			rc.Close()
			installRDF := InstallRDF{}
			err = xml.Unmarshal(buf, &installRDF)
			if err != nil {
				return "", errors.New(cpferr + err.Error())
			}
			if installRDF.Description.Id == "" {
				return "", errors.New(cpferr + "unable to find extension Id from install.rdf")
			}
			extName = installRDF.Description.Id
			break
		}
	}
	extPath := filepath.Join(extsPath, extName)
	err = os.Mkdir(extPath, 0770)
	if err != nil {
		return "", errors.New(cpferr + err.Error())
	}
	for _, f := range zr.File {
		if err = writeExtensionFile(f, extPath); err != nil {
			return "", err
		}
	}
	fuserName := filepath.Join(profilePath, "user.js")
	fuser, err := os.OpenFile(fuserName, os.O_WRONLY|os.O_CREATE, 0600)
	if err != nil {
		return "", errors.New(cpferr + err.Error())
	}
	defer fuser.Close()
	for k, i := range prefs {
		fuser.WriteString("user_pref(\"" + k + "\", ")
		switch x := i.(type) {
		case bool:
			if x {
				fuser.WriteString("true")
			} else {
				fuser.WriteString("false")
			}
		case int:
			fuser.WriteString(strconv.Itoa(x))
		case string:
			fuser.WriteString("\"" + x + "\"")
		default:
			return "", errors.New(cpferr + "unexpected preference type: " + k)
		}
		fuser.WriteString(");\n")
	}
	return profilePath, nil
}

func writeExtensionFile(f *zip.File, extPath string) error {
	weferr := "write extension failed: "
	rc, err := f.Open()
	if err != nil {
		return errors.New(weferr + err.Error())
	}
	defer rc.Close()
	filename := filepath.Join(extPath, f.Name)
	if f.FileInfo().IsDir() {
		err = os.Mkdir(filename, 0770)
		if err != nil {
			return err
		}
	} else {
		dst, err := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return errors.New(weferr + err.Error())
		}
		defer dst.Close()
		_, err = io.Copy(dst, rc)
		if err != nil {
			return errors.New(weferr + err.Error())
		}
	}
	return nil
}

func (d *FirefoxDriver) Stop() error {
	if d.cmd == nil {
		return errors.New("stop failed: firefoxdriver not running")
	}
	defer func() {
		d.cmd = nil
	}()
	d.cmd.Process.Signal(os.Interrupt)
	if d.logFile != nil {
		d.logFile.Close()
	}
	if d.DeleteProfileOnClose {
		os.RemoveAll(d.profilePath)
	}
	return nil
}

func (d *FirefoxDriver) NewSession(desired, required Capabilities) (*Session, error) {
	session, err := d.newSession(desired, required)
	if err != nil {
		return nil, err
	}
	session.wd = d
	return session, nil
}

func (d *FirefoxDriver) Sessions() ([]Session, error) {
	sessions, err := d.sessions()
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		sessions[i].wd = d
	}
	return sessions, nil
}

/*func (d *FirefoxDriver) NewSession(desired, required Capabilities) (*Session, error) {
	id, capabs, err := d.newSession(desired, required)
	return &Session{id, capabs, d}, err
}*/
