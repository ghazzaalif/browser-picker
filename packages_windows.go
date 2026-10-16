//go:build windows

package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const packageRepoPath = `Software\Classes\Local Settings\Software\Microsoft\Windows\CurrentVersion\AppModel\Repository\Packages`

// findPackagedBrowsers reads the manifest of every package installed for
// this user and returns the apps that declare they open http/https links.
func findPackagedBrowsers() []Browser {
	var out []Browser
	for _, full := range regSubkeys(hkcu, packageRepoPath, 0) {
		root, _ := regString(hkcu, packageRepoPath+`\`+full, "PackageRootFolder", 0)
		if root == "" {
			root = filepath.Join(os.Getenv("ProgramFiles"), "WindowsApps", full)
		}
		data, err := os.ReadFile(filepath.Join(root, "AppxManifest.xml"))
		if err != nil || !mightHandleWeb(data) {
			continue
		}
		m, _ := parseManifest(data)
		family := familyName(full)
		for _, a := range m.Apps {
			if !a.HandlesWeb || a.ID == "" {
				continue
			}
			b := Browser{Packaged: true, AUMID: family + "!" + a.ID}
			b.ID = "pkg:" + b.AUMID

			name := a.DisplayName
			if name == "" {
				name = m.DisplayName
			}
			name = loadIndirectString(resourceURI(full, m.Name, name))
			if name == "" || strings.HasPrefix(name, "@") || strings.HasPrefix(strings.ToLower(name), "ms-resource:") {
				name = nameFromAUMID(b.AUMID)
			}
			b.Name = name

			if a.Logo != "" {
				p := loadIndirectString(fileResourceURI(full, m.Name, a.Logo))
				if p == "" || strings.HasPrefix(p, "@") {
					p = bestLogoFile(filepath.Join(root, a.Logo))
				}
				b.IconPath = p
			}
			if a.Executable != "" && b.IconPath == "" {
				b.IconPath = filepath.Join(root, a.Executable)
			}

			// An execution alias is an ordinary .exe stub we can start with
			// arguments, which also allows private windows.
			if a.Alias != "" {
				wa := filepath.Join(os.Getenv("LOCALAPPDATA"), "Microsoft", "WindowsApps")
				for _, p := range []string{filepath.Join(wa, family, a.Alias), filepath.Join(wa, a.Alias)} {
					if _, err := os.Lstat(p); err == nil {
						b.Exe = p
						b.Command = `"` + p + `"`
						break
					}
				}
			}
			out = append(out, b)
		}
	}
	return out
}

var qualifierNum = regexp.MustCompile(`(targetsize|scale)-(\d+)`)

// bestLogoFile finds the on-disk variant of a logo like Assets\Logo.png,
// which is usually stored as Logo.targetsize-48.png, Logo.scale-200.png, ...
func bestLogoFile(path string) string {
	if _, err := os.Stat(path); err == nil {
		return path
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	matches, _ := filepath.Glob(base + ".*" + ext)
	if len(matches) == 0 {
		return ""
	}
	score := func(p string) int {
		l := strings.ToLower(p)
		s := 0
		if m := qualifierNum.FindStringSubmatch(l); m != nil {
			n, _ := strconv.Atoi(m[2])
			if m[1] == "targetsize" {
				s = 1000 - abs(n-48)
			} else {
				s = 500 - abs(n-200)
			}
		}
		if strings.Contains(l, "unplated") {
			s += 50
		}
		if strings.Contains(l, "contrast-") {
			s -= 2000
		}
		return s
	}
	sort.SliceStable(matches, func(i, j int) bool { return score(matches[i]) > score(matches[j]) })
	return matches[0]
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}
