//go:build windows

package main

// A small custom-drawn window used for both the picker popup and the settings
// screen. Everything is painted with GDI so it can follow the Windows light /
// dark theme and scale cleanly on high-DPI monitors.

import (
	"syscall"
	"unsafe"
)

type row struct {
	title, sub, hint string
	privateHint      bool // show "Private" while Ctrl is held
	icon             uintptr
	glyph            rune
	dim              bool
	action           func(private bool)
}

type footerItem struct {
	text     string
	glyph    rune
	checkbox bool
	checked  bool
	right    bool
	action   func(f *footerItem)
	rect     RECT
}

type theme struct {
	bg, text, sub, hover, pressed, border, box, accent, onAccent uint32
}

type win struct {
	hwnd      uintptr
	dpi       int32
	th        theme
	dark      bool
	settings  bool
	title     string
	headSmall string // picker: small label above the link
	headRight string
	headMain  string
	headSub   string // settings: status line
	headSubOK bool
	rows      []row
	rowRects  []RECT
	footer    []*footerItem
	sel, hot  int
	pressed   int
	tracking  bool

	head1, head2 RECT
	sepY         int32
	clientW      int32
	clientH      int32
	iconSize     int32

	fNormal, fTitle, fSmall, fGlyph, fGlyphSmall uintptr
	measureDC                                    uintptr
	loadIcons                                    func(size int32)
	onActivate                                   func()
}

const (
	hitNone   = -1
	hitFooter = 1000
)

var current *win

func readTheme() (theme, bool) {
	dark := false
	if v, ok := regDWORD(hkcu, `Software\Microsoft\Windows\CurrentVersion\Themes\Personalize`, "AppsUseLightTheme"); ok && v == 0 {
		dark = true
	}
	accent := rgb(0x00, 0x67, 0xC0)
	if v, ok := regDWORD(hkcu, `Software\Microsoft\Windows\DWM`, "AccentColor"); ok {
		accent = v & 0xFFFFFF // stored as AABBGGRR, same byte order as COLORREF
	}
	if dark {
		return theme{
			bg: rgb(0x20, 0x20, 0x20), text: rgb(0xFF, 0xFF, 0xFF), sub: rgb(0xA8, 0xA8, 0xA8),
			hover: rgb(0x2E, 0x2E, 0x2E), pressed: rgb(0x29, 0x29, 0x29), border: rgb(0x38, 0x38, 0x38),
			box: rgb(0x9A, 0x9A, 0x9A), accent: lighten(accent), onAccent: rgb(0, 0, 0),
		}, true
	}
	return theme{
		bg: rgb(0xF9, 0xF9, 0xF9), text: rgb(0x1B, 0x1B, 0x1B), sub: rgb(0x5F, 0x5F, 0x5F),
		hover: rgb(0xEA, 0xEA, 0xEA), pressed: rgb(0xE0, 0xE0, 0xE0), border: rgb(0xE3, 0xE3, 0xE3),
		box: rgb(0x86, 0x86, 0x86), accent: darken(accent), onAccent: rgb(0xFF, 0xFF, 0xFF),
	}, false
}

func lum(c uint32) int { return int(c&0xFF)*3 + int(c>>8&0xFF)*6 + int(c>>16&0xFF) }

// Keep the accent readable on the background.
func lighten(c uint32) uint32 {
	for i := 0; i < 6 && lum(c) < 1300; i++ {
		c = mix(c, 0xFFFFFF, 0.25)
	}
	return c
}

func darken(c uint32) uint32 {
	for i := 0; i < 6 && lum(c) > 1150; i++ {
		c = mix(c, 0, 0.2)
	}
	return c
}

func mix(a, b uint32, t float64) uint32 {
	ch := func(s uint) uint32 {
		x := float64(a>>s&0xFF)*(1-t) + float64(b>>s&0xFF)*t
		return uint32(x+0.5) << s
	}
	return ch(0) | ch(8) | ch(16)
}

func (w *win) s(v int32) int32 { return (v*w.dpi + 48) / 96 }

func (w *win) makeFonts() {
	for _, f := range []uintptr{w.fNormal, w.fTitle, w.fSmall, w.fGlyph, w.fGlyphSmall} {
		if f != 0 {
			pDeleteObject.Call(f)
		}
	}
	lf := messageFont(w.dpi)
	lf.Quality = 5
	w.fNormal = createFont(lf)
	t := lf
	t.Weight = 600
	if w.settings {
		t.Height = t.Height * 13 / 10
	}
	w.fTitle = createFont(t)
	sm := lf
	sm.Height = sm.Height * 87 / 100
	w.fSmall = createFont(sm)
	w.fGlyph = iconFont(w.s(18))
	w.fGlyphSmall = iconFont(w.s(12))
}

