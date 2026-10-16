// Generates rsrc_windows_<arch>.syso: icon, manifest (per-monitor DPI v2,
// common controls v6) and version info for Browser Picker.
package main

import (
	"os"
	"path/filepath"

	"github.com/tc-hib/winres"
	"github.com/tc-hib/winres/version"
)

func main() {
	out := os.Args[1]
	f, err := os.Open(os.Args[2])
	must(err)
	ico, err := winres.LoadICO(f)
	must(err)
	rs := &winres.ResourceSet{}
	must(rs.SetIcon(winres.ID(1), ico))
	rs.SetManifest(winres.AppManifest{
		Description:         "Browser Picker",
		Compatibility:       winres.Win10AndAbove,
		ExecutionLevel:      winres.AsInvoker,
		DPIAwareness:        winres.DPIPerMonitorV2,
		UseCommonControlsV6: true,
		LongPathAware:       true,
	})
	var vi version.Info
	vi.SetFileVersion("1.0.0.0")
	vi.SetProductVersion("1.0.0.0")
	vi.Set(0x0409, version.ProductName, "Browser Picker")
	vi.Set(0x0409, version.FileDescription, "Browser Picker")
	vi.Set(0x0409, version.OriginalFilename, "BrowserPicker.exe")
	vi.Set(0x0409, version.LegalCopyright, "Free to use")
	rs.SetVersionInfo(vi)
	for _, a := range []winres.Arch{winres.ArchAMD64, winres.ArchARM64} {
		w, err := os.Create(filepath.Join(out, "rsrc_windows_"+string(a)+".syso"))
		must(err)
		must(rs.WriteObject(w, a))
		w.Close()
	}
}

func must(err error) {
	if err != nil {
		panic(err)
	}
}
