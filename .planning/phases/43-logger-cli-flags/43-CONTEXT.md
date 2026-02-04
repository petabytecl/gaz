# Phase 43: Logger CLI Flags - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Enable logger module to register CLI flags for runtime configuration. Users can configure log format, level, source inclusion, and output destination via CLI flags. This follows the established ConfigProvider pattern from Phase 41-42.

</domain>

<decisions>
## Implementation Decisions

### Flag naming
- All flags use `--log-` prefix: `--log-level`, `--log-format`, `--log-output`, `--log-add-source`
- No short flags (e.g., no `-v` for verbose) — avoid conflicts with app-specific flags
- Default format is `text` (development-oriented)
- `--log-add-source` enables file:line inclusion in logs (default false)

### Level handling
- Named levels only: `debug`, `info`, `warn`, `error`
- Case-sensitive: lowercase only (no DEBUG, Info, etc.)
- Default level is `info`
- Invalid level (e.g., `--log-level=trace`) errors and exits with clear message about valid options

### Output destination
- `--log-output` flag accepts: `stdout`, `stderr`, or file path
- Default is `stdout`
- File output appends (does not overwrite)
- If file cannot be opened (permissions, missing directory): warn and fall back to stdout

### Format behavior
- Supported formats: `text` and `json`
- Default format: `text` (always, no TTY auto-detection for format choice)
- Colors in text format: auto-enabled for TTY output, disabled when piped
- Invalid format (e.g., `--log-format=yaml`) errors and exits with clear message

### Claude's Discretion
- Error message wording for invalid flags
- Implementation details of file handle management
- Whether to add a `--log-no-color` override flag
- Time format in text output

</decisions>

<specifics>
## Specific Ideas

- Consistent with existing server flag patterns (`--grpc-port`, `--http-port`)
- Follow ConfigProvider pattern established in Phase 41-42
- Text format uses existing tint handler for colored output
- Fail-fast on invalid level/format, but graceful degradation on file output errors

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 43-logger-cli-flags*
*Context gathered: 2026-02-04*