func (w *win) textWidth(font uintptr, s string) int32 {
	if w.measureDC == 0 {
		w.measureDC, _, _ = pCreateCompatibleDC.Call(0)
	}
	old, _, _ := pSelectObject.Call(w.measureDC, font)
	r := RECT{0, 0, 10000, 1000}
	utf, _ := syscall.UTF16FromString(s)
	pDrawTextW.Call(w.measureDC, uintptr(unsafe.Pointer(&utf[0])), ^uintptr(0), uintptr(unsafe.Pointer(&r)), DT_CALCRECT|DT_SINGLELINE|DT_NOPREFIX)
	pSelectObject.Call(w.measureDC, old)
	return r.Right - r.Left
}

func (w *win) layout() {
	s := w.s
	W := s(400)
	pad := s(16)
	y := s(12)
	if w.settings {
		w.head1 = RECT{pad, y, W - pad, y + s(28)}
		y += s(28)
		w.head2 = RECT{pad, y, W - pad, y + s(20)}
		y += s(20)
	} else {
		w.head1 = RECT{pad, y, W - pad, y + s(18)}
		y += s(19)
		w.head2 = RECT{pad, y, W - pad, y + s(22)}
		y += s(22)
	}
	y += s(10)
	rowH := s(42)
	if w.settings {
		rowH = s(54)
	}
	w.rowRects = w.rowRects[:0]
	for range w.rows {
		w.rowRects = append(w.rowRects, RECT{s(6), y, W - s(6), y + rowH})
		y += rowH + s(2)
	}
	y += s(6)
	w.sepY = y
	y += 1 + s(6)

	fh := s(30)
	right := W - s(10)
	for i := len(w.footer) - 1; i >= 0; i-- {
		f := w.footer[i]
		if !f.right {
			continue
		}
		var fw int32
		if f.text == "" {
			fw = s(32)
		} else {
			fw = w.textWidth(w.fNormal, f.text) + s(24)
		}
		f.rect = RECT{right - fw, y, right, y + fh}
		right -= fw + s(4)
	}
	left := s(10)
	for _, f := range w.footer {
		if f.right {
			continue
		}
		fw := w.textWidth(w.fNormal, f.text) + s(12)
		if f.checkbox {
			fw += s(26)
		}
		if left+fw > right-s(6) {
			fw = right - s(6) - left
		}
		f.rect = RECT{left, y, left + fw, y + fh}
		left += fw + s(4)
	}
	y += fh + s(10)
	w.clientW, w.clientH = W, y
}

func (w *win) windowStyle() (uint32, uint32) {
	style := uint32(WS_OVERLAPPED | WS_CAPTION | WS_SYSMENU)
	ex := uint32(0)
	if w.settings {
		style |= WS_MINIMIZEBOX
		ex = WS_EX_APPWINDOW
	} else {
		ex = WS_EX_TOPMOST | WS_EX_DLGMODALFRAME
	}
	return style, ex
}

func (w *win) windowSize() (int32, int32) {
	style, ex := w.windowStyle()
	r := RECT{0, 0, w.clientW, w.clientH}
	if pAdjustWindowRectExForDpi.Find() == nil {
		pAdjustWindowRectExForDpi.Call(uintptr(unsafe.Pointer(&r)), uintptr(style), 0, uintptr(ex), uintptr(w.dpi))
	} else {
		pAdjustWindowRectEx.Call(uintptr(unsafe.Pointer(&r)), uintptr(style), 0, uintptr(ex))
	}
	return r.Right - r.Left, r.Bottom - r.Top
}

func (w *win) hitTest(x, y int32) int {
	for i, r := range w.rowRects {
		if r.contains(x, y) && w.rows[i].action != nil {
			return i
		}
	}
	for i, f := range w.footer {
		if f.action != nil && f.rect.contains(x, y) {
			return hitFooter + i
		}
	}
	return hitNone
}

func (w *win) invalidate() { pInvalidateRect.Call(w.hwnd, 0, 0) }

