package main

// Parsing of MSIX/AppX package manifests (AppxManifest.xml), used to find
// packaged browsers such as Arc. Platform-independent so it can be tested
// anywhere.

import (
	"bytes"
	"encoding/xml"
	"strings"
)

type manifestApp struct {
	ID          string
	Executable  string
	DisplayName string // may be "ms-resource:..."
	Logo        string // relative path, e.g. Assets\Square44x44Logo.png
	Alias       string // app execution alias, e.g. arc.exe
	HandlesWeb  bool   // declares the http or https protocol
}

type packageManifest struct {
	Name        string // identity name, e.g. TheBrowserCompany.Arc
	DisplayName string
	Apps        []manifestApp
}

// mightHandleWeb is a cheap pre-check before parsing the XML.
func mightHandleWeb(data []byte) bool {
	l := bytes.ToLower(data)
	return bytes.Contains(l, []byte("windows.protocol")) &&
		(bytes.Contains(l, []byte(`name="https"`)) || bytes.Contains(l, []byte(`name="http"`)))
}

func parseManifest(data []byte) (packageManifest, error) {
	var m packageManifest
	d := xml.NewDecoder(bytes.NewReader(data))
	d.Strict = false
	var cur *manifestApp
	var inProps, inPropName bool
	attr := func(se xml.StartElement, name string) string {
		for _, a := range se.Attr {
			if strings.EqualFold(a.Name.Local, name) {
				return a.Value
			}
		}
		return ""
	}
	for {
		tok, err := d.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "Identity":
				if m.Name == "" {
					m.Name = attr(t, "Name")
				}
			case "Properties":
				if cur == nil {
					inProps = true
				}
			case "DisplayName":
				inPropName = inProps
			case "Application":
				m.Apps = append(m.Apps, manifestApp{ID: attr(t, "Id"), Executable: attr(t, "Executable")})
				cur = &m.Apps[len(m.Apps)-1]
			case "VisualElements":
				if cur != nil {
					cur.DisplayName = attr(t, "DisplayName")
					cur.Logo = attr(t, "Square44x44Logo")
					if cur.Logo == "" {
						cur.Logo = attr(t, "Square150x150Logo")
					}
				}
			case "Protocol":
				if cur != nil {
					n := strings.ToLower(attr(t, "Name"))
					if n == "http" || n == "https" {
						cur.HandlesWeb = true
					}
				}
			case "ExecutionAlias":
				if cur != nil && cur.Alias == "" {
					cur.Alias = attr(t, "Alias")
				}
			}
		case xml.CharData:
			if inPropName && m.DisplayName == "" {
				m.DisplayName = strings.TrimSpace(string(t))
			}
		case xml.EndElement:
			switch t.Name.Local {
			case "Application":
				cur = nil
				// Pointers into m.Apps stay valid only until the next append,
				// and cur is reset before that happens.
			case "Properties":
				inProps = false
			case "DisplayName":
				inPropName = false
			}
		}
	}
	return m, nil
}

// familyName turns a package full name
// (Name_Version_Arch_ResourceId_PublisherId) into its family name
// (Name_PublisherId).
func familyName(fullName string) string {
	first := strings.IndexByte(fullName, '_')
	last := strings.LastIndexByte(fullName, '_')
	if first < 0 || last <= first {
		return fullName
	}
	return fullName[:first] + fullName[last:]
}

// resourceURI builds the "@{package?ms-resource://...}" string that
// SHLoadIndirectString resolves to a localized string.
func resourceURI(fullName, pkgName, value string) string {
	if !strings.HasPrefix(strings.ToLower(value), "ms-resource:") {
		return value
	}
	res := value[len("ms-resource:"):]
	switch {
	case strings.HasPrefix(res, "//"):
		res = "ms-resource:" + res
	case strings.HasPrefix(res, "/"):
		res = "ms-resource://" + pkgName + res
	case strings.Contains(res, "/"):
		res = "ms-resource://" + pkgName + "/" + res
	default:
		res = "ms-resource://" + pkgName + "/Resources/" + res
	}
	return "@{" + fullName + "?" + res + "}"
}

// fileResourceURI builds the indirect string that resolves a package asset
// (e.g. Assets\Logo.png) to its best scale/target-size variant on disk.
func fileResourceURI(fullName, pkgName, rel string) string {
	return "@{" + fullName + "?ms-resource://" + pkgName + "/Files/" + strings.ReplaceAll(rel, `\`, "/") + "}"
}
