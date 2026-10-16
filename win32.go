//go:build windows

package main

// Thin Win32 bindings built on the standard library's syscall package only,
// so the project has no third-party dependencies.

import (
	"syscall"
	"unsafe"
)

var (
	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shell32  = syscall.NewLazyDLL("shell32.dll")
	shlwapi  = syscall.NewLazyDLL("shlwapi.dll")
	advapi32 = syscall.NewLazyDLL("advapi32.dll")
	dwmapi   = syscall.NewLazyDLL("dwmapi.dll")
	shcore   = syscall.NewLazyDLL("shcore.dll")

	pRegisterClassExW         = user32.NewProc("RegisterClassExW")
	pCreateWindowExW          = user32.NewProc("CreateWindowExW")
	pDefWindowProcW           = user32.NewProc("DefWindowProcW")
	pDestroyWindow            = user32.NewProc("DestroyWindow")
	pShowWindow               = user32.NewProc("ShowWindow")
	pUpdateWindow             = user32.NewProc("UpdateWindow")
	pGetMessageW              = user32.NewProc("GetMessageW")
	pTranslateMessage         = user32.NewProc("TranslateMessage")
	pDispatchMessageW         = user32.NewProc("DispatchMessageW")
	pPostQuitMessage          = user32.NewProc("PostQuitMessage")
	pPostMessageW             = user32.NewProc("PostMessageW")
	pLoadCursorW              = user32.NewProc("LoadCursorW")
	pSetCursor                = user32.NewProc("SetCursor")
	pLoadIconW                = user32.NewProc("LoadIconW")
	pInvalidateRect           = user32.NewProc("InvalidateRect")
	pBeginPaint               = user32.NewProc("BeginPaint")
	pEndPaint                 = user32.NewProc("EndPaint")
	pGetClientRect            = user32.NewProc("GetClientRect")
	pFillRect                 = user32.NewProc("FillRect")
	pDrawTextW                = user32.NewProc("DrawTextW")
	pDrawIconEx               = user32.NewProc("DrawIconEx")
	pDestroyIcon              = user32.NewProc("DestroyIcon")
	pGetCursorPos             = user32.NewProc("GetCursorPos")
	pMonitorFromPoint         = user32.NewProc("MonitorFromPoint")
	pGetMonitorInfoW          = user32.NewProc("GetMonitorInfoW")
	pSetWindowPos             = user32.NewProc("SetWindowPos")
	pGetDpiForWindow          = user32.NewProc("GetDpiForWindow")
	pAdjustWindowRectEx       = user32.NewProc("AdjustWindowRectEx")
	pAdjustWindowRectExForDpi = user32.NewProc("AdjustWindowRectExForDpi")
	pSystemParametersInfoW    = user32.NewProc("SystemParametersInfoW")
	pSystemParamsInfoForDpi   = user32.NewProc("SystemParametersInfoForDpi")
	pSetForegroundWindow      = user32.NewProc("SetForegroundWindow")
	pGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	pGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")
	pAttachThreadInput        = user32.NewProc("AttachThreadInput")
	pBringWindowToTop         = user32.NewProc("BringWindowToTop")
	pSetFocus                 = user32.NewProc("SetFocus")
	pGetKeyState              = user32.NewProc("GetKeyState")
	pGetAsyncKeyState         = user32.NewProc("GetAsyncKeyState")
	pTrackMouseEvent          = user32.NewProc("TrackMouseEvent")
	pSetCapture               = user32.NewProc("SetCapture")
	pReleaseCapture           = user32.NewProc("ReleaseCapture")
	pMessageBoxW              = user32.NewProc("MessageBoxW")
	pOpenClipboard            = user32.NewProc("OpenClipboard")
	pCloseClipboard           = user32.NewProc("CloseClipboard")
	pEmptyClipboard           = user32.NewProc("EmptyClipboard")
	pSetClipboardData         = user32.NewProc("SetClipboardData")
	pPrivateExtractIconsW     = user32.NewProc("PrivateExtractIconsW")
	pSetProcessDpiAwarenessCt = user32.NewProc("SetProcessDpiAwarenessContext")

	pCreateCompatibleDC     = gdi32.NewProc("CreateCompatibleDC")
	pCreateCompatibleBitmap = gdi32.NewProc("CreateCompatibleBitmap")
	pSelectObject           = gdi32.NewProc("SelectObject")
	pDeleteObject           = gdi32.NewProc("DeleteObject")
	pDeleteDC               = gdi32.NewProc("DeleteDC")
	pBitBlt                 = gdi32.NewProc("BitBlt")
	pCreateSolidBrush       = gdi32.NewProc("CreateSolidBrush")
	pCreatePen              = gdi32.NewProc("CreatePen")
	pGetStockObject         = gdi32.NewProc("GetStockObject")
	pRoundRect              = gdi32.NewProc("RoundRect")
	pSetBkMode              = gdi32.NewProc("SetBkMode")
	pSetTextColor           = gdi32.NewProc("SetTextColor")
	pCreateFontIndirectW    = gdi32.NewProc("CreateFontIndirectW")
	pGetDeviceCaps          = gdi32.NewProc("GetDeviceCaps")

	pGetModuleHandleW          = kernel32.NewProc("GetModuleHandleW")
	pGlobalAlloc               = kernel32.NewProc("GlobalAlloc")
	pGlobalLock                = kernel32.NewProc("GlobalLock")
	pGlobalUnlock              = kernel32.NewProc("GlobalUnlock")
	pGlobalFree                = kernel32.NewProc("GlobalFree")
	pExpandEnvironmentStringsW = kernel32.NewProc("ExpandEnvironmentStringsW")
	pGetCurrentThreadId        = kernel32.NewProc("GetCurrentThreadId")
	pShellExecuteW             = shell32.NewProc("ShellExecuteW")
	pSHChangeNotify            = shell32.NewProc("SHChangeNotify")
	pSHLoadIndirectString      = shlwapi.NewProc("SHLoadIndirectString")
	pRegCreateKeyExW           = advapi32.NewProc("RegCreateKeyExW")
	pRegSetValueExW            = advapi32.NewProc("RegSetValueExW")
	pRegDeleteTreeW            = advapi32.NewProc("RegDeleteTreeW")
	pRegDeleteValueW           = advapi32.NewProc("RegDeleteValueW")
	pRegEnumValueW             = advapi32.NewProc("RegEnumValueW")
	pDwmSetWindowAttribute     = dwmapi.NewProc("DwmSetWindowAttribute")
	pGetDpiForMonitor          = shcore.NewProc("GetDpiForMonitor")
)