func (w *win) activate(hit int, private bool) {
	if hit >= 0 && hit < len(w.rows) {
		if a := w.rows[hit].action; a != nil {
			a(private)
		}
	} else if hit >= hitFooter && hit-hitFooter < len(w.footer) {
		f := w.footer[hit-hitFooter]
		if f.action != nil {
			f.action(f)
		}
	}
	if w.hwnd != 0 {
		w.invalidate()
	}
}

func (w *win) close() {
	if w.hwnd != 0 {
		pDestroyWindow.Call(w.hwnd)
	}
}

// relayout recomputes the layout and resizes the window, keeping its position.
func (w *win) relayout() {
	w.layout()
	ww, wh := w.windowSize()
	pSetWindowPos.Call(w.hwnd, 0, 0, 0, uintptr(ww), uintptr(wh), SWP_NOZORDER|SWP_NOACTIVATE|0x0002 /*SWP_NOMOVE*/)
	w.invalidate()
}

// --- drawing ---------------------------------------------------------------

func fillRect(dc uintptr, r RECT, c uint32) {
	b, _, _ := pCreateSolidBrush.Call(uintptr(c))
	pFillRect.Call(dc, uintptr(unsafe.Pointer(&r)), b)
	pDeleteObject.Call(b)
}

func fillRound(dc uintptr, r RECT, rad int32, c uint32) {
	b, _, _ := pCreateSolidBrush.Call(uintptr(c))
	pen, _, _ := pGetStockObject.Call(NULL_PEN)
	ob, _, _ := pSelectObject.Call(dc, b)
	op, _, _ := pSelectObject.Call(dc, pen)
	pRoundRect.Call(dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right+1), uintptr(r.Bottom+1), uintptr(rad), uintptr(rad))
	pSelectObject.Call(dc, ob)
	pSelectObject.Call(dc, op)
	pDeleteObject.Call(b)
}

func strokeRound(dc uintptr, r RECT, rad, width int32, c uint32) {
	pen, _, _ := pCreatePen.Call(PS_SOLID, uintptr(width), uintptr(c))
	nb, _, _ := pGetStockObject.Call(NULL_BRUSH)
	op, _, _ := pSelectObject.Call(dc, pen)
	ob, _, _ := pSelectObject.Call(dc, nb)
	pRoundRect.Call(dc, uintptr(r.Left), uintptr(r.Top), uintptr(r.Right), uintptr(r.Bottom), uintptr(rad), uintptr(rad))
	pSelectObject.Call(dc, op)
	pSelectObject.Call(dc, ob)
	pDeleteObject.Call(pen)
}

func drawText(dc uintptr, s string, r RECT, font uintptr, c uint32, flags uint32) {
	if s == "" {
		return
	}
	utf, _ := syscall.UTF16FromString(s)
	old, _, _ := pSelectObject.Call(dc, font)
	pSetTextColor.Call(dc, uintptr(c))
	pDrawTextW.Call(dc, uintptr(unsafe.Pointer(&utf[0])), ^uintptr(0), uintptr(unsafe.Pointer(&r)),
		uintptr(flags|DT_SINGLELINE|DT_NOPREFIX|DT_VCENTER))
	pSelectObject.Call(dc, old)
}

func (w *win) paint() {
	var ps PAINTSTRUCT
	hdc, _, _ := pBeginPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
	var rc RECT
	pGetClientRect.Call(w.hwnd, uintptr(unsafe.Pointer(&rc)))
	mdc, _, _ := pCreateCompatibleDC.Call(hdc)
	bmp, _, _ := pCreateCompatibleBitmap.Call(hdc, uintptr(rc.Right), uintptr(rc.Bottom))
	old, _, _ := pSelectObject.Call(mdc, bmp)
	pSetBkMode.Call(mdc, TRANSPARENT)
	w.draw(mdc, rc)
	pBitBlt.Call(hdc, 0, 0, uintptr(rc.Right), uintptr(rc.Bottom), mdc, 0, 0, SRCCOPY)
	pSelectObject.Call(mdc, old)
	pDeleteObject.Call(bmp)
	pDeleteDC.Call(mdc)
	pEndPaint.Call(w.hwnd, uintptr(unsafe.Pointer(&ps)))
}

