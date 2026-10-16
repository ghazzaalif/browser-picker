//go:build windows

// Browser Picker — set it as your default browser and every link you open
// outside a browser asks which installed browser to use.
//
//	BrowserPicker.exe                 settings window (install / uninstall)
//	BrowserPicker.exe <url>           show the picker for <url>
//	BrowserPicker.exe --register      install for the current user, no UI
//	BrowserPicker.exe --unregister    remove the registration, no UI
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

func init() { runtime.LockOSThread() }

func main() {
	if pSetProcessDpiAwarenessCt.Find() == nil {
		pSetProcessDpiAwarenessCt.Call(^uintptr(3)) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2 (-4)
	}
	initCOM()
	args := os.Args[1:]
	if len(args) == 0 {
		runSettings()
		return
	}
	switch strings.ToLower(args[0]) {
	case "--settings", "/settings":
		runSettings()
	case "--register", "/register":
		exe, err := installSelf()
		if err == nil {
			err = register(exe)
		}
		if err != nil {
			messageBox(0, "Install failed: "+err.Error(), appName, MB_ICONERROR)
			os.Exit(1)
		}
	case "--diagnose", "/diagnose":
		runDiagnose()
	case "--unregister", "/unregister":
		unregister()
	default:
		link := strings.TrimSpace(strings.Join(args, " "))
		if !acceptable(link) {
			messageBox(0, "Browser Picker can't open this:\n\n"+link, appName, MB_ICONWARNING)
			os.Exit(1)
		}
		runPicker(link)
	}
}

func acceptable(link string) bool {
	l := strings.ToLower(link)
	if strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://") || strings.HasPrefix(l, "file:") {
		return true
	}
	if link == "" || strings.HasPrefix(link, "-") || strings.HasPrefix(link, "/") {
		return false // never let an argument turn into a browser command-line switch
	}
	_, err := os.Stat(link) // local .html file
	return err == nil
}

func isWeb(link string) bool {
	l := strings.ToLower(link)
	return strings.HasPrefix(l, "http://") || strings.HasPrefix(l, "https://")
}

// --- picker ------------------------------------------------------------------

func runPicker(link string) {
	browsers := findBrowsers()
	if len(browsers) == 0 {
		messageBox(0, "No other browsers were found on this PC.", appName, MB_ICONWARNING)
		return
	}

	site := ""
	if isWeb(link) {
		site = siteKey(link)
	}
	// A remembered site opens straight away. Hold Shift while clicking the
	// link to get the picker anyway.
	if id := rememberedBrowser(site); id != "" && !asyncKeyDown(VK_SHIFT) {
		for _, b := range browsers {
			if strings.EqualFold(b.ID, id) && b.launch(link, false) == nil {
				return
			}
		}
	}

	w := &win{title: "Open link", headSmall: "Open link in", headMain: link}
	anyPrivate := false
	var rememberBox *footerItem
	for i, b := range browsers {
		b := b
		hint := ""
		if i < 9 {
			hint = strconv.Itoa(i + 1)
		}
		if b.supportsPrivate() {
			anyPrivate = true
		}
		w.rows = append(w.rows, row{title: b.Name, hint: hint, privateHint: b.supportsPrivate(), action: func(private bool) {
			if err := b.launch(link, private); err != nil {
				messageBox(w.hwnd, "Couldn't start "+b.Name+":\n"+err.Error(), appName, MB_ICONERROR)
				return
			}
			if rememberBox != nil && rememberBox.checked {
				remember(site, b.ID)
			}
			w.close()
		}})
	}
	if anyPrivate {
		w.headRight = "Ctrl+click: private window"
	}
	w.loadIcons = func(size int32) {
		for i := range w.rows {
			if w.rows[i].icon != 0 {
				pDestroyIcon.Call(w.rows[i].icon)
			}
			h := browsers[i].loadIcon(size)
			w.rows[i].icon = h
			if h == 0 {
				w.rows[i].glyph = '\uE774' // globe
			}
		}
	}

	if site != "" {
		rememberBox = &footerItem{checkbox: true, text: "Always use for " + site, action: func(f *footerItem) { f.checked = !f.checked }}
		w.footer = append(w.footer, rememberBox)
	}
	w.footer = append(w.footer,
		&footerItem{right: true, text: "Copy link", action: func(*footerItem) {
			setClipboardText(w.hwnd, link)
			w.close()
		}},
		&footerItem{right: true, glyph: '\uE713', action: func(*footerItem) { // gear
			self, _ := os.Executable()
			startProcess(self, `"`+self+`" --settings`, 0)
			w.close()
		}},
	)
	w.run()
	for _, r := range w.rows {
		if r.icon != 0 {
			pDestroyIcon.Call(r.icon)
		}
	}
}