const (
	WS_OVERLAPPED       = 0x00000000
	WS_CAPTION          = 0x00C00000
	WS_SYSMENU          = 0x00080000
	WS_MINIMIZEBOX      = 0x00020000
	WS_EX_TOPMOST       = 0x00000008
	WS_EX_DLGMODALFRAME = 0x00000001
	WS_EX_APPWINDOW     = 0x00040000

	CS_HREDRAW = 0x0002
	CS_VREDRAW = 0x0001

	SW_SHOW       = 5
	SW_SHOWNORMAL = 1

	WM_CREATE         = 0x0001
	WM_DESTROY        = 0x0002
	WM_ACTIVATE       = 0x0006
	WM_SETFOCUS       = 0x0007
	WM_PAINT          = 0x000F
	WM_CLOSE          = 0x0010
	WM_ERASEBKGND     = 0x0014
	WM_SETTINGCHANGE  = 0x001A
	WM_SETCURSOR      = 0x0020
	WM_KEYDOWN        = 0x0100
	WM_KEYUP          = 0x0101
	WM_SYSKEYDOWN     = 0x0104
	WM_SYSKEYUP       = 0x0105
	WM_MOUSEMOVE      = 0x0200
	WM_LBUTTONDOWN    = 0x0201
	WM_LBUTTONUP      = 0x0202
	WM_MOUSELEAVE     = 0x02A3
	WM_DPICHANGED     = 0x02E0
	WM_THEMECHANGED   = 0x031A
	WM_APP            = 0x8000
	WM_APP_REFRESH    = WM_APP + 1
	WM_DWMCOLORCHANGE = 0x0320

	VK_TAB     = 0x09
	VK_RETURN  = 0x0D
	VK_SHIFT   = 0x10
	VK_CONTROL = 0x11
	VK_ESCAPE  = 0x1B
	VK_SPACE   = 0x20
	VK_END     = 0x23
	VK_HOME    = 0x24
	VK_UP      = 0x26
	VK_DOWN    = 0x28
	VK_NUMPAD1 = 0x61
	VK_NUMPAD9 = 0x69

	IDC_ARROW = 32512
	IDC_HAND  = 32649

	DT_LEFT         = 0x0000
	DT_CENTER       = 0x0001
	DT_RIGHT        = 0x0002
	DT_VCENTER      = 0x0004
	DT_SINGLELINE   = 0x0020
	DT_NOPREFIX     = 0x0800
	DT_END_ELLIPSIS = 0x8000
	DT_CALCRECT     = 0x0400

	DI_NORMAL = 0x0003

	SRCCOPY     = 0x00CC0020
	TRANSPARENT = 1
	NULL_PEN    = 8
	NULL_BRUSH  = 5
	PS_SOLID    = 0
	LOGPIXELSY  = 90

	MONITOR_DEFAULTTONEAREST = 2

	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010

	SPI_GETNONCLIENTMETRICS = 0x0029

	TME_LEAVE = 0x00000002

	MB_OK              = 0x00000000
	MB_YESNO           = 0x00000004
	MB_ICONERROR       = 0x00000010
	MB_ICONINFORMATION = 0x00000040
	MB_ICONWARNING     = 0x00000030
	MB_TOPMOST         = 0x00040000
	MB_SETFOREGROUND   = 0x00010000
	IDYES              = 6

	CF_UNICODETEXT = 13
	GMEM_MOVEABLE  = 0x0002

	SHCNE_ASSOCCHANGED = 0x08000000

	DWMWA_USE_IMMERSIVE_DARK_MODE_OLD = 19
	DWMWA_USE_IMMERSIVE_DARK_MODE     = 20
	DWMWA_WINDOW_CORNER_PREFERENCE    = 33
)

