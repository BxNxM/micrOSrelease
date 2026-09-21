# Architecture

`microsctl` is a Go 1.25 terminal application using Bubble Tea v2 and Lip Gloss v2.
USB flashing (`tinygo.org/x/espflasher`), serial REPL access (`go.bug.st/serial`),
and TCP discovery run natively in Go. See [README.md](README.md) for usage and
[AGENTS.md](AGENTS.md) for change guidelines.

## Boundaries and layout

| Location | Responsibility |
| --- | --- |
| `main.go` | Composition root: flags, storage, embedded assets, services, TUI. |
| `internal/tui/` | Model, event handling, navigation, asynchronous commands. |
| `internal/tui/views/*_view.go` | Compose Nodes, details, Shell, USB actions, and firmware screens. |
| `internal/tui/widgets/*_widget.go` | Reusable rendering; `state.go` holds the rendering snapshot, `styles.go` the styles. |
| `internal/usb/` | Inventory, board/image selection, ESP probing, install/update orchestration, backups, reconnect. |
| `internal/micropython/` | Raw/raw-paste REPL, runtime inspection, verified file transfer, watchdog maintenance. |
| `internal/network/` | TCP discovery, interactive sessions, prompt framing, identity and Web UI URLs. |
| `internal/network/self_update/` | Application release manifest, downloads, executable replacement, and restart. |
| `internal/storage/` | Host data directory and versioned device cache. |
| `storage/` | Read-only embedded firmware, board configuration, modules, and web files. |
| `scripts/`, `dist/` | Asset refresh script, installer, and platform binaries. |

`main.go` injects `usb.ReleaseManager` and `network.Service` through `usb.Manager`
and `network.Discoverer`. The network service accepts a `DeviceStore`, implemented
by `internal/storage.Store`. Optional cache/removal capabilities are discovered
through small interfaces. USB connection factories are replaceable in tests.

## UI and discovery flow

`model.go` owns UI state; `update.go` and focused event handlers consume messages.
`commands.go` runs I/O and streams device/stage messages through channels with
context cancellation. `*_navigation.go` owns screen-specific keys and transitions.
`view.go` builds `widgets.State` and routes to views. Views and widgets treat this
snapshot as read-only and receive no services.

Shell uses the optional `network.ShellConnector` interface and a persistent TCP
client. `client.go` handles prompt framing and cumulative response snapshots;
`shell.go` adds interactive authentication and UID verification. Discovery uses
a three-second timeout; Shell uses a 20-second inactivity timeout and a strict
1 MiB response limit. Close messages are classified at EOF, not TCP read boundaries.
`exit` sends once and closes without awaiting a reply.

TUI Shell commands stream through cancellable channels. Session generations
reject late replies and close late connections. The model retains up to 100
commands and 64 KiB of completed transcript in memory. A shared wrapped-line
layout supports scrollback; byte ranges identify local command echoes so styling
survives wrapping and trimming without interpreting server text as UI markup.
See [AGENTS.md](AGENTS.md) for Shell behavior to preserve.

Startup loads cached nodes, inventories USB ports/assets, and begins network
discovery. A single five-minute timer chain refreshes nodes without overlapping
scans. Returning from a finished USB install/update triggers the manual-refresh
path; leaving during USB work defers that refresh until completion. The same
scan/removal guards prevent overlapping discovery. Network discovery uses 32
workers on TCP 9008, validates `hello`, then reads
version and feature flags. It checks configured/private IPv4 ranges, saved
addresses, localhost, and AP mode. Observations merge by UID; reachable special
endpoints take precedence over LAN aliases and are never displayed from cache
alone. Cache writes use a temporary file, sync, and rename.

## Application releases and self-update

`MANIFEST.yaml` is the single source of the microsctl version and is embedded
by `go build`. `--version` prints it without starting services.
`make build` refreshes manifest platform paths and the latest informational micrOS
version before building all four `dist/` executables. `make manifest` refreshes
that metadata alone, preserving the microsctl version and without inspecting
binary contents. Change `microsctl.version` in the manifest for each release;
publish the manifest and matching `dist/` binaries together.

