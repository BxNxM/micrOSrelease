# microsctl

> This is a PoC - not yet a fully functional installer

![microsctl TUI](media/TUI.png)

A Go TUI prototype for micrOS release workflows. It models USB install, USB
update, firmware/target selection, and micrOS node discovery on TCP port 9008.

USB device discovery, clean ESP installation, and state-preserving USB update
are native Go. Network discovery and status are native Go: scan TCP 9008,
validate `hello`, read version and feature flags.
Discovered devices persist between launches. Startup shows cached results
immediately and refreshes them in the background. Discovery and status refresh
every five minutes; press `r` to refresh manually. Results stream into existing
cards. The Nodes screen names the active background operation while it runs.
Two additional cards appear only when micrOS responds: **localhost** at
`127.0.0.1:9008` (blue) and **AP mode** at `192.168.4.1:9008` (orange). Both are
checked on every scan, independently of the LAN range. They disappear when a
refresh finds them unavailable and are never restored from the device cache.
Special cards can be hidden until the next scan from their details view.
Devices are merged by UID across addresses: a reachable localhost or AP endpoint
takes precedence over a LAN alias, so the simulator appears only once as localhost.

## Install

Run the installer from the directory where you want to place `microsctl`:

```bash
curl -fsSL https://raw.githubusercontent.com/BxNxM/micrOSrelease/main/dist/install.sh | sh
```

The installer detects macOS ARM64, Linux x64, or Windows x64 (from Git Bash,
MSYS2, or Cygwin) and downloads the matching prebuilt binary from this
repository's `dist/` directory. On Windows, the output filename is
`microsctl.exe`.

## Run

Requires Go 1.25 or newer.

```bash
go mod tidy
go run .
# Optional explicit network:
go run . --cidr 10.0.1.0/24
# Single binary:
go build -o microsctl .
```

Use arrow keys to navigate, `Enter` to select, `Y/N` to confirm, `</>` to
change the image, and `Q` to quit. Nodes is the home screen. Use arrows to select a card,
Enter for details, or open **USB Tools** for discovery, installation, updates, and USB scanning. Set `MICROS_PASSWORD` for protected
devices. Automatic scans use active private IPv4 interfaces, capped to /24 per
interface. No Python runtime or toolkit is required.

Node cards show identity, address, online status, version, release/development
mode, communication time, and WEBUI/ESPNOW/CRON/TIMIRQ flags. Unknown feature
values are shown as `n/a`. Password-protected nodes require credentials to be
identified. Discovery does not import the Python toolkit's device cache.

## Storage

Add build-time content to `storage/frameworks/<board>/` (firmware binaries) and
`storage/modules/` (modules/resources). Rebuild with `go build -o microsctl .`;
Go embeds these directories recursively into the single executable. List the
bundled files with `./microsctl --list-assets`. Symlinks and files beginning
with `.` or `_` are not included.

Go features can read bundled content without extracting it:

```go
import (
    "io/fs"
    assets "github.com/micros/microsctl/storage"
)

data, err := fs.ReadFile(assets.Files(), "frameworks/micrOS-esp32.bin")
```

