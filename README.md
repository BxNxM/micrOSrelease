# micrOS Release Manager

> This is a PoC - not yet a fully functional installer

![micrOS Release Manager TUI](media/TUI.png)

A Go TUI prototype for micrOS release workflows. It models USB install, USB
update, firmware/target selection, and micrOS node discovery on TCP port 9008.

USB install/update use dummy implementations. Network discovery and status are
native Go: scan TCP 9008, validate `hello`, read version and feature flags.
Discovered devices persist between launches. Startup shows cached results
immediately and refreshes them in the background. Discovery and status refresh
every five minutes; press `r` to refresh manually. Results stream into existing
cards. The Nodes screen names the active background operation while it runs.

## Run

Requires Go 1.25 or newer.

```bash
go mod tidy
go run .
# Optional explicit network:
go run . --cidr 10.0.1.0/24
# Single binary:
go build -o micros-release .
```

Use arrow keys to navigate, `Enter` to select, `Y/N` to confirm, `</>` to
change the image, and `Q` to quit. Simulated latency can be changed with
`--demo-delay=500ms`. Nodes is the home screen. Use arrows to select a card,
Enter for details, or **+ ADD/Actions** for USB tools. Set `MICROS_PASSWORD` for protected
devices. Automatic scans use active private IPv4 interfaces, capped to /24 per
interface. No Python runtime or toolkit is required.

Node cards show identity, address, online status, version, release/development
mode, communication time, and WEBUI/ESPNOW/CRON/TIMIRQ flags. Unknown feature
values are shown as `n/a`. Password-protected nodes require credentials to be
identified. Discovery does not import the Python toolkit's device cache.

## Storage

Add build-time content to `storage/frameworks/<board>/` (firmware binaries) and
`storage/modules/` (modules/resources). Rebuild with `go build -o micros-release .`;
Go embeds these directories recursively into the single executable. List the
bundled files with `./micros-release --list-assets`. Symlinks and files beginning
with `.` or `_` are not included.

Go features can read bundled content without extracting it:

```go
import (
    "io/fs"
    assets "github.com/micros/micros-release/storage"
)

data, err := fs.ReadFile(assets.Files(), "frameworks/micrOS-esp32.bin")
```

Embedded files are read-only; adding files requires rebuilding. Open
**+ ADD/Actions**, then press **f** to browse bundled `.bin`/`.uf2` images.
Arrow keys browse, Enter selects the release target, and Escape cancels.
Board and version metadata comes from micrOS filenames. USB targets and
install/update remain simulated; selecting an image does not flash hardware.

Runtime data lives outside the executable in the platform user configuration
directory under `micros-release` (macOS: `~/Library/Application Support/micros-release`).
Use `--data-dir ./data` for portable storage. `devices.json` stores last-known
device observations; `firmware/` is reserved for firmware files. Passwords are
not saved. Cards mark cached observations as **saved**; details include the last
check time. Firmware download/import and real USB flashing are not implemented.

## Structure

- `main.go`: composition root; wires features to the UI.
- `internal/usb`: USB inventory, install, and update contracts/dummies.
- `internal/network`: socket client, discovery, and node status.
- `internal/tui`: Bubble Tea state, events, commands, styles, and rendering.
- `internal/storage`: persistent device cache and firmware directory.
- `storage/frameworks`, `storage/modules`: embedded build-time assets.

UI screens live under `internal/tui/views/` in `actions_view.go`, `nodes_view.go`,
`device_details_view.go`, and `firmware_view.go`; `view.go` only routes screens.
Reusable rendering lives under `internal/tui/widgets/` in `*_widget.go` (cards, feature values, menus,
release target, progress, and status). `*_navigation.go` handles screen-specific
keys; `model.go`, `update.go`, and `commands.go` own state and asynchronous work
in `internal/tui/`. Views receive a read-only rendering snapshot, not services.

The UI depends only on feature interfaces, so real Go implementations can
replace the dummies without changing screen logic.

## TUI choice

This prototype uses [Bubble Tea](https://github.com/charmbracelet/bubbletea)
for the event loop and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
for styling (the library sometimes misremembered as “libglass”). Alternatives
include `tview` for widget-heavy forms, `gocui` for a smaller view-based API,
and `termui` for dashboard-style layouts.