func (w *win) draw(dc uintptr, rc RECT) {
	s, th := w.s, w.th
	fillRect(dc, rc, th.bg)

	// Header
	if w.settings {
		drawText(dc, w.headMain, w.head1, w.fTitle, th.text, DT_LEFT|DT_END_ELLIPSIS)
		c := th.sub
		if w.headSubOK {
			c = th.accent
		}
		drawText(dc, w.headSub, w.head2, w.fSmall, c, DT_LEFT|DT_END_ELLIPSIS)
	} else {
		rw := int32(0)
		if w.headRight != "" {
			rw = w.textWidth(w.fSmall, w.headRight)
			drawText(dc, w.headRight, w.head1, w.fSmall, th.sub, DT_RIGHT)
		}
		l := w.head1
		l.Right -= rw + s(8)
		drawText(dc, w.headSmall, l, w.fSmall, th.sub, DT_LEFT|DT_END_ELLIPSIS)
		drawText(dc, w.headMain, w.head2, w.fNormal, th.text, DT_LEFT|DT_END_ELLIPSIS)
	}

	ctrl := keyDown(VK_CONTROL)
	for i, r := range w.rowRects {
		rw := w.rows[i]
		if i == w.sel && rw.action != nil {
			bg := th.hover
			if w.pressed == i && w.hot == i {
				bg = th.pressed
			}
			fillRound(dc, r, s(8), bg)
			bar := RECT{r.Left, r.Top + (r.Bottom-r.Top)/2 - s(8), r.Left + s(3), r.Top + (r.Bottom-r.Top)/2 + s(8)}
			fillRound(dc, bar, s(3), th.accent)
		}
		x := r.Left + s(14)
		box := RECT{x, r.Top + (r.Bottom-r.Top-w.iconSize)/2, x + w.iconSize, 0}
		box.Bottom = box.Top + w.iconSize
		if rw.icon != 0 {
			pDrawIconEx.Call(dc, uintptr(box.Left), uintptr(box.Top), rw.icon, uintptr(w.iconSize), uintptr(w.iconSize), 0, 0, DI_NORMAL)
		} else if rw.glyph != 0 {
			c := th.text
			if rw.dim {
				c = th.sub
			}
			drawText(dc, string(rw.glyph), box, w.fGlyph, c, DT_CENTER)
		}
		tx := box.Right + s(14)
		hint := rw.hint
		if ctrl && rw.privateHint {
			if hint != "" {
				hint = "Private   " + hint
			} else {
				hint = "Private"
			}
		}
		hintW := int32(0)
		if hint != "" {
			hintW = w.textWidth(w.fSmall, hint)
			hr := RECT{r.Right - s(14) - hintW, r.Top, r.Right - s(14), r.Bottom}
			hc := th.sub
			if ctrl && rw.privateHint {
				hc = th.accent
			}
			drawText(dc, hint, hr, w.fSmall, hc, DT_RIGHT)
		}
		textR := RECT{tx, r.Top, r.Right - s(14) - hintW - s(10), r.Bottom}
		tc := th.text
		if rw.dim {
			tc = th.sub
		}
		if rw.sub != "" {
			mid := r.Top + (r.Bottom-r.Top)/2
			t1 := textR
			t1.Top, t1.Bottom = mid-s(20), mid
			t2 := textR
			t2.Top, t2.Bottom = mid, mid+s(18)
			drawText(dc, rw.title, t1, w.fNormal, tc, DT_LEFT|DT_END_ELLIPSIS)
			drawText(dc, rw.sub, t2, w.fSmall, th.sub, DT_LEFT|DT_END_ELLIPSIS)
		} else {
			drawText(dc, rw.title, textR, w.fNormal, tc, DT_LEFT|DT_END_ELLIPSIS)
		}
	}

	fillRect(dc, RECT{0, w.sepY, rc.Right, w.sepY + 1}, th.border)

	for i, f := range w.footer {
		r := f.rect
		hot := w.hot == hitFooter+i
		if hot && !f.checkbox {
			bg := th.hover
			if w.pressed == w.hot {
				bg = th.pressed
			}
			fillRound(dc, r, s(6), bg)
		}
		tc := th.text
		if f.action == nil {
			tc = th.sub
		}
		if f.checkbox {
			bs := s(16)
			b := RECT{r.Left + s(6), r.Top + (r.Bottom-r.Top-bs)/2, r.Left + s(6) + bs, 0}
			b.Bottom = b.Top + bs
			if f.checked {
				fillRound(dc, b, s(5), th.accent)
				drawText(dc, "", b, w.fGlyphSmall, th.onAccent, DT_CENTER)
			} else {
				if hot {
					fillRound(dc, b, s(5), th.hover)
				}
				strokeRound(dc, b, s(5), maxi(1, s(1)), th.box)
			}
			tr := RECT{b.Right + s(8), r.Top, r.Right, r.Bottom}
			drawText(dc, f.text, tr, w.fNormal, tc, DT_LEFT|DT_END_ELLIPSIS)
		} else if f.text == "" && f.glyph != 0 {
			drawText(dc, string(f.glyph), r, w.fGlyph, tc, DT_CENTER)
		} else {
			flags := uint32(DT_CENTER)
			if f.action == nil {
				flags = DT_LEFT | DT_END_ELLIPSIS
				r.Left += s(6)
			}
			drawText(dc, f.text, r, w.fNormal, tc, flags)
		}
	}
}

