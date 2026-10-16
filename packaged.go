//go:build windows

package main

// Support for browsers that aren't in the classic StartMenuInternet list:
// packaged (MSIX / Microsoft Store) browsers such as Arc, plus any other app
// that registered itself as an http/https handler.
//
// Packaged apps live in the protected WindowsApps folder and can't be started
// with CreateProcess, so they are launched exactly the way Windows does it
// when they are the default browser: through their ProgID.

import (
	"image"
	"image/draw"
	_ "image/png"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var (
	ole32               = syscall.NewLazyDLL("ole32.dll")
	pCoInitializeEx     = ole32.NewProc("CoInitializeEx")
	pCoCreateInstance   = ole32.NewProc("CoCreateInstance")
	pShellExecuteExW    = shell32.NewProc("ShellExecuteExW")
	pCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	pCreateBitmap       = gdi32.NewProc("CreateBitmap")
	pCreateIconIndirect = user32.NewProc("CreateIconIndirect")
)

const (
	packageProtocolPath = `Software\Classes\Local Settings\Software\Microsoft\Windows\CurrentVersion\AppModel\PackageRepository\Extensions\windows.protocol\`
	assocToastsPath     = `Software\Microsoft\Windows\CurrentVersion\ApplicationAssociationToasts`
)

func initCOM() {
	pCoInitializeEx.Call(0, 0x2|0x4) // COINIT_APARTMENTTHREADED | COINIT_DISABLE_OLE1DDE
}

// httpsProgIDs returns every ProgID registered to handle https links.
func httpsProgIDs() []string {
	seen := map[string]bool{}
	var out []string
	add := func(p string) {
		lp := strings.ToLower(p)
		if p == "" || seen[lp] || strings.EqualFold(p, progURL) || strings.EqualFold(p, progHTML) {
			return
		}
		seen[lp] = true
		out = append(out, p)
	}
	// Packaged apps declare their protocols here (one subkey per ProgID).
	for _, root := range []syscall.Handle{hkcu, hklm} {
		for _, p := range regSubkeys(root, packageProtocolPath+"https", 0) {
			add(p)
		}
	}
	// Windows records every "<ProgID>_https" handler it has seen registered,
	// for both packaged and classic apps.
	for _, name := range regValueNames(hkcu, assocToastsPath) {
		if strings.HasSuffix(strings.ToLower(name), "_https") {
			add(name[:len(name)-len("_https")])
		}
	}
	return out
}

// findExtraBrowsers lists https handlers that findBrowsers didn't already find.
func findExtraBrowsers(known []Browser) []Browser {
	self, _ := os.Executable()
	seenExe := map[string]bool{}
	seenName := map[string]bool{}
	for _, b := range known {
		seenExe[strings.ToLower(filepath.Clean(b.Exe))] = true
		seenName[strings.ToLower(b.Name)] = true
	}
	var out []Browser
	byAUMID := map[string]int{}
	for _, b := range findPackagedBrowsers() {
		if seenName[strings.ToLower(b.Name)] {
			continue
		}
		seenName[strings.ToLower(b.Name)] = true
		byAUMID[strings.ToLower(b.AUMID)] = len(out)
		out = append(out, b)
	}
	for _, prog := range httpsProgIDs() {
		lp := strings.ToLower(prog)
		if strings.HasPrefix(lp, "ie.") || strings.HasPrefix(lp, "iexplore") {
			continue
		}
		cls := `Software\Classes\` + prog
		root, ok := classRoot(cls)
		isAppX := strings.HasPrefix(lp, "appx")
		if !ok && !isAppX {
			continue
		}
		read := func(sub, val string) string {
			if !ok {
				return ""
			}
			v, _ := regString(root, cls+sub, val, 0)
			return v
		}
		name := loadIndirectString(read(`\Application`, "ApplicationName"))
		icon := read(`\Application`, "ApplicationIcon")

		b := Browser{ID: "progid:" + prog, ProgID: prog}
		if isAppX {
			// Packaged app. Each detail has a few possible homes, so try them
			// all, finishing with the shell's own association lookup.
			b.Packaged = true
			b.AUMID = firstNonEmpty(read(`\Application`, "AppUserModelID"), read(`\Shell\open`, "AppUserModelID"), assocString(prog, assocAppID))
			if i, ok := byAUMID[strings.ToLower(b.AUMID)]; ok && b.AUMID != "" {
				out[i].ProgID = prog // same app, found via its manifest
				continue
			}
			if name == "" || strings.HasPrefix(name, "@") {
				name = loadIndirectString(assocString(prog, assocFriendlyAppName))
			}
			if (name == "" || strings.HasPrefix(name, "@")) && b.AUMID != "" {
				name = nameFromAUMID(b.AUMID)
			}
			if (name == "" || strings.HasPrefix(name, "@")) && read(`\Shell\open`, "PackageId") != "" {
				name = nameFromAUMID(read(`\Shell\open`, "PackageId"))
			}
			for _, ic := range []string{icon, assocString(prog, assocAppIconReference), read(`\DefaultIcon`, "")} {
				if ic == "" {
					continue
				}
				p, idx := parseIconRef(loadIndirectString(ic))
				if p != "" && !strings.HasPrefix(p, "@") {
					b.IconPath, b.IconIdx = p, idx
					break
				}
			}
		} else {
			// A classic app that registered for https without a
			// StartMenuInternet entry.
			cmd, ok := regString(root, cls+`\shell\open\command`, "", 0)
			if !ok {
				continue
			}
			exe := exeFromCommand(expandEnv(cmd))
			if exe == "" || strings.EqualFold(filepath.Base(exe), filepath.Base(self)) ||
				strings.EqualFold(filepath.Base(exe), "iexplore.exe") || strings.EqualFold(filepath.Base(exe), "rundll32.exe") {
				continue
			}
			if _, err := os.Stat(exe); err != nil || seenExe[strings.ToLower(filepath.Clean(exe))] {
				continue
			}
			b.Command, b.Exe, b.IconPath = cmd, exe, exe
			if icon != "" {
				b.IconPath, b.IconIdx = parseIconRef(expandEnv(icon))
			} else if di, ok := regString(root, cls+`\DefaultIcon`, "", 0); ok && di != "" {
				b.IconPath, b.IconIdx = parseIconRef(expandEnv(di))
			}
			if name == "" || strings.HasPrefix(name, "@") {
				name = strings.TrimSuffix(filepath.Base(exe), filepath.Ext(exe))
			}
			seenExe[strings.ToLower(filepath.Clean(exe))] = true
		}
		if name == "" || seenName[strings.ToLower(name)] {
			continue
		}
		b.Name = name
		seenName[strings.ToLower(name)] = true
		out = append(out, b)
	}
	return out
}

const (
	assocFriendlyAppName  = 4
	assocAppID            = 21
	assocAppIconReference = 23
)

var pAssocQueryStringW = shlwapi.NewProc("AssocQueryStringW")

// assocString asks the shell about a ProgID (works for packaged apps too).
func assocString(prog string, what uint32) string {
	if pAssocQueryStringW.Find() != nil {
		return ""
	}
	buf := make([]uint16, 2048)
	n := uint32(len(buf))
	hr, _, _ := pAssocQueryStringW.Call(0, uintptr(what), u16p(prog), u16p("open"), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&n)))
	if hr != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func firstNonEmpty(v ...string) string {
	for _, s := range v {
		if s != "" {
			return s
		}
	}
	return ""
}

// classRoot finds which hive holds a Software\Classes key.
func classRoot(path string) (syscall.Handle, bool) {
	for _, r := range []syscall.Handle{hkcu, hklm} {
		if h, ok := regOpen(r, path, keyRead); ok {
			syscall.RegCloseKey(h)
			return r, true
		}
	}
	return 0, false
}

// "TheBrowserCompany.Arc_ttt1ap7aakyb4!Arc" -> "Arc"
func nameFromAUMID(aumid string) string {
	if i := strings.LastIndexByte(aumid, '!'); i >= 0 && i+1 < len(aumid) && !strings.EqualFold(aumid[i+1:], "App") {
		return aumid[i+1:]
	}
	pkg := aumid
	if i := strings.IndexByte(pkg, '_'); i >= 0 {
		pkg = pkg[:i]
	}
	if i := strings.LastIndexByte(pkg, '.'); i >= 0 {
		pkg = pkg[i+1:]
	}
	return pkg
}

// --- launching ------------------------------------------------------------

type shellExecuteInfo struct {
	cbSize       uint32
	fMask        uint32
	hwnd         uintptr
	lpVerb       *uint16
	lpFile       *uint16
	lpParameters *uint16
	lpDirectory  *uint16
	nShow        int32
	hInstApp     uintptr
	lpIDList     uintptr
	lpClass      *uint16
	hkeyClass    uintptr
	dwHotKey     uint32
	hIcon        uintptr
	hProcess     uintptr
}

// openWithProgID opens link with a specific ProgID, the same call Windows
// makes when that app is the default browser.
func openWithProgID(prog, link string) bool {
	sei := shellExecuteInfo{
		fMask:   0x00000001 | 0x00000100 | 0x00000400, // CLASSNAME | NOASYNC | FLAG_NO_UI
		lpVerb:  u16("open"),
		lpFile:  u16(link),
		lpClass: u16(prog),
		nShow:   SW_SHOWNORMAL,
	}
	sei.cbSize = uint32(unsafe.Sizeof(sei))
	r, _, _ := pShellExecuteExW.Call(uintptr(unsafe.Pointer(&sei)))
	return r != 0
}

var (
	clsidAppActivationManager = syscall.GUID{Data1: 0x45BA127D, Data2: 0x10A8, Data3: 0x46EA, Data4: [8]byte{0x8A, 0xB7, 0x56, 0xEA, 0x90, 0x78, 0x94, 0x3C}}
	iidAppActivationManager   = syscall.GUID{Data1: 0x2e941141, Data2: 0x7f97, Data3: 0x4756, Data4: [8]byte{0xba, 0x1d, 0x9d, 0xec, 0xde, 0x89, 0x4a, 0x3d}}
)

// activateApp starts a packaged app by its AppUserModelID, passing args on
// its command line (desktop-bridge apps like Arc receive them as-is).
func activateApp(aumid, args string) bool {
	var mgr uintptr
	hr, _, _ := pCoCreateInstance.Call(uintptr(unsafe.Pointer(&clsidAppActivationManager)), 0, 0x4, /*CLSCTX_LOCAL_SERVER*/
		uintptr(unsafe.Pointer(&iidAppActivationManager)), uintptr(unsafe.Pointer(&mgr)))
	if hr != 0 || mgr == 0 {
		return false
	}
	vtbl := *(*[6]uintptr)(unsafe.Pointer(*(*uintptr)(unsafe.Pointer(mgr))))
	defer syscall.SyscallN(vtbl[2], mgr) // Release
	var pid uint32
	hr, _, _ = syscall.SyscallN(vtbl[3], mgr, u16p(aumid), u16p(args), 0, uintptr(unsafe.Pointer(&pid)))
	return hr == 0
}

func (b Browser) launchPackaged(link string, private bool) error {
	// 1. Its execution alias: a normal exe stub that takes arguments.
	if b.Exe != "" {
		if err := startProcess(b.Exe, buildCommandLine(b.Command, b.Exe, link, private), 0); err == nil {
			return nil
		}
	}
	quoted := `"` + strings.ReplaceAll(link, `"`, "%22") + `"`
	// 2. A private window needs a command-line switch, so activate directly.
	if private && b.AUMID != "" {
		if f := b.packagedPrivateFlag(); f != "" && activateApp(b.AUMID, f+" "+quoted) {
			return nil
		}
	}
	// 3. Its https ProgID: exactly what Windows does for a default browser.
	if b.ProgID != "" && openWithProgID(b.ProgID, link) {
		return nil
	}
	// 4. Activate the app with the link as its argument.
	if b.AUMID != "" && activateApp(b.AUMID, quoted) {
		return nil
	}
	return syscall.Errno(1155) // ERROR_NO_ASSOCIATION
}

func (b Browser) packagedPrivateFlag() string {
	if b.Exe != "" {
		if f := privateFlag(b.Exe); f != "" {
			return f
		}
	}
	return privateFlagByName(b.Name + " " + b.AUMID)
}

// --- icons ------------------------------------------------------------------

// pngIcon turns a packaged app's PNG logo into an HICON of the given size.
func pngIcon(path string, size int32) uintptr {
	f, err := os.Open(path)
	if err != nil {
		return 0
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		return 0
	}
	rgba := image.NewNRGBA(src.Bounds())
	draw.Draw(rgba, rgba.Bounds(), src, src.Bounds().Min, draw.Src)
	px := resizeNRGBA(rgba, int(size))

	type bitmapInfoHeader struct {
		Size          uint32
		Width, Height int32
		Planes, Bits  uint16
		Compression   uint32
		SizeImage     uint32
		XPels, YPels  int32
		ClrUsed       uint32
		ClrImportant  uint32
	}
	bi := bitmapInfoHeader{Width: size, Height: -size, Planes: 1, Bits: 32}
	bi.Size = uint32(unsafe.Sizeof(bi))
	var bits uintptr
	color, _, _ := pCreateDIBSection.Call(0, uintptr(unsafe.Pointer(&bi)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if color == 0 || bits == 0 {
		return 0
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(bits)), int(size*size*4))
	for i := 0; i < int(size*size); i++ {
		dst[i*4+0] = px[i*4+2] // B
		dst[i*4+1] = px[i*4+1] // G
		dst[i*4+2] = px[i*4+0] // R
		dst[i*4+3] = px[i*4+3] // A
	}
	mask, _, _ := pCreateBitmap.Call(uintptr(size), uintptr(size), 1, 1, 0)
	type iconInfo struct {
		FIcon    int32
		XHotspot uint32
		YHotspot uint32
		HbmMask  uintptr
		HbmColor uintptr
	}
	ii := iconInfo{FIcon: 1, HbmMask: mask, HbmColor: color}
	h, _, _ := pCreateIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	pDeleteObject.Call(color)
	pDeleteObject.Call(mask)
	return h
}

// loadIcon returns the browser's icon at the given pixel size.
func (b Browser) loadIcon(size int32) uintptr {
	if strings.EqualFold(filepath.Ext(b.IconPath), ".png") {
		if h := pngIcon(b.IconPath, size); h != 0 {
			return h
		}
	}
	if h := extractIcon(b.IconPath, b.IconIdx, size); h != 0 {
		return h
	}
	if b.Exe != "" {
		return extractIcon(b.Exe, 0, size)
	}
	return 0
}
