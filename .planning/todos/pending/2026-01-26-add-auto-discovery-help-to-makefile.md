---
created: 2026-01-26T15:52
title: Add auto-discovery help to Makefile
area: tooling
files:
  - Makefile:5-15
---

## Problem

The Makefile help target (lines 5-15) manually echoes each target description. As targets are added, removed, or renamed, the help output must be updated manually — easy to forget, leads to stale documentation.

The `## description` annotations already exist on each target (e.g., `test: ## Run tests`) but aren't being used for auto-discovery.

## Solution

Replace manual echo statements with awk/grep pattern that extracts targets with `##` annotations and formats them automatically. Common pattern used by AWS, Hashicorp, and many OSS projects:

```makefile
help: ## Show this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-12s %s\n", $$1, $$2}'
```

This ensures help is always in sync with actual targets.