func maxi(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}

// --- window procedure -------------------------------------------------------

func (w *win) moveSel(delta int) {
	n := len(w.rows)
	if n == 0 {
		return
	}
	i := w.sel
	for k := 0; k < n; k++ {
		i = ((i+delta)%n + n) % n
		if w.rows[i].action != nil {
			w.sel = i
			w.invalidate()
			return
		}
	}
}

func wndProc(hwnd, msg, wparam, lparam uintptr) uintptr {
	w := current
	if w == nil || (w.hwnd != 0 && w.hwnd != hwnd) {
		r, _, _ := pDefWindowProcW.Call(hwnd, msg, wparam, lparam)
		return r
	}
	switch msg {
	case WM_ERASEBKGND:
		return 1
	case WM_PAINT:
		w.paint()
		return 0
	case WM_MOUSEMOVE:
		if !w.tracking {
			tme := TRACKMOUSEEVENT{CbSize: uint32(unsafe.Sizeof(TRACKMOUSEEVENT{})), DwFlags: TME_LEAVE, HwndTrack: hwnd}
			pTrackMouseEvent.Call(uintptr(unsafe.Pointer(&tme)))
			w.tracking = true
		}
		h := w.hitTest(loword(lparam), hiword(lparam))
		if h != w.hot {
			w.hot = h
			if h >= 0 && h < len(w.rows) {
				w.sel = h
			}
			w.invalidate()
		}
		return 0
	case WM_MOUSELEAVE:
		w.tracking = false
		if w.hot != hitNone {
			w.hot = hitNone
			w.invalidate()
		}
		return 0
	case WM_SETCURSOR:
		if loword(lparam) == 1 && w.hot != hitNone { // HTCLIENT
			c, _, _ := pLoadCursorW.Call(0, IDC_HAND)
			pSetCursor.Call(c)
			return 1
		}
	case WM_LBUTTONDOWN:
		w.pressed = w.hitTest(loword(lparam), hiword(lparam))
		pSetCapture.Call(hwnd)
		w.invalidate()
		return 0
	case WM_LBUTTONUP:
		pReleaseCapture.Call()
		h := w.hitTest(loword(lparam), hiword(lparam))
		p := w.pressed
		w.pressed = hitNone
		w.invalidate()
		if h == p && h != hitNone {
			w.activate(h, wparam&0x0008 != 0) // MK_CONTROL
		}
		return 0
	case WM_KEYDOWN:
		ctrl := keyDown(VK_CONTROL)
		switch k := int(wparam); {
		case k == VK_ESCAPE:
			w.close()
		case k == VK_UP:
			w.moveSel(-1)
		case k == VK_DOWN:
			w.moveSel(1)
		case k == VK_TAB:
			if keyDown(VK_SHIFT) {
				w.moveSel(-1)
			} else {
				w.moveSel(1)
			}
		case k == VK_HOME:
			w.sel = -1
			w.moveSel(1)
		case k == VK_END:
			w.sel = 0
			w.moveSel(-1)
		case k == VK_RETURN || k == VK_SPACE:
			w.activate(w.sel, ctrl)
		case k >= '1' && k <= '9':
			if i := k - '1'; i < len(w.rows) && !w.settings {
				w.activate(i, ctrl)
			}
		case k >= VK_NUMPAD1 && k <= VK_NUMPAD9:
			if i := k - VK_NUMPAD1; i < len(w.rows) && !w.settings {
				w.activate(i, ctrl)
			}
		case k == 'C' && ctrl:
			for i, f := range w.footer {
				if f.text == "Copy link" {
					w.activate(hitFooter+i, false)
				}
			}
		case k == VK_CONTROL:
			w.invalidate()
		}
		return 0
	case WM_KEYUP:
		if wparam == VK_CONTROL {
			w.invalidate()
		}
		return 0
	case WM_ACTIVATE:
		if loword(wparam) != 0 && w.onActivate != nil {
			w.onActivate()
		}
		w.invalidate()
	case WM_SETTINGCHANGE, WM_DWMCOLORCHANGE, WM_THEMECHANGED:
		w.th, w.dark = readTheme()
		setDarkTitleBar(hwnd, w.dark)
		w.invalidate()
	case WM_DPICHANGED:
		w.dpi = hiword(wparam)
		w.applyDPI()
		sug := (*RECT)(unsafe.Pointer(lparam))
		ww, wh := w.windowSize()
		pSetWindowPos.Call(hwnd, 0, uintptr(sug.Left), uintptr(sug.Top), uintptr(ww), uintptr(wh), SWP_NOZORDER|SWP_NOACTIVATE)
		w.invalidate()
		return 0
	case WM_CLOSE:
		w.close()
		return 0
	case WM_DESTROY:
		w.hwnd = 0
		pPostQuitMessage.Call(0)
		return 0
	}
	r, _, _ := pDefWindowProcW.Call(hwnd, msg, wparam, lparam)
	return r
}

