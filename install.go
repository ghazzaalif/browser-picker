//go:build windows

package main

import (
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const (
	appName     = "Browser Picker"
	appKey      = "BrowserPicker" // key under StartMenuInternet
	progURL     = "BrowserPickerURL"
	progHTML    = "BrowserPickerHTML"
	smiPath     = `Software\Clients\StartMenuInternet\` + appKey
	classesPath = `Software\Classes\`
	regAppsPath = `Software\RegisteredApplications`
	rulesPath   = `Software\BrowserPicker\Rules`
	appRootPath = `Software\BrowserPicker`
)

func installDir() string {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		base = filepath.Join(os.Getenv("USERPROFILE"), "AppData", "Local")
	}
	return filepath.Join(base, "Programs", "BrowserPicker")
}

// installSelf copies the running exe to %LOCALAPPDATA%\Programs\BrowserPicker
// (unless it's already running from there) and returns the installed path.
func installSelf() (string, error) {
	self, err := os.Executable()
	if err != nil {
		return "", err
	}
	dir := installDir()
	target := filepath.Join(dir, "BrowserPicker.exe")
	if strings.EqualFold(filepath.Clean(self), filepath.Clean(target)) {
		return target, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	src, err := os.Open(self)
	if err != nil {
		return "", err
	}
	defer src.Close()
	tmp := target + ".new"
	dst, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(dst, src); err != nil {
		dst.Close()
		os.Remove(tmp)
		return "", err
	}
	dst.Close()
	os.Remove(target + ".old")
	if _, err := os.Stat(target); err == nil {
		// A running copy can be renamed but not overwritten.
		if err := os.Rename(target, target+".old"); err != nil {
			os.Remove(tmp)
			return "", fmt.Errorf("close any open Browser Picker windows and try again (%v)", err)
		}
	}
	if err := os.Rename(tmp, target); err != nil {
		return "", err
	}
	return target, nil
}

// register makes Windows list Browser Picker as a web browser for the current
// user (no admin rights needed). Windows doesn't allow apps to make themselves
// the default, so the user still picks it once in Settings > Default apps.
func register(exe string) error {
	q := `"` + exe + `"`
	icon := exe + ",0"
	ok := true
	set := func(path, name, val string) {
		if !regSetString(hkcu, path, name, val) {
			ok = false
		}
	}
	for _, p := range []struct{ id, desc string }{{progURL, "Browser Picker URL"}, {progHTML, "Browser Picker HTML Document"}} {
		base := classesPath + p.id
		set(base, "", p.desc)
		set(base, "FriendlyTypeName", p.desc)
		if p.id == progURL {
			set(base, "URL Protocol", "")
		}
		set(base+`\Application`, "ApplicationName", appName)
		set(base+`\Application`, "ApplicationIcon", icon)
		set(base+`\DefaultIcon`, "", icon)
		set(base+`\shell\open\command`, "", q+` "%1"`)
	}
	set(smiPath, "", appName)
	set(smiPath+`\DefaultIcon`, "", icon)
	set(smiPath+`\shell\open\command`, "", q+" --settings")
	caps := smiPath + `\Capabilities`
	set(caps, "ApplicationName", appName)
	set(caps, "ApplicationDescription", "Asks which browser to use every time you open a link.")
	set(caps, "ApplicationIcon", icon)
	set(caps+`\StartMenu`, "StartMenuInternet", appKey)
	for _, s := range []string{"http", "https"} {
		set(caps+`\URLAssociations`, s, progURL)
	}
	for _, e := range []string{".htm", ".html", ".shtml", ".xhtml"} {
		set(caps+`\FileAssociations`, e, progHTML)
	}
	set(regAppsPath, appName, caps)
	notifyAssocChanged()
	if !ok {
		return fmt.Errorf("some registry values could not be written")
	}
	return nil
}

func unregister() {
	regDeleteTree(hkcu, smiPath)
	regDeleteTree(hkcu, classesPath+progURL)
	regDeleteTree(hkcu, classesPath+progHTML)
	regDeleteValue(hkcu, regAppsPath, appName)
	regDeleteTree(hkcu, appRootPath)
	notifyAssocChanged()
}

func notifyAssocChanged() { pSHChangeNotify.Call(SHCNE_ASSOCCHANGED, 0, 0, 0) }

// registeredExe returns the exe path Browser Picker is registered with, if any.
func registeredExe() string {
	cmd, ok := regString(hkcu, classesPath+progURL+`\shell\open\command`, "", 0)
	if !ok {
		return ""
	}
	return exeFromCommand(cmd)
}

func isDefaultBrowser() bool {
	for _, s := range []string{"http", "https"} {
		p, _ := regString(hkcu, `Software\Microsoft\Windows\Shell\Associations\UrlAssociations\`+s+`\UserChoice`, "ProgId", 0)
		if !strings.EqualFold(p, progURL) {
			return false
		}
	}
	return true
}

func openDefaultAppsSettings() {
	// On Windows 11 this jumps straight to Browser Picker's page; Windows 10
	// ignores the parameter and shows the main Default apps page.
	shellOpen("ms-settings:defaultapps?registeredAppUser="+url.PathEscape(appName), "")
}

// --- remembered choices -------------------------------------------------

func rememberedBrowser(site string) string {
	if site == "" {
		return ""
	}
	v, _ := regString(hkcu, rulesPath, site, 0)
	return v
}

func remember(site, browserID string) {
	if site != "" {
		regSetString(hkcu, rulesPath, site, browserID)
	}
}

func rememberedCount() int { return len(regValues(hkcu, rulesPath)) }

func forgetAll() { regDeleteTree(hkcu, rulesPath) }