type POINT struct{ X, Y int32 }
type RECT struct{ Left, Top, Right, Bottom int32 }

func (r RECT) contains(x, y int32) bool {
	return x >= r.Left && x < r.Right && y >= r.Top && y < r.Bottom
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
	private uint32
}

type WNDCLASSEXW struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type MONITORINFO struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
}

type LOGFONTW struct {
	Height         int32
	Width          int32
	Escapement     int32
	Orientation    int32
	Weight         int32
	Italic         byte
	Underline      byte
	StrikeOut      byte
	CharSet        byte
	OutPrecision   byte
	ClipPrecision  byte
	Quality        byte
	PitchAndFamily byte
	FaceName       [32]uint16
}

type NONCLIENTMETRICSW struct {
	CbSize           uint32
	BorderWidth      int32
	ScrollWidth      int32
	ScrollHeight     int32
	CaptionWidth     int32
	CaptionHeight    int32
	CaptionFont      LOGFONTW
	SmCaptionWidth   int32
	SmCaptionHeight  int32
	SmCaptionFont    LOGFONTW
	MenuWidth        int32
	MenuHeight       int32
	MenuFont         LOGFONTW
	StatusFont       LOGFONTW
	MessageFont      LOGFONTW
	PaddedBorderWidt int32
}

type TRACKMOUSEEVENT struct {
	CbSize      uint32
	DwFlags     uint32
	HwndTrack   uintptr
	DwHoverTime uint32
}