func (w *win) applyDPI() {
	w.makeFonts()
	w.iconSize = w.s(24)
	if w.loadIcons != nil {
		w.loadIcons(w.iconSize)
	}
	w.layout()
}

var classRegistered bool

const className = "BrowserPickerWindow"

func registerClass() {
	if classRegistered {
		return
	}
	hinst, _, _ := pGetModuleHandleW.Call(0)
	icon, _, _ := pLoadIconW.Call(hinst, 1) // icon resource #1 embedded at build time
	cursor, _, _ := pLoadCursorW.Call(0, IDC_ARROW)
	wc := WNDCLASSEXW{
		CbSize:        uint32(unsafe.Sizeof(WNDCLASSEXW{})),
		Style:         CS_HREDRAW | CS_VREDRAW,
		LpfnWndProc:   syscall.NewCallback(wndProc),
		HInstance:     hinst,
		HIcon:         icon,
		HIconSm:       icon,
		HCursor:       cursor,
		LpszClassName: u16(className),
	}
	pRegisterClassExW.Call(uintptr(unsafe.Pointer(&wc)))
	classRegistered = true
}

// run creates the window near the mouse (picker) or centred (settings) and
// runs the message loop until it closes.
func (w *win) run() {
	registerClass()
	current = w
	w.sel, w.hot, w.pressed = 0, hitNone, hitNone
	if len(w.rows) > 0 && w.rows[0].action == nil {
		w.moveSel(1)
	}
	w.th, w.dark = readTheme()
	pt, _, work, dpi := cursorMonitor()
	w.dpi = dpi
	w.applyDPI()
	ww, wh := w.windowSize()
	var x, y int32
	if w.settings {
		x = work.Left + (work.Right-work.Left-ww)/2
		y = work.Top + (work.Bottom-work.Top-wh)/2
	} else {
		x = pt.X - ww/2
		y = pt.Y - w.s(60)
	}
	m := w.s(8)
	if x+ww > work.Right-m {
		x = work.Right - m - ww
	}
	if y+wh > work.Bottom-m {
		y = work.Bottom - m - wh
	}
	if x < work.Left+m {
		x = work.Left + m
	}
	if y < work.Top+m {
		y = work.Top + m
	}
	style, ex := w.windowStyle()
	hinst, _, _ := pGetModuleHandleW.Call(0)
	hwnd, _, _ := pCreateWindowExW.Call(uintptr(ex), u16p(className), u16p(w.title), uintptr(style),
		uintptr(x), uintptr(y), uintptr(ww), uintptr(wh), 0, 0, hinst, 0)
	if hwnd == 0 {
		messageBox(0, "Could not create the window.", appName, MB_ICONERROR)
		return
	}
	w.hwnd = hwnd
	setDarkTitleBar(hwnd, w.dark)
	if d := windowDPI(hwnd); d != w.dpi {
		w.dpi = d
		w.applyDPI()
		ww, wh = w.windowSize()
		pSetWindowPos.Call(hwnd, 0, uintptr(x), uintptr(y), uintptr(ww), uintptr(wh), SWP_NOZORDER|SWP_NOACTIVATE)
	}
	pShowWindow.Call(hwnd, SW_SHOW)
	pUpdateWindow.Call(hwnd)
	forceForeground(hwnd)

	var msg MSG
	for {
		r, _, _ := pGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		pTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		pDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}
	if w.measureDC != 0 {
		pDeleteDC.Call(w.measureDC)
	}
	current = nil
}
