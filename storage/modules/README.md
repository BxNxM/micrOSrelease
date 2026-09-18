# Modules

Put distributable micrOS modules and their resources here, optionally in
subdirectories. Rebuild the application to include new or changed files.

Both USB install and update copy this directory's contents to the board root,
preserving every relative directory and filename:

- `storage/modules/modules/foo.mpy` → `/modules/foo.mpy`
- `storage/modules/web/index.html` → `/web/index.html`
- `storage/modules/data/nested/settings.json` → `/data/nested/settings.json`

The mapping is generic; new directories need no code or configuration changes.
The top-level `README.md` is excluded. Board-specific `IO_<platform>.py` or
`.mpy` files directly inside `modules/` are filtered using the selected board's
`platform` from `frameworks/<board>/install.json`. Other files are copied for
every board. Explicit `repl.resources` entries can override a destination.

Files are uploaded through temporary paths and verified before they replace
the active copies; `main.py` is transferred last. Updates refresh these resources
even when the firmware version already matches, without reflashing or rewriting
the existing node configuration. The board restarts after a successful transfer.
