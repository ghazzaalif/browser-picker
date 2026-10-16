package main

// Platform-independent helpers (unit-tested on any OS).

import (
	"image"
	"net/url"
	"strconv"
	"strings"
)

func exeFromCommand(cmd string) string {
	cmd = strings.TrimSpace(cmd)
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			return cmd[1 : end+1]
		}
		return ""
	}
	if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		return cmd[:i+4]
	}
	if i := strings.IndexByte(cmd, ' '); i >= 0 {
		return cmd[:i]
	}
	return cmd
}

func parseIconRef(s string) (string, int) {
	s = strings.TrimSpace(s)
	idx := 0
	if i := strings.LastIndexByte(s, ','); i > 0 {
		if n, err := strconv.Atoi(strings.TrimSpace(s[i+1:])); err == nil {
			idx = n
			s = s[:i]
		}
	}
	return strings.Trim(strings.TrimSpace(s), `"`), idx
}

// privateFlag returns the command-line switch that opens a private window,
// or "" if the browser isn't one we know.
func privateFlag(exe string) string {
	switch strings.ToLower(winBase(exe)) {
	case "msedge.exe":
		return "--inprivate"
	case "chrome.exe", "arc.exe", "brave.exe", "vivaldi.exe", "chromium.exe", "thorium.exe", "cent.exe", "iridium.exe", "supermium.exe":
		return "--incognito"
	case "opera.exe", "launcher.exe": // Opera and Opera GX register launcher.exe
		if strings.Contains(strings.ToLower(exe), "opera") {
			return "--private"
		}
	case "firefox.exe", "librewolf.exe", "waterfox.exe", "floorp.exe", "zen.exe", "mercury.exe", "palemoon.exe":
		return "-private-window"
	}
	return ""
}

// buildCommandLine turns a registered browser command such as
//
//	"C:\Program Files\Mozilla Firefox\firefox.exe"
//
// into the full command line that opens link (optionally in a private window).
func buildCommandLine(cmd, exe, link string, private bool) string {
	cmd = strings.TrimSpace(cmd)
	exePart, rest := cmd, ""
	if strings.HasPrefix(cmd, `"`) {
		if end := strings.Index(cmd[1:], `"`); end >= 0 {
			exePart, rest = cmd[:end+2], cmd[end+2:]
		}
	} else if i := strings.Index(strings.ToLower(cmd), ".exe"); i >= 0 {
		exePart, rest = `"`+cmd[:i+4]+`"`, cmd[i+4:]
	}
	if private {
		if f := privateFlag(exe); f != "" {
			rest = " " + f + rest
		}
	}
	quoted := `"` + strings.ReplaceAll(link, `"`, "%22") + `"`
	if strings.Contains(rest, "%1") {
		return exePart + strings.ReplaceAll(strings.ReplaceAll(rest, `"%1"`, "%1"), "%1", quoted)
	}
	return exePart + rest + " " + quoted
}

// siteKey is the host a remembered choice applies to ("www." ignored).
func siteKey(raw string) string {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return ""
	}
	return strings.TrimPrefix(strings.ToLower(u.Hostname()), "www.")
}

// winBase is filepath.Base for Windows paths, independent of the build OS.
func winBase(p string) string {
	if i := strings.LastIndexAny(p, `\/`); i >= 0 {
		return p[i+1:]
	}
	return p
}

// privateFlagByName guesses the private-window switch for a packaged browser
// from its display name / AppUserModelID.
func privateFlagByName(s string) string {
	s = strings.ToLower(s)
	switch {
	case strings.Contains(s, "firefox"), strings.Contains(s, "librewolf"), strings.Contains(s, "zen"):
		return "-private-window"
	case strings.Contains(s, "msedge"), strings.Contains(s, "microsoft edge"):
		return "--inprivate"
	case strings.Contains(s, "opera"):
		return "--private"
	case strings.Contains(s, "thebrowsercompany"), strings.Contains(s, "chrome"), strings.Contains(s, "brave"), strings.Contains(s, "vivaldi"):
		return "--incognito"
	}
	return ""
}

// resizeNRGBA scales a straight-alpha image to size x size (aspect kept,
// centred) using area averaging, and returns RGBA bytes.
func resizeNRGBA(src *image.NRGBA, size int) []byte {
	out := make([]byte, size*size*4)
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	if sw == 0 || sh == 0 || size <= 0 {
		return out
	}
	scale := float64(size) / float64(sw)
	if s2 := float64(size) / float64(sh); s2 < scale {
		scale = s2
	}
	dw, dh := float64(sw)*scale, float64(sh)*scale
	ox, oy := (float64(size)-dw)/2, (float64(size)-dh)/2
	const ss = 4 // supersamples per axis
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, bl, a float64
			for j := 0; j < ss; j++ {
				for i := 0; i < ss; i++ {
					fx := (float64(x) + (float64(i)+0.5)/ss - ox) / scale
					fy := (float64(y) + (float64(j)+0.5)/ss - oy) / scale
					if fx < 0 || fy < 0 || fx >= float64(sw) || fy >= float64(sh) {
						continue
					}
					c := src.NRGBAAt(b.Min.X+int(fx), b.Min.Y+int(fy))
					ca := float64(c.A)
					r += float64(c.R) * ca
					g += float64(c.G) * ca
					bl += float64(c.B) * ca
					a += ca
				}
			}
			o := (y*size + x) * 4
			if a > 0 {
				out[o] = byte(r/a + 0.5)
				out[o+1] = byte(g/a + 0.5)
				out[o+2] = byte(bl/a + 0.5)
				out[o+3] = byte(a/(ss*ss) + 0.5)
			}
		}
	}
	return out
}
