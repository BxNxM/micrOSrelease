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
| `internal/tui/views/*_view.go` | Compose the Nodes, details, USB actions, and firmware screens. |
| `internal/tui/widgets/*_widget.go` | Reusable rendering; `state.go` holds the rendering snapshot, `styles.go` the styles. |
| `internal/usb/` | Inventory, board/image selection, ESP probing, install/update orchestration, backups, reconnect. |
| `internal/micropython/` | Raw/raw-paste REPL, runtime inspection, verified file transfer, watchdog maintenance. |
| `internal/network/` | Authenticated TCP client, discovery/status, endpoint identity and Web UI URLs. |
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

Startup loads cached nodes, inventories USB ports/assets, and begins network
discovery. A single five-minute timer chain refreshes nodes without overlapping
scans. Network discovery uses 32 workers on TCP 9008, validates `hello`, then reads
version and feature flags. It checks configured/private IPv4 ranges, saved
addresses, localhost, and AP mode. Observations merge by UID; reachable special
endpoints take precedence over LAN aliases and are never displayed from cache
alone. Cache writes use a temporary file, sync, and rename.

## USB lifecycle and preservation

Inventory lists serial ports and embedded `.bin`/`.uf2` images; installation
currently supports the `esp-rom` protocol. Discovery probes ESP bootloaders and
resets boards; a plain scan does not. Probing uses UART reset sequences for known
USB-to-UART bridges (with a port-name fallback), and auto reset for native
Espressif USB and unknown endpoints. Board/version metadata comes from firmware
filenames; board behavior comes from `frameworks/<board>/install.json`.
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
64 MiB total abort the backup. Archives are privately created, synced, and renamed.

After flashing, reconnect follows stable USB serial identity across port changes
(otherwise the original port), validates chip/runtime, and defaults to waiting
until cancellation. Restore preserves user files except release destinations and
configured node-config paths; configuration is written to `restore_path` with the
new micrOS version, then release resources are copied. The already-current path
does not rewrite configuration. File writes verify temporary uploads before
replacement; release `main.py` files go last. REPL maintenance feeds a temporary
watchdog, cleared by the final hardware reset. Resource copies stream the current
destination and file count through stage details before each verified upload;
the TUI replaces a single detail line, truncating it to the panel width.
Pre-write REPL failures attempt to
restart the unchanged application; post-flash update errors include the backup path.

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
make build                           # macOS ARM64, Linux x64, Windows x64
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

MICROS_TEST_PORT=/dev/cu.usbmodem2101 \
MICROS_TEST_BACKUP_DIR=/absolute/path/to/backups \
go test ./internal/usb -run '^TestHardwareUSBUpdate$' -v -count=1 -timeout=16m

MICROS_TEST_RECONNECT_PORT=/dev/cu.usbmodem2101 \
MICROS_TEST_RECONNECT_CHIP=esp32c6 \
go test ./internal/usb -run '^TestHardwareUSBReconnect$' -v -count=1 -timeout=16m
```

`MICROS_TEST_FLASH=1` forces erase/restore in the update test even when current.
The probe test only identifies and resets the selected board, without writing files.
The reconnect test prints `READY`; unplug for at least three seconds, then
reconnect. It validates the interpreter and resets without flashing/writing files.
