# Repository guidance

Read [README.md](README.md) for user workflows and [ARCHITECTURE.md](ARCHITECTURE.md)
for implementation boundaries. Verify behavior against code; keep all three
documents compact and update the relevant one when behavior changes.

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