`microsctl.url` in `MANIFEST.yaml` is the sole release URL, including the branch
and trailing `/` (for example, `https://raw.githubusercontent.com/BxNxM/micrOSrelease/main/`).
The build embeds it and `make manifest` preserves manual edits. Startup fetches
`<url>MANIFEST.yaml` with a 10-second timeout. The fetched manifest's `url` is
used with the selected binary's `path`, so newer manifests can move downloads
to another repository, branch, or host. Requests use the latest branch contents;
there is no commit lookup or pinning. Editing the local manifest takes effect
in the next build.

Any different version is offered, including
a lower version. `u` installs the exact OS/architecture artifact from `dist/`;
unsupported platforms never fall back to another binary. Downloads have a
five-minute timeout and 128 MiB limit; the manifest is limited to 1 MiB.

`selfupdate.ExecutableInstaller` resolves the executable path, stages beside it,
finishes the bounded download, syncs the file, and retains a
`.microsctl-backup-*` copy before replacement. Unix replaces by rename; Windows
moves the running image aside and restores it if the replacement rename fails.
Windows may retain a `.running.exe` file until the old process exits. TUI update
commands stream progress, block navigation/USB work, and wait for cancellation
before quitting. Successful installation exits Bubble Tea, then restarts with
the original arguments, environment, and working directory. Restart uses `exec`
on Unix and a child process on Windows. A failed restart prints the path to run.

## USB lifecycle and preservation

Inventory lists serial ports and embedded `.bin`/`.uf2` images; installation
currently supports the `esp-rom` protocol. Discovery probes ESP bootloaders and
resets boards; a plain scan does not. Probing uses UART reset sequences for known
USB-to-UART bridges (with a port-name fallback), and auto reset for native
Espressif USB and unknown endpoints. Install/update apply the same transport
selection when the manifest requests auto reset; original ESP32/ESP8266 targets
always use UART reset. Explicit manifest reset modes still take precedence.
Board/version metadata comes from firmware filenames; board behavior comes from
`frameworks/<board>/install.json`.
The flash-ID adapter sets/restores the original ESP32's dedicated SPI receive
length register to work around espflasher v0.8.1's incomplete JEDEC reads. Both
discovery and pre-erase capacity validation use this adapter.

Both install and update preload/validate firmware, configuration, and resource
payloads before touching hardware. Flashing checks chip identity and physical
flash capacity before erase, then writes/verifies the image and resets according
to board settings. Bundled configurations enable full erase, compression, and
reset; the ESP flasher performs MD5 verification.

Update reads and archives `node_config.json`, then validates the runtime. If chip,
micrOS version, and MicroPython version match, it only refreshes resources and
resets. Otherwise it requires a host backup directory and archives the entire
filesystem, including hidden files and empty directories, before erase. Downloads
are SHA-256 verified; unreadable files, changed sizes, or limits of 1 MiB/file and
64 MiB total abort the backup. Archives are privately created and synced, then
published durably: Unix syncs the renamed entry and ancestor directories; Windows
uses a write-through move.
Unavailable runtime access aborts before flashing. Bootloader connection failure
after backup aborts before erase, retains the archive path, and never retries as
a clean install. This safety policy is shared by every board type.

After flashing, reconnect uses the physical USB location and vendor when available
(macOS registry), otherwise stable USB serial identity, otherwise the original
port. Physical matching supports firmware changes to USB serial/product identity
on the same socket and applies to every board type. Multiple matching endpoints
retain the selected port when present (including macOS CP210x driver aliases);
otherwise reconnect waits rather than guessing.
Reconnect validates chip/runtime and defaults to waiting
until cancellation. Restore preserves user files except release destinations and
configured node-config paths; configuration is written to `restore_path` with the
new micrOS version, then release resources are copied. The already-current path
does not rewrite configuration. File writes use unique, collision-checked scratch
paths and verify uploads before replacement; release `main.py` files go last.
REPL maintenance feeds a temporary watchdog, cleared by the final hardware reset.
Reset first synchronizes the filesystem and waits for a preparation acknowledgement
and interpreter prompt. It then sends `machine.reset()` separately, without reading
from the disappearing USB endpoint; earlier disconnects and write failures remain
errors. Final-reset failures explain that verified files only need a normal reboot.
Resource copies stream the current
destination and file count through stage details before each verified upload;
the TUI replaces a single detail line, truncating it to the panel width. Install
and Update also forward the shared flasher's active stage and transfer percentage
into that detail line, without duplicating the flashing workflow.
Operation errors are retained separately from transient status and rendered in
the Install/Update panel, including preflight failures before stages exist.
Background failures reopen the operation panel and retain recovery details until
the user leaves that panel.
Pre-write REPL failures attempt to restart the unchanged application; post-flash
update errors include the backup path.