Embedded files are read-only; adding files requires rebuilding. Open
**USB Tools**, then press **f** to browse bundled `.bin`/`.uf2` images.
Arrow keys browse, Enter selects the release target, and Escape cancels.
Board and version metadata comes from micrOS filenames. USB scan detects the
same common ESP serial adapters as the Python toolkit (`wchusbserial`,
`SLAB_USBtoUART`, `USB0`, `usbserial`, `usbmodem`, `ttyACM`, and `ttyUSB` on Unix;
CP210, CH340, CH343, and CH9102 adapters plus native Espressif USB interfaces on
Windows). On macOS, discovery prefers `cu.*` ports and hides their `tty.*` duplicates. Install is available
without Python or `esptool.py`: it performs a full erase, compressed write,
MD5 verification, and reset. Chip identity and physical flash capacity are checked before erase; oversized bundled resources are rejected before connecting. Both install and update copy bundled resources
through the native MicroPython raw REPL client: `storage/modules/modules/…`
maps to `/modules/…`, `storage/modules/web/…` maps to `/web/…`, and other
directories follow the same pattern. Update validates the connected chip and
MicroPython runtime. Before replacing
firmware it archives the entire device filesystem, including hidden configuration,
custom modules, pin maps, user files, and empty directories, to a private ZIP in
`backups/`. Downloads are SHA-256 verified before erase. Updates restore preserved
files and replace bundled resource destinations with the release copies. Backups
are limited to 1 MiB per file and 64 MiB total; incomplete or oversized backups
stop the update before erase. A host backup directory is required for flashing.

If both micrOS and MicroPython versions match on the correct chip, update refreshes
resources without flashing. REPL maintenance feeds a temporary watchdog during
transfers and uses a hardware reset afterward to resume the application and clear
that watchdog. Failures before device writes also restart the unchanged application.
During **Reconnect to MicroPython REPL**, you may unplug and reconnect the USB
cable. Keep the operation open: it waits until the device returns, then continues
restoring files without flashing again. Stable USB serial identity lets it follow
a changed port name; adapters without a serial identity must return on their
original port. The chip and MicroPython version are checked before restoration.
This stage has no default time limit; **Ctrl+C** cancels. Disconnecting during
flashing or file transfers is still unsafe.

The USB card starts with **Discovery** (selected by default), followed by
**Install micrOS**, **Update micrOS**, and **USB Scan**. Discovery scans USB ports
and identifies each board through the ESP ROM bootloader, showing chip type,
silicon revision, flash size and JEDEC ID, MAC, and chip features. It selects the
framework board matching the selected device; a single matching firmware image
is selected automatically. Identification resets boards. **USB Scan** only lists
ports and does not identify or reset them.

Runtime data lives outside the executable in the platform user configuration
directory under `microsctl` (macOS: `~/Library/Application Support/microsctl`).
Use `--data-dir ./data` for portable storage. `devices.json` stores last-known
device observations; `firmware/` is reserved for firmware files; and `backups/`
stores timestamped `node_config.json` snapshots and filesystem ZIP archives
created before USB updates.
Passwords are not saved separately, but configuration backups can contain node
credentials and are therefore created with user-only permissions. Cards mark
cached observations as **saved**; details include the last check time. Firmware
download/import is not implemented.

## Structure

- `main.go`: composition root; wires features to the UI.
- `internal/usb`: native USB inventory, ESP install, and preserved-state update.
- `internal/micropython`: serial raw-REPL execution and verified file transfer.
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

## Hardware validation

Hardware tests are opt-in and reset/update the selected board. Keep their backups
in a persistent directory:

```bash
MICROS_TEST_PORT=/dev/cu.usbmodem2101 \
MICROS_TEST_BACKUP_DIR=/absolute/path/to/backups \
go test ./internal/usb -run '^TestHardwareUSBUpdate$' -v -count=1 -timeout=16m
```

The test reads the runtime, selects a matching image, and saves a verified filesystem
backup before probing the bootloader. It updates the board and verifies user configuration, preserved files, and
release resources. Set `MICROS_TEST_FLASH=1` to exercise full erase and restore
even when the installed runtime is already current. Normal `go test ./...` never
opens a serial port.

To test physical reconnect without flashing or writing files:

```bash
MICROS_TEST_RECONNECT_PORT=/dev/cu.usbmodem2101 \
MICROS_TEST_RECONNECT_CHIP=esp32c6 \
go test ./internal/usb -run '^TestHardwareUSBReconnect$' -v -count=1 -timeout=16m
```

After the test prints `READY`, unplug USB for at least three seconds and reconnect
it. The test validates the returning interpreter and resets the board.