// --- settings ------------------------------------------------------------------

func runSettings() {
	w := &win{settings: true, title: appName, headMain: appName}
	self, _ := os.Executable()

	build := func() {
		regExe := registeredExe()
		installed := regExe != ""
		def := installed && isDefaultBrowser()
		switch {
		case def:
			w.headSub, w.headSubOK = "Active: links opened outside a browser will ask where to open", true
		case installed:
			w.headSub, w.headSubOK = "Installed, but not your default browser yet", false
		default:
			w.headSub, w.headSubOK = "Not installed yet", false
		}

		w.rows = w.rows[:0]
		inPlace := installed && strings.EqualFold(filepath.Clean(regExe), filepath.Join(installDir(), "BrowserPicker.exe"))
		var setup row
		switch {
		case !inPlace:
			setup = row{glyph: '\uE896', title: "Install and set as default browser",
				sub: "Then choose Browser Picker in the Default apps page that opens"}
		case !def:
			setup = row{glyph: '\uE7AC', title: "Set as default browser",
				sub: "Opens Default apps: choose Browser Picker for HTTP and HTTPS"}
		default:
			setup = row{glyph: '\uE73E', title: "Browser Picker is your default browser",
				sub: "Open Default apps to change this"}
		}
		setup.action = func(bool) {
			if !inPlace {
				exe, err := installSelf()
				if err == nil {
					err = register(exe)
				}
				if err != nil {
					messageBox(w.hwnd, "Install failed: "+err.Error(), appName, MB_ICONERROR)
					return
				}
			}
			openDefaultAppsSettings()
		}
		w.rows = append(w.rows, setup)

		w.rows = append(w.rows, row{glyph: '\uE768', title: "Try it", sub: "Opens the picker for https://example.com", action: func(bool) {
			exe := self
			if r := registeredExe(); r != "" {
				if _, err := os.Stat(r); err == nil {
					exe = r
				}
			}
			startProcess(exe, `"`+exe+`" "https://example.com"`, 0)
		}})

		n := rememberedCount()
		forget := row{glyph: '\uE81C', title: "Forget remembered sites"}
		switch n {
		case 0:
			forget.sub, forget.dim = "None yet. Tick “Always use for…” in the picker to add one", true
		case 1:
			forget.sub = "1 site opens without asking (hold Shift on a link to ask anyway)"
		default:
			forget.sub = fmt.Sprintf("%d sites open without asking (hold Shift on a link to ask anyway)", n)
		}
		if n > 0 {
			forget.action = func(bool) {
				forgetAll()
				w.onActivate()
			}
		}
		w.rows = append(w.rows, forget)

		uninstall := row{glyph: '\uE74D', title: "Uninstall", sub: "Removes Browser Picker from this PC"}
		if installed || dirExists(installDir()) {
			uninstall.action = func(bool) { doUninstall(w, self) }
		} else {
			uninstall.dim = true
		}
		w.rows = append(w.rows, uninstall)

		if w.sel >= len(w.rows) || (w.sel >= 0 && w.rows[w.sel].action == nil) {
			w.sel = 0
			if w.rows[0].action == nil {
				w.moveSel(1)
			}
		}

		var names []string
		for _, b := range findBrowsers() {
			names = append(names, b.Name)
		}
		found := "No browsers found"
		if len(names) > 0 {
			found = "Found: " + strings.Join(names, ", ")
		}
		w.footer = []*footerItem{
			{text: found},
			{right: true, text: "Close", action: func(*footerItem) { w.close() }},
		}
	}
	w.onActivate = func() {
		build()
		if w.hwnd != 0 {
			w.relayout()
		}
	}
	build()
	w.run()
}

func doUninstall(w *win, self string) {
	if messageBox(w.hwnd, "Remove Browser Picker from this PC?\n\nWindows will ask you to choose another default browser.",
		appName, MB_YESNO|MB_ICONWARNING) != IDYES {
		return
	}
	unregister()
	dir := installDir()
	if strings.EqualFold(filepath.Dir(filepath.Clean(self)), filepath.Clean(dir)) {
		// We're running from the install folder: delete it once we've exited.
		cmd := os.Getenv("ComSpec")
		if cmd == "" {
			cmd = `C:\Windows\System32\cmd.exe`
		}
		startProcess(cmd, `"`+cmd+`" /c ping -n 3 127.0.0.1 >nul & rmdir /s /q "`+dir+`"`, 0x08000000 /*CREATE_NO_WINDOW*/)
	} else {
		os.RemoveAll(dir)
	}
	shellOpen("ms-settings:defaultapps", "")
	w.close()
}

func dirExists(p string) bool {
	st, err := os.Stat(p)
	return err == nil && st.IsDir()
}
