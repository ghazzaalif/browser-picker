# Browser Picker for Windows

Set Browser Picker as your default browser, and every link you open from
outside a browser (email, Slack, Teams, Word, PDFs, Discord, the Start menu…)
pops up a small window asking which of your installed browsers to use.

- Lists every browser Windows knows about (Chrome, Edge, Firefox, Brave, Opera,
  Vivaldi, Zen, LibreWolf…), including packaged / Microsoft Store browsers
  such as Arc, with their icons, and updates automatically when you install or
  remove one.
- Click a browser, or press **1–9**, or use **↑/↓ + Enter**. **Esc** cancels.
- **Ctrl+click** (or Ctrl+number) opens the link in a private/incognito window.
- Tick **"Always use for example.com"** to skip the question for that site from
  then on. Hold **Shift** when clicking a link to get the picker anyway.
- **Copy link** copies the URL instead of opening it.
- Follows Windows light/dark mode and your accent color; sharp on high-DPI screens.
- Single ~2 MB .exe, no installer, no admin rights, no background process.

## Install

1. Download `BrowserPicker.exe` from the
   [latest release](https://github.com/ghazzaalif/browser-picker/releases/latest)
   and run it (use `BrowserPicker-ARM64.exe` on ARM PCs such as
   Snapdragon laptops). Windows SmartScreen may warn that the app is unrecognized
   because it isn't code-signed: click **More info → Run anyway**.
2. Click **Install and set as default browser**. The app copies itself to
   `%LOCALAPPDATA%\Programs\BrowserPicker` and opens Windows **Default apps**.
3. Windows doesn't let apps make themselves the default, so finish it there:
   - **Windows 11:** on the Browser Picker page click **Set default** (or set
     **HTTP** and **HTTPS** to Browser Picker).
   - **Windows 10:** under **Web browser**, choose **Browser Picker**.
4. Click **Try it** to see the picker.

To get back to the settings later, click the gear icon in the picker, or run
`%LOCALAPPDATA%\Programs\BrowserPicker\BrowserPicker.exe`.

## Uninstall

Open the settings and click **Uninstall**. It removes the registration, the
remembered sites, and the app folder, then opens Default apps so you can pick a
new default browser.

## Command line

```
BrowserPicker.exe                 settings window
BrowserPicker.exe <url>           show the picker for <url>
BrowserPicker.exe --register      install for the current user without UI
BrowserPicker.exe --unregister    remove the registration without UI
BrowserPicker.exe --diagnose      write a detection report and open it in Notepad
```

## How it works

- Browsers are read from `Software\Clients\StartMenuInternet` in both HKCU and
  HKLM (64- and 32-bit views), the same list Windows' Default apps uses.
- Packaged (MSIX/Store) browsers like Arc are found by reading the manifest of
  each installed package and picking apps that declare the http/https
  protocol. They're launched through their execution alias, their https
  ProgID, or app activation, in that order.
- Browser Picker registers itself in StartMenuInternet under HKCU, plus `BrowserPickerURL`
  (http/https) and `BrowserPickerHTML` (.htm/.html) handlers in
  `HKCU\Software\Classes`.
- Remembered sites live in `HKCU\Software\BrowserPicker\Rules`.

## Building from source

Requires Go 1.21+. The `rsrc_windows_*.syso` files (icon, manifest, version
info) are already generated, so a plain cross-compile works from any OS:

```
GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-H windowsgui -s -w" -o BrowserPicker.exe .
GOOS=windows GOARCH=arm64 go build -trimpath -ldflags "-H windowsgui -s -w" -o BrowserPicker-ARM64.exe .
go test ./...
```

The code only uses Go's standard library (Win32 is called through `syscall`).

To regenerate the icon / manifest / version resources after editing
`assets/icon.ico` or `tools/genres/main.go`:

```
cd tools/genres
go mod tidy
go run . ../.. ../../assets/icon.ico
```

## License

MIT

## Releasing

Publish a release on GitHub (Releases → Draft a new release → new tag such as
`v1.0.1` → Publish). The workflow builds both executables and attaches them to
the release automatically. Every push to `main` also builds them as a
downloadable workflow artifact.
