# ![logo](./media/logo_mini.png) microsctl

Discover [micrOS](https://github.com/BxNxM/micrOS) nodes, run commands, and install
or update ESP boards over USB from one terminal app. Firmware and resources are
bundled and self contained.

![ESP32 verified target](https://img.shields.io/badge/ESP32-verified-brightgreen)
![ESP32-C6 verified target](https://img.shields.io/badge/ESP32--C6-verified-brightgreen)
![ESP32-S3 verified target](https://img.shields.io/badge/ESP32--S3-verified-brightgreen)
![ESP32-C3 verified target](https://img.shields.io/badge/ESP32--C3-verified-brightgreen)
![ESP32-S3-Octo verified target](https://img.shields.io/badge/ESP32--S3--Octo-coming--soon-yellow)
![ESP32-S31 verified target](https://img.shields.io/badge/ESP32--S31-tbd-yellow)

![microsctl TUI](media/TUI.png)

## Install and run

Run in the directory where you want the executable:

```sh
curl -fsSL https://raw.githubusercontent.com/BxNxM/micrOSrelease/main/dist/install.sh | sh
./microsctl
```

Supports macOS ARM64, Linux x64/ARM64 (including 64-bit Raspberry Pi OS), and
Windows x64. On Windows, run the installer in Git Bash, MSYS2, or Cygwin, then
start `./microsctl.exe`.

## Update microsctl

The top line checks GitHub for a different microsctl version. Press **u** when
an update is offered to install the binary for your platform and restart
automatically. **Esc** cancels a download; **u** retries a failed check or update.
Your current command-line settings are kept.

## Find and use nodes

Use the arrow keys to select a card and **Enter** for details. **r** refreshes;
scans also run every five minutes. Saved nodes appear immediately, while
localhost and AP mode nodes appear when reachable.

In details, open **Web UI** (**o**), choose **Shell**, or remove the node from the
local cache. Removing a node does not erase it; a later scan can rediscover it.
Shell shows `-` for offline nodes.

In **Shell**, type a command and press **Enter**. Replies stream as they arrive;
the bold prompt means the node is ready again. Enter your password when asked.

| Key | Action |
| --- | --- |
| **↑ / ↓** | Recall commands from this session; passwords are excluded. |
| **← / →** | Scroll older/newer output. |
| **Ctrl+U** | Clear the input. |
| **Esc**, **Ctrl+C**, or `exit` | Close Shell and return to details. |

To reconnect after an error, leave Shell and open it again.
Outside Shell, **Esc** goes back and **q** quits when idle.

## Install or update over USB

1. Connect the board and open **USB Tools**.
2. Choose **Discovery** to identify the board and select matching firmware.
   This resets the board. **USB Scan** only lists ports.
3. Check the device, board, and firmware. **[ / ]** switches devices;
   **← / →** switches boards; **f** opens the firmware picker.
4. Choose **Install micrOS** or **Update micrOS**, then confirm with **y** or
   **Enter**. **n** or **Esc** cancels the confirmation.

**Install erases the device.** **Update preserves configuration and user files**,
while replacing bundled files. Before reflashing, Update saves a verified
filesystem backup. Backup failures or size limits (1 MiB/file, 64 MiB total)
stop the update before erase. Keep the backup until you have checked the board.
Binary OTA firmware updates are not yet supported.

Start Update with MicroPython running normally. For Install on some native-USB
boards, hold **BOOT** while connecting to enter programming mode. After flashing,
release BOOT and reconnect normally when prompted. Keep USB connected otherwise.

After a successful Install:

1. Connect to the **node01** Wi-Fi network (default password: `ADmin123`).
2. Press **Esc** to return to Nodes.
3. Open the **AP mode** device and configure it.

At **Reconnect to MicroPython REPL**, the app waits for the board and resumes
without reflashing; **Ctrl+C** cancels and exits. Reconnect to the same physical
USB socket on macOS. On other systems, use the original port unless the board
has a stable USB serial identity.

If an operation fails, follow the error shown in its panel. If only the final
reset fails after files are verified, reboot normally without holding BOOT;
another install is not needed.

## Settings and backups

```sh
./microsctl --cidr 10.0.1.0/24  # Choose an IPv4 scan range (/24 or smaller)
./microsctl --data-dir ./data   # Choose where to store cache and backups
MICROS_PASSWORD='your-password' ./microsctl  # Identify protected nodes during scans
./microsctl --list-assets      # List bundled firmware and files
```

Shell asks for passwords interactively, without autofill. By default, cache and backups
live in the platform's user configuration directory under `microsctl`
(on macOS: `~/Library/Application Support/microsctl`). Backups are in `backups/`
and may contain credentials; keep them private.

For development, asset refreshes, and tests, see [ARCHITECTURE.md](ARCHITECTURE.md).
Coding-agent guidance and behavior to preserve are in [AGENTS.md](AGENTS.md).

## Build from source

With Go 1.25 or newer installed, run:

```sh
git clone https://github.com/BxNxM/micrOSrelease.git
cd micrOSrelease
go build -o microsctl .
./microsctl
```

On Windows, build with `go build -o microsctl.exe .` and run `./microsctl.exe`.
