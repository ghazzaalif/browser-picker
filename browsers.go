//go:build windows

package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"unsafe"
)

const startMenuInternet = `SOFTWARE\Clients\StartMenuInternet`

type Browser struct {
	ID       string // registry key name under StartMenuInternet
	Name     string
	Command  string // shell\open\command
	Exe      string
	IconPath string
	IconIdx  int

	// Set for browsers found through their https ProgID (e.g. packaged
	// MSIX apps such as Arc).
	ProgID   string
	AUMID    string
	Packaged bool
}

// findBrowsers lists every browser registered with Windows (the same list
// Windows' "Default apps" uses), from both the per-user and machine-wide
// registry, including 32-bit registrations.
func findBrowsers() []Browser {
	self, _ := os.Executable()
	type src struct {
		root syscall.Handle
		view uint32
	}
	sources := []src{{hkcu, 0}, {hklm, keyWow64_64}, {hklm, keyWow64_32}}
	seenID := map[string]bool{}
	seenExe := map[string]bool{}
	var out []Browser
	for _, s := range sources {
		for _, id := range regSubkeys(s.root, startMenuInternet, s.view) {
			lid := strings.ToLower(id)
			// Skip ourselves and the leftover Internet Explorer entry (it just opens Edge).
			if lid == strings.ToLower(appKey) || lid == "iexplore.exe" || seenID[lid] {
				continue
			}
			base := startMenuInternet + `\` + id
			cmd, ok := regString(s.root, base+`\shell\open\command`, "", s.view)
			if !ok || strings.TrimSpace(cmd) == "" {
				continue
			}
			exe := exeFromCommand(expandEnv(cmd))
			if exe == "" {
				continue
			}
			if _, err := os.Stat(exe); err != nil {
				continue // stale registration
			}
			lexe := strings.ToLower(filepath.Clean(exe))
			if seenExe[lexe] || (self != "" && strings.EqualFold(filepath.Clean(self), exe)) ||
				strings.EqualFold(filepath.Base(exe), filepath.Base(self)) {
				continue
			}
			name, _ := regString(s.root, base+`\Capabilities`, "ApplicationName", s.view)
			name = loadIndirectString(name)
			if name == "" || strings.HasPrefix(name, "@") {
				name, _ = regString(s.root, base, "", s.view)
				name = loadIndirectString(name)
			}
			if name == "" || strings.HasPrefix(name, "@") {
				name = id
			}
			b := Browser{ID: id, Name: name, Command: cmd, Exe: exe, IconPath: exe}
			if ic, ok := regString(s.root, base+`\DefaultIcon`, "", s.view); ok && ic != "" {
				b.IconPath, b.IconIdx = parseIconRef(expandEnv(ic))
			}
			seenID[lid] = true
			seenExe[lexe] = true
			out = append(out, b)
		}
	}
	out = append(out, findExtraBrowsers(out)...)
	sort.SliceStable(out, func(i, j int) bool { return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name) })
	return out
}

func (b Browser) supportsPrivate() bool {
	if b.Packaged {
		return (b.Exe != "" || b.AUMID != "") && b.packagedPrivateFlag() != ""
	}
	return privateFlag(b.Exe) != ""
}

// launch starts the browser with the given URL.
func (b Browser) launch(url string, private bool) error {
	if b.Packaged {
		return b.launchPackaged(url, private)
	}
	return startProcess(b.Exe, buildCommandLine(expandEnv(b.Command), b.Exe, url, private), 0)
}

func startProcess(exe, cmdline string, flags uint32) error {
	var si syscall.StartupInfo
	si.Cb = uint32(unsafe.Sizeof(si))
	var pi syscall.ProcessInformation
	cl, _ := syscall.UTF16PtrFromString(cmdline)
	ex, _ := syscall.UTF16PtrFromString(exe)
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(exe))
	err := syscall.CreateProcess(ex, cl, nil, nil, false, flags, nil, dir, &si, &pi)
	if err != nil {
		return err
	}
	syscall.CloseHandle(pi.Thread)
	syscall.CloseHandle(pi.Process)
	return nil
}
