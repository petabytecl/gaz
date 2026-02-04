# Phase 44: Config File CLI Flag - Context

**Gathered:** 2026-02-04
**Status:** Ready for planning

<domain>
## Phase Boundary

Enable configuration to register a `--config` CLI flag for specifying a config file path. When the flag is not provided, auto-search standard locations for a default config file. Integrate with existing Viper-based configuration system.

</domain>

<decisions>
## Implementation Decisions

### Flag behavior
- Flag is **optional** — app runs without config file
- Flag name: `--config` (no short flag)
- When `--config` is NOT provided: auto-search for default config file
- Search locations (in order): current working directory, then XDG config dir (`~/.config/appname/`)
- Default filename: `config.*` (any supported extension)

### File format support
- Use **Viper's default formats** — no extra format dependencies
- Auto-detect format from file extension (`.yaml`, `.yml`, `.json`, `.toml`, etc.)
- **Single file only** — no merging multiple config files

### Error handling
- If `--config` is provided but file doesn't exist: **exit with error**
- If no `--config` and auto-search finds nothing: **silent continue** (no config is fine)
- If file has invalid syntax (malformed YAML, etc.): **exit with error** showing parse location
- If file has unrecognized keys: **exit with error** (strict mode) — prevents typos in config

### Override precedence
- Standard order: **CLI flags > env vars > config file > defaults**
- Auto-bind environment variables to config keys (e.g., `GAZ_LOG_LEVEL` → `log.level`)
- Env var prefix is **configurable** via option
- Default prefix: `GAZ_` (if no custom prefix set)

### Claude's Discretion
- Exact XDG path resolution implementation
- How to derive app name for XDG subdir (from cobra command name)
- Viper setup details and config struct binding

</decisions>

<specifics>
## Specific Ideas

- Keep it simple — this is a common pattern, follow Viper conventions
- Strict unknown key validation catches config typos early
- The configurable env prefix allows apps to use their own namespace (e.g., `MYAPP_`)

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 44-config-file-cli-flag*
*Context gathered: 2026-02-04*