func u16(s string) *uint16 {
	p, _ := syscall.UTF16PtrFromString(s)
	return p
}

func u16p(s string) uintptr { return uintptr(unsafe.Pointer(u16(s))) }

func rgb(r, g, b byte) uint32 { return uint32(r) | uint32(g)<<8 | uint32(b)<<16 }

func loword(v uintptr) int32 { return int32(int16(v & 0xFFFF)) }
func hiword(v uintptr) int32 { return int32(int16((v >> 16) & 0xFFFF)) }

func messageBox(hwnd uintptr, text, title string, flags uint32) int {
	r, _, _ := pMessageBoxW.Call(hwnd, u16p(text), u16p(title), uintptr(flags|MB_TOPMOST|MB_SETFOREGROUND))
	return int(r)
}

func shellOpen(target, args string) {
	var a uintptr
	if args != "" {
		a = u16p(args)
	}
	pShellExecuteW.Call(0, u16p("open"), u16p(target), a, 0, SW_SHOWNORMAL)
}

func keyDown(vk int) bool {
	r, _, _ := pGetKeyState.Call(uintptr(vk))
	return int16(r) < 0
}

func asyncKeyDown(vk int) bool {
	r, _, _ := pGetAsyncKeyState.Call(uintptr(vk))
	return int16(r) < 0
}

func expandEnv(s string) string {
	buf := make([]uint16, 32768)
	n, _, _ := pExpandEnvironmentStringsW.Call(u16p(s), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if n == 0 || int(n) > len(buf) {
		return s
	}
	return syscall.UTF16ToString(buf[:n])
}

// loadIndirectString resolves strings like "@C:\path\app.dll,-123".
func loadIndirectString(s string) string {
	if len(s) == 0 || s[0] != '@' {
		return s
	}
	if pSHLoadIndirectString.Find() != nil {
		return s
	}
	buf := make([]uint16, 512)
	hr, _, _ := pSHLoadIndirectString.Call(u16p(s), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)), 0)
	if hr != 0 {
		return ""
	}
	return syscall.UTF16ToString(buf)
}

func extractIcon(path string, index int, size int32) uintptr {
	if path == "" || pPrivateExtractIconsW.Find() != nil {
		return 0
	}
	var h uintptr
	var id uint32
	n, _, _ := pPrivateExtractIconsW.Call(u16p(path), uintptr(int32(index)), uintptr(size), uintptr(size),
		uintptr(unsafe.Pointer(&h)), uintptr(unsafe.Pointer(&id)), 1, 0)
	if n == 0 || n == 0xFFFFFFFF {
		return 0
	}
	return h
}

func setClipboardText(hwnd uintptr, s string) bool {
	utf, err := syscall.UTF16FromString(s)
	if err != nil {
		return false
	}
	if r, _, _ := pOpenClipboard.Call(hwnd); r == 0 {
		return false
	}
	defer pCloseClipboard.Call()
	pEmptyClipboard.Call()
	size := uintptr(len(utf) * 2)
	h, _, _ := pGlobalAlloc.Call(GMEM_MOVEABLE, size)
	if h == 0 {
		return false
	}
	p, _, _ := pGlobalLock.Call(h)
	if p == 0 {
		pGlobalFree.Call(h)
		return false
	}
	copy(unsafe.Slice((*uint16)(unsafe.Pointer(p)), len(utf)), utf)
	pGlobalUnlock.Call(h)
	if r, _, _ := pSetClipboardData.Call(CF_UNICODETEXT, h); r == 0 {
		pGlobalFree.Call(h)
		return false
	}
	return true
}

