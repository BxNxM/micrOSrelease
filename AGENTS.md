# Repository guidance

Read [README.md](README.md) for user workflows and [ARCHITECTURE.md](ARCHITECTURE.md)
for implementation boundaries. Verify behavior against code; keep all three
documents compact and update the relevant one when behavior changes. Keep README
limited to setup, everyday controls, and essential recovery guidance; put design
details in ARCHITECTURE and coding-agent constraints here.

## Structure

- Keep `main.go` as the composition root and feature logic in `internal/usb`,
  `internal/micropython`, `internal/network`, or `internal/storage`.
- Preserve native Go workflows and the single executable with embedded assets.
- Keep TUI state/events/commands in `internal/tui`; put screen keys in
  `*_navigation.go`, screen composition in `views/*_view.go`, and reusable
  rendering in `widgets/*_widget.go`. Keep `view.go` limited to snapshot/routing.
- Views/widgets consume read-only `widgets.State`; no service calls, serial I/O,
  network requests, or state mutation during rendering. Run I/O through commands
  and return messages; preserve cancellation and streamed progress.
- Depend on feature interfaces and existing injectable hardware factories.
  Extend focused files rather than growing a monolithic model or view.

## Invariants

- Validate payloads, chip, and flash capacity before destructive writes. Updates
  must finish a verified, durable filesystem backup before firmware replacement;
  never skip hidden/custom files or bypass transfer/backup limits.
- Preserve user state, board-specific resource filtering, verified temporary-file
  transfers, `main.py` ordering, reconnect identity checks, and watchdog/reset
  behavior. Version matches still require chip validation and resource refresh.
- Keep embedded `storage/` distinct from host `internal/storage` data. Do not
  commit device credentials, backups, or machine-specific source paths.
- Use `make mr` only for requested asset refreshes; it copies from an external
  micrOS checkout. Rebuild `dist/` binaries only when distribution changes are
  requested. Consult the two `storage/` READMEs before changing release layouts.

## Shell behavior to preserve

- Device details lists Shell below Web UI as `IP:port`; offline nodes show `-`
  and cannot open Shell. Default to Shell when no Web UI URL is available.
- Authenticate interactively with masked input; never autofill the discovery
  password or retain password submissions in history. Verify UID after login.
- Stream replies immediately; only a complete trailing node prompt enables the
  next command. Track `[password]` / `[configure]` prefixes and retain partial
  output on failure. Do not retry user commands automatically.
- `exit` sends before disconnecting; Esc/Ctrl+C disconnect and return to details.
  Keep cancellation and stale-session guards; rendering never accesses a client.
- Up/Down recalls session-only commands and restores the draft; Left/Right scrolls
  output, including during replies. New commands return to the latest output.
- Preserve bold prompts, normal command text, grey responses, and blue status
  through scrollback and wrapping. Sanitize server terminal controls and retain
  bounded history/transcripts; never persist them without an explicit request.

## Application update invariants

- Check releases in the background; install only after `u` on Nodes. Offer any
  different microsctl version, and never select a fallback OS/architecture.
- Nodes has a hidden `x` toggle for update mode. Keep it out of user hints/README.
  Update mode appears automatically for a different version; manual mode also
  permits a real same-version reinstall with `u`, using a successfully checked
  manifest. Keep the base version banner visible when update mode is off.
- Root `MANIFEST.yaml` is the sole application version source. Bump its
  microsctl version for releases; publish it with matching `dist/` builds.
- Keep the release URL (including branch) in `microsctl.url` in `MANIFEST.yaml`
  and preserve it during generation. Use the fetched manifest's URL for downloads;
  fetch latest branch contents without commit lookup or pinning.
- Self-update concerns only microsctl; bundled firmware metadata is informational.
  Do not validate binary hashes or executable formats. Keep bounded downloads,
  backup and rollback on replacement failure. Never self-update during USB work
  or restart before update completion and terminal cleanup.
- Preserve arguments/environment/working directory on restart. Test using local
  HTTP fixtures and temporary executables; never replace the developer's tool or
  publish GitHub changes as part of routine tests.

## Checks

- Format changed Go files with `gofmt`. Run relevant tests for code changes;
  use `go test ./...` and `go vet ./...` for changes spanning packages.
- Add focused regression coverage for changed behavior using existing fakes.
  For documentation-only edits, check implementation accuracy, links, and diff.
- Keep `MICROS_TEST_*` hardware variables unset for routine tests. Live discovery
  resets boards; install/update can erase them. Run hardware tests only when
  authorized for the selected device, with persistent backups for update tests.
- Review `git diff --check` and the final diff; report checks and any unverified
  behavior. Keep README usage-focused and technical details in ARCHITECTURE.
