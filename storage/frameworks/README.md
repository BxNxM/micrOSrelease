# Frameworks

Put firmware binaries in board folders, for example:
`esp32/micrOS-esp32-1.28.0-3.6.0-0.bin`.
Rebuild the application to bundle new or changed files into the executable.

Each installable board folder also needs an `install.json`:

```json
{
  "version": 1,
  "protocol": "esp-rom",
  "chip": "esp32",
  "platform": "esp32",
  "initial_baud": 115200,
  "flash_baud": 460800,
  "flash_offset": "0x1000",
  "erase_flash": true,
  "compress": true,
  "reset_mode": "auto",
  "reset_after_flash": true,
  "flash_mode": "keep",
  "flash_frequency": "keep",
  "flash_size": "keep",
  "hint": "Optional bootloader instructions shown to users.",
  "repl": {
    "baud": 115200,
    "connect_timeout_seconds": 10,
    "reconnect_timeout_seconds": 0,
    "config_paths": [
      "/config/node_config.json",
      "/node_config.json"
    ],
    "restore_path": "/config/node_config.json",
    "resources": [
      {
        "source": "modules/example.mpy",
        "target": "/modules/example.mpy"
      }
    ]
  }
}
```

Supported ESP chips are ESP8266, ESP32, ESP32-S2/S3, ESP32-C2/C3/C5/C6,
ESP32-H2, and ESP32-P4-Rev1. Reset mode is `default`, `usb-jtag`, `no-reset`,
or `auto`. Empty or `keep` flash header settings preserve values in the image.

The `repl` object is optional and uses the connection defaults shown above.
`connect_timeout_seconds` bounds each attempt. `reconnect_timeout_seconds: 0`
waits until the operation is cancelled, allowing a manual USB unplug/replug; set
a positive value to impose a total reconnect deadline for automation. USB serial
identity is captured before flashing so renamed ports can be followed safely.
During both install and update, files embedded under `storage/modules/` are
copied automatically to the board root with their relative paths intact. For
example, `storage/modules/modules/foo.mpy` becomes `/modules/foo.mpy` and
`storage/modules/web/index.html` becomes `/web/index.html`; arbitrary nested
directories work the same way. Of the `IO_<platform>.py` or `.mpy` files in
`storage/modules/modules/`, only those selected by `platform` are copied.
Explicit `resources` add files or override
an automatically generated destination. The complete device filesystem is archived on the host before firmware replacement.
Hidden configuration, custom modules and user files are restored; release resource
destinations are refreshed. A host backup directory is mandatory for firmware
replacement, and backup failures stop before erase. Files are transferred through
temporary paths and verified before replacing each destination. `main.py`
resources are always transferred last. Updates skip flashing only when the connected chip, MicroPython version, and
micrOS version match; they still refresh resources and restart the application. `reset_after_flash` must
be true when installing resources or updating, so MicroPython can start before
REPL transfer.