## Embedded releases

[`storage/assets.go`](storage/assets.go) exposes `assets.Files()` as `fs.FS` using
`go:embed frameworks modules`. Rebuild after asset changes. Hidden/underscore
entries and symlinks are not bundled. Host runtime data is separate from assets.

Resources map from `storage/modules/<path>` to `/<path>` on the device, excluding
the top-level README. Direct `modules/IO_<platform>.py`/`.mpy` files are filtered by
the board's `platform`; explicit `repl.resources` entries override destinations.
See [board configuration](storage/frameworks/README.md) and
[resource mapping](storage/modules/README.md) for details.

```sh
make mr MICROS_SOURCE=/path/to/micrOS  # Or: make micros-refresh
make build                           # macOS ARM64, Linux x64/ARM64, Windows x64
make linux-arm64                     # 64-bit Raspberry Pi OS and other ARM64 Linux
```

Refresh uses `micrOS/micropython/micrOS-*.bin`, copies firmware into existing board
folders, refreshes only module filenames already selected in
`storage/modules/modules/`, and recursively copies non-hidden precompiled web
files from `toolkit/workspace/precompiled/web/`. Module sources come from the
adjacent `precompiled/modules/`. It preserves older destination files and
`install.json`; it does not compile micrOS. The source path comes from
`MICROS_SOURCE`, the ignored `.micros-source` cache, or an interactive prompt.

## Validation

```sh
sh scripts/install-test.sh
go test ./...
go vet ./...
```

With hardware-test variables unset, tests use fakes/local fixtures and do not
open serial ports. Live tests reset or update the selected board; use only an
explicitly selected test device and a persistent backup directory:

```sh
MICROS_TEST_PROBE_PORT=/dev/cu.usbserial-0001 \
MICROS_TEST_PROBE_CHIP=esp32 \
go test ./internal/usb -run '^TestHardwareUSBProbe$' -v -count=1 -timeout=45s

MICROS_TEST_INSTALL_PORT=/dev/cu.SLAB_USBtoUART \
MICROS_TEST_INSTALL_BOARD=esp32 \
MICROS_TEST_BACKUP_DIR=/absolute/path/to/backups \
go test ./internal/usb -run '^TestHardwareUSBInstall$' -v -count=1 -timeout=16m

MICROS_TEST_PORT=/dev/cu.usbmodem2101 \
MICROS_TEST_BACKUP_DIR=/absolute/path/to/backups \
go test ./internal/usb -run '^TestHardwareUSBUpdate$' -v -count=1 -timeout=16m

MICROS_TEST_RECONNECT_PORT=/dev/cu.usbmodem2101 \
MICROS_TEST_RECONNECT_CHIP=esp32c6 \
go test ./internal/usb -run '^TestHardwareUSBReconnect$' -v -count=1 -timeout=16m
```

`MICROS_TEST_FLASH=1` forces erase/restore in the update test even when current.
The probe test only identifies and resets the selected board, without writing files.
The install test first backs up the filesystem, performs a clean install, verifies
the resources, and restores saved user files/configuration. The connection-only
variant `TestHardwareUSBInstallConnection` uses the same port/board variables and
tests bootloader/stub access without erasing or writing flash.
The reconnect test prints `READY`; unplug for at least three seconds, then
reconnect. It validates the interpreter and resets without flashing/writing files.
