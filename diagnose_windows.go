//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runDiagnose writes a report of how browsers were detected and opens it.
func runDiagnose() {
	var sb strings.Builder
	p := func(f string, a ...any) { fmt.Fprintf(&sb, f+"\n", a...) }
	self, _ := os.Executable()
	p("Browser Picker diagnostics")
	p("exe: %s", self)
	p("registered exe: %s   default browser: %v", registeredExe(), isDefaultBrowser())
	p("")
	p("== Browsers shown in the picker")
	for _, b := range findBrowsers() {
		p("- %s", b.Name)
		p("    id=%s packaged=%v", b.ID, b.Packaged)
		p("    exe=%s", b.Exe)
		p("    cmd=%s", b.Command)
		p("    progid=%s aumid=%s", b.ProgID, b.AUMID)
		p("    icon=%s,%d  private=%v", b.IconPath, b.IconIdx, b.supportsPrivate())
	}
	p("")
	p("== Installed packages that declare http/https")
	all := regSubkeys(hkcu, packageRepoPath, 0)
	p("(%d packages in repository)", len(all))
	for _, full := range all {
		root, _ := regString(hkcu, packageRepoPath+`\`+full, "PackageRootFolder", 0)
		if root == "" {
			root = filepath.Join(os.Getenv("ProgramFiles"), "WindowsApps", full)
		}
		data, err := os.ReadFile(filepath.Join(root, "AppxManifest.xml"))
		if err != nil {
			if strings.Contains(strings.ToLower(full), "arc") {
				p("! %s: cannot read manifest: %v", full, err)
			}
			continue
		}
		if !mightHandleWeb(data) {
			continue
		}
		m, _ := parseManifest(data)
		p("- %s  (name %s)", full, m.Name)
		for _, a := range m.Apps {
			p("    app id=%s exe=%s web=%v alias=%s display=%s logo=%s", a.ID, a.Executable, a.HandlesWeb, a.Alias, a.DisplayName, a.Logo)
		}
	}
	p("")
	p("== ProgIDs registered for https")
	for _, prog := range httpsProgIDs() {
		cls := `Software\Classes\` + prog
		_, ok := classRoot(cls)
		p("- %s  key=%v  assocName=%q  assocAppID=%q", prog, ok, assocString(prog, assocFriendlyAppName), assocString(prog, assocAppID))
	}
	out := filepath.Join(os.TempDir(), "BrowserPicker-diagnostics.txt")
	if err := os.WriteFile(out, []byte(strings.ReplaceAll(sb.String(), "\n", "\r\n")), 0o644); err != nil {
		messageBox(0, "Could not write "+out+": "+err.Error(), appName, MB_ICONERROR)
		return
	}
	shellOpen("notepad.exe", `"`+out+`"`)
}