func cursorMonitor() (pt POINT, mon uintptr, work RECT, dpi int32) {
	pGetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	mon, _, _ = pMonitorFromPoint.Call(uintptr(uint32(pt.X))|uintptr(uint32(pt.Y))<<32, MONITOR_DEFAULTTONEAREST)
	mi := MONITORINFO{CbSize: uint32(unsafe.Sizeof(MONITORINFO{}))}
	pGetMonitorInfoW.Call(mon, uintptr(unsafe.Pointer(&mi)))
	work = mi.RcWork
	dpi = 96
	if pGetDpiForMonitor.Find() == nil {
		var dx, dy uint32
		if hr, _, _ := pGetDpiForMonitor.Call(mon, 0, uintptr(unsafe.Pointer(&dx)), uintptr(unsafe.Pointer(&dy))); hr == 0 && dy > 0 {
			dpi = int32(dy)
		}
	}
	return
}

func windowDPI(hwnd uintptr) int32 {
	if pGetDpiForWindow.Find() == nil {
		if d, _, _ := pGetDpiForWindow.Call(hwnd); d > 0 {
			return int32(d)
		}
	}
	return 96
}

func messageFont(dpi int32) LOGFONTW {
	var ncm NONCLIENTMETRICSW
	ncm.CbSize = uint32(unsafe.Sizeof(ncm))
	ok := false
	if pSystemParamsInfoForDpi.Find() == nil {
		r, _, _ := pSystemParamsInfoForDpi.Call(SPI_GETNONCLIENTMETRICS, uintptr(ncm.CbSize), uintptr(unsafe.Pointer(&ncm)), 0, uintptr(dpi))
		ok = r != 0
	}
	if !ok {
		pSystemParametersInfoW.Call(SPI_GETNONCLIENTMETRICS, uintptr(ncm.CbSize), uintptr(unsafe.Pointer(&ncm)), 0)
		if ncm.MessageFont.Height != 0 {
			ncm.MessageFont.Height = ncm.MessageFont.Height * dpi / 96
		}
	}
	lf := ncm.MessageFont
	if lf.Height == 0 {
		lf.Height = -12 * dpi / 72
		copy(lf.FaceName[:], syscall.StringToUTF16("Segoe UI"))
	}
	return lf
}

func createFont(lf LOGFONTW) uintptr {
	h, _, _ := pCreateFontIndirectW.Call(uintptr(unsafe.Pointer(&lf)))
	return h
}

// iconFont returns the Windows symbol font. "Segoe MDL2 Assets" ships with
// both Windows 10 and 11 and uses the same code points as Fluent Icons.
func iconFont(px int32) uintptr {
	var lf LOGFONTW
	lf.Height = -px
	lf.Weight = 400
	lf.Quality = 5 // CLEARTYPE_QUALITY
	copy(lf.FaceName[:], syscall.StringToUTF16("Segoe MDL2 Assets"))
	return createFont(lf)
}

func setDarkTitleBar(hwnd uintptr, dark bool) {
	if pDwmSetWindowAttribute.Find() != nil {
		return
	}
	var v int32
	if dark {
		v = 1
	}
	if hr, _, _ := pDwmSetWindowAttribute.Call(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE, uintptr(unsafe.Pointer(&v)), 4); hr != 0 {
		pDwmSetWindowAttribute.Call(hwnd, DWMWA_USE_IMMERSIVE_DARK_MODE_OLD, uintptr(unsafe.Pointer(&v)), 4)
	}
}

// forceForeground tries hard to bring hwnd to the front. Windows normally lets
// a process started by the shell take the foreground, but some apps launch
// links in ways that don't grant it.
func forceForeground(hwnd uintptr) {
	pSetForegroundWindow.Call(hwnd)
	if fg, _, _ := pGetForegroundWindow.Call(); fg == hwnd {
		return
	}
	fg, _, _ := pGetForegroundWindow.Call()
	fgThread, _, _ := pGetWindowThreadProcessId.Call(fg, 0)
	me, _, _ := pGetCurrentThreadId.Call()
	if fgThread != 0 && fgThread != me {
		pAttachThreadInput.Call(me, fgThread, 1)
		pBringWindowToTop.Call(hwnd)
		pSetForegroundWindow.Call(hwnd)
		pSetFocus.Call(hwnd)
		pAttachThreadInput.Call(me, fgThread, 0)
	}
}
