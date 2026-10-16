package main

import (
	"image"
	"testing"
)

func TestBuildCommandLine(t *testing.T) {
	const u = "https://example.com/a?b=1&c=2"
	cases := []struct {
		cmd, exe string
		private  bool
		want     string
	}{
		{`"C:\Program Files\Google\Chrome\Application\chrome.exe"`, `C:\Program Files\Google\Chrome\Application\chrome.exe`, false,
			`"C:\Program Files\Google\Chrome\Application\chrome.exe" "` + u + `"`},
		{`"C:\Program Files\Google\Chrome\Application\chrome.exe"`, `C:\Program Files\Google\Chrome\Application\chrome.exe`, true,
			`"C:\Program Files\Google\Chrome\Application\chrome.exe" --incognito "` + u + `"`},
		{`C:\Program Files\Mozilla Firefox\firefox.exe`, `C:\Program Files\Mozilla Firefox\firefox.exe`, true,
			`"C:\Program Files\Mozilla Firefox\firefox.exe" -private-window "` + u + `"`},
		{`"C:\x\msedge.exe" --profile-directory=Default "%1"`, `C:\x\msedge.exe`, true,
			`"C:\x\msedge.exe" --inprivate --profile-directory=Default "` + u + `"`},
		{`"C:\x\foo.exe" -url %1`, `C:\x\foo.exe`, true,
			`"C:\x\foo.exe" -url "` + u + `"`},
	}
	for _, c := range cases {
		if got := buildCommandLine(c.cmd, c.exe, u, c.private); got != c.want {
			t.Errorf("\n got %s\nwant %s", got, c.want)
		}
	}
	if got := buildCommandLine(`"C:\a\b.exe"`, `C:\a\b.exe`, `https://x/"&calc`, false); got != `"C:\a\b.exe" "https://x/%22&calc"` {
		t.Errorf("quote not escaped: %s", got)
	}
}

func TestExeAndIcon(t *testing.T) {
	if e := exeFromCommand(`"C:\Program Files\Opera\launcher.exe" --x`); e != `C:\Program Files\Opera\launcher.exe` {
		t.Error(e)
	}
	if e := exeFromCommand(`C:\Program Files\Mozilla Firefox\firefox.exe -osint`); e != `C:\Program Files\Mozilla Firefox\firefox.exe` {
		t.Error(e)
	}
	if p, i := parseIconRef(`"C:\x\chrome.exe",-101`); p != `C:\x\chrome.exe` || i != -101 {
		t.Error(p, i)
	}
	if p, i := parseIconRef(`C:\x\firefox.exe,0`); p != `C:\x\firefox.exe` || i != 0 {
		t.Error(p, i)
	}
}

func TestSiteKeyAndPrivate(t *testing.T) {
	if s := siteKey("https://WWW.YouTube.com/watch?v=1"); s != "youtube.com" {
		t.Error(s)
	}
	if s := siteKey("https://user@sub.example.org:8443/x"); s != "sub.example.org" {
		t.Error(s)
	}
	if f := privateFlag(`C:\Users\a\AppData\Local\Programs\Opera GX\launcher.exe`); f != "--private" {
		t.Error(f)
	}
	if f := privateFlag(`C:\Some\Other\launcher.exe`); f != "" {
		t.Error(f)
	}
}

func TestPackagedHelpers(t *testing.T) {
	if f := privateFlagByName("Arc TheBrowserCompany.Arc_ttt1ap7aakyb4!Arc"); f != "--incognito" {
		t.Error(f)
	}
	if f := privateFlagByName("Some App Foo.Bar_123!App"); f != "" {
		t.Error(f)
	}
	src := image.NewNRGBA(image.Rect(0, 0, 100, 50)) // wide, opaque red
	for i := 0; i < len(src.Pix); i += 4 {
		src.Pix[i], src.Pix[i+3] = 255, 255
	}
	px := resizeNRGBA(src, 24)
	at := func(x, y int) []byte { return px[(y*24+x)*4 : (y*24+x)*4+4] }
	if c := at(12, 12); c[0] != 255 || c[3] != 255 {
		t.Errorf("centre %v", c)
	}
	if c := at(12, 1); c[3] != 0 { // letterboxed area stays transparent
		t.Errorf("top %v", c)
	}
}

const arcLikeManifest = `<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:uap3="http://schemas.microsoft.com/appx/manifest/uap/windows10/3"
  xmlns:desktop="http://schemas.microsoft.com/appx/manifest/desktop/windows10">
  <Identity Name="TheBrowserCompany.Arc" Publisher="CN=X" Version="1.2.3.0" ProcessorArchitecture="x64"/>
  <Properties><DisplayName>Arc</DisplayName><Logo>Assets\StoreLogo.png</Logo></Properties>
  <Applications>
    <Application Id="Arc" Executable="Arc.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="Arc" Square44x44Logo="Assets\Square44x44Logo.png" Square150x150Logo="Assets\Square150x150Logo.png"/>
      <Extensions>
        <uap:Extension Category="windows.protocol"><uap:Protocol Name="http"/></uap:Extension>
        <uap:Extension Category="windows.protocol"><uap:Protocol Name="https"/></uap:Extension>
        <uap3:Extension Category="windows.appExecutionAlias"><uap3:AppExecutionAlias><desktop:ExecutionAlias Alias="arc.exe"/></uap3:AppExecutionAlias></uap3:Extension>
      </Extensions>
    </Application>
    <Application Id="Helper" Executable="Helper.exe"><uap:VisualElements DisplayName="ms-resource:HelperName" Square44x44Logo="a.png"/></Application>
  </Applications>
</Package>`

func TestManifest(t *testing.T) {
	data := []byte(arcLikeManifest)
	if !mightHandleWeb(data) {
		t.Fatal("pre-check failed")
	}
	m, _ := parseManifest(data)
	if m.Name != "TheBrowserCompany.Arc" || m.DisplayName != "Arc" || len(m.Apps) != 2 {
		t.Fatalf("%+v", m)
	}
	a := m.Apps[0]
	if a.ID != "Arc" || !a.HandlesWeb || a.Alias != "arc.exe" || a.Logo != `Assets\Square44x44Logo.png` {
		t.Errorf("%+v", a)
	}
	if m.Apps[1].HandlesWeb {
		t.Error("helper should not handle web")
	}
	full := "TheBrowserCompany.Arc_1.2.3.0_x64__ttt1ap7aakyb4"
	if f := familyName(full); f != "TheBrowserCompany.Arc_ttt1ap7aakyb4" {
		t.Error(f)
	}
	if u := resourceURI(full, m.Name, "ms-resource:HelperName"); u != "@{"+full+"?ms-resource://TheBrowserCompany.Arc/Resources/HelperName}" {
		t.Error(u)
	}
	if u := fileResourceURI(full, m.Name, `Assets\Square44x44Logo.png`); u != "@{"+full+"?ms-resource://TheBrowserCompany.Arc/Files/Assets/Square44x44Logo.png}" {
		t.Error(u)
	}
	if f := privateFlag(`C:\Users\a\AppData\Local\Microsoft\WindowsApps\arc.exe`); f != "--incognito" {
		t.Error(f)
	}
}
