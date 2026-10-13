# ZiadaPOS Desktop — initial testing build

These archives contain the ZiadaPOS desktop shell and its bundled web UI. They do **not** contain the ZiadaPOS Go API, PostgreSQL, Redis, or object storage; a reachable API service is required.

## Downloads

Choose the archive matching your operating system:

- Linux x86-64: `ziadapos-<version>-linux-amd64.tar.gz`
- Windows x86-64: `ziadapos-<version>-windows-amd64.zip`
- macOS: `ziadapos-<version>-macos-native.zip` (built for the architecture of the macOS runner used for that release; not a universal binary)

The GitHub Release also includes `SHA256SUMS` for verifying the downloaded files.

## Requirements and first run

- **Linux:** GTK 3 and WebKit2GTK 4.1 must be installed. Extract the archive and run `ZiadaPOS`.
- **Windows:** the WebView2 Runtime must be installed. Extract the archive and run `ZiadaPOS.exe`.
- **macOS:** extract the archive and open `ZiadaPOS.app`. The app uses the system WKWebView.
- Set `POS_API_URL` before launching if your API is not at the default `http://localhost:8080/api/v1` address. The desktop UI is not usable for POS operations without access to the API.

The executable also accepts `--browser` to launch the same bundled UI in the system browser. Its local web server binds to loopback on a temporary port and exits with the executable.

## App icon and archive format

All desktop builds use the ZiadaPOS blue-and-white logo. The Windows executable contains the generated multi-size app icon; the macOS `.app` bundle contains its generated `iconfile.icns`; and the Linux Wails window uses the embedded PNG icon. These downloads are **archives, not native installers**. In particular, the Linux tarball does not install a `.desktop` launcher or add a desktop-menu icon.

## Distribution status

These initial archives are unsigned, are not native installers, and do not include an automatic updater. macOS Gatekeeper may warn about an unsigned app; Windows code signing and WebView2 bootstrap are not yet included. The app is not an offline POS product.
