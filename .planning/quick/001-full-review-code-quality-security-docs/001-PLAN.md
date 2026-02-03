---
phase: 001-full-review-code-quality-security-docs
plan: 001
type: execute
wave: 1
depends_on: []
files_modified: [.planning/quick/001-full-review-code-quality-security-docs/REPORT.md]
autonomous: true
must_haves:
  truths:
    - "Automated linting and testing checks have run"
    - "Security vulnerabilities (if any) are identified"
    - "Documentation gaps are identified"
    - "Structured report exists with actionable recommendations"
  artifacts:
    - path: ".planning/quick/001-full-review-code-quality-security-docs/REPORT.md"
      provides: "Full code quality and security review"
  key_links: []
---

<objective>
Perform a comprehensive code quality, security, and documentation review of the gaz framework.
Purpose: Identify improvements, security risks, and documentation gaps to ensure production readiness.
Output: A structured REPORT.md with actionable recommendations.
</objective>

<execution_context>
@AGENTS.md
@.planning/PROJECT.md
@Makefile
</execution_context>

<tasks>

<task type="auto">
  <name>Task 1: Run automated analysis and data gathering</name>
  <files>.</files>
  <action>
    Run the following analysis steps to gather data for the report:
    1. Run `make lint` to check for code quality issues (capture output).
    2. Run `go test -race -cover ./...` to check reliability and coverage (capture output).
    3. Run `go vet ./...` for standard static analysis.
    4. Search for `TODO`, `FIXME`, and `XXX` tags excluding `_tmp` directories.
    5. List all exported functions without comments (using grep/regex on `*.go` files).
    6. Read `go.mod` to review dependency tree size and versions.
    
    Ignore any `_tmp*` directories during search operations.
  </action>
  <verify>
    Ensure analysis commands complete and outputs are available in the conversation history for synthesis.
  </verify>
  <done>
    Raw data for quality, security, and testing status is collected.
  </done>
</task>

<task type="auto">
  <name>Task 2: Synthesize findings and generate report</name>
  <files>.planning/quick/001-full-review-code-quality-security-docs/REPORT.md</files>
  <action>
    Based on the output from Task 1 and a manual review of key files (DI container, worker pool, config loading):
    
    1. Analyze the `di`, `worker`, and `config` packages for complexity and safety gaps.
    2. Create a structured report at `.planning/quick/001-full-review-code-quality-security-docs/REPORT.md` with:
       - **Executive Summary**: Overall health score (0-100%).
       - **Code Quality**: Lint results, complexity hotspots, test coverage gaps.
       - **Security**: Dependency risks, race conditions, unsafe pointer usage (if any).
       - **Documentation**: Missing comments on exported API, README clarity.
       - **Performance**: Potential bottlenecks in hot paths (DI resolution, event bus).
       - **Recommendations**: Prioritized list of actions (High/Medium/Low).
  </action>
  <verify>
    Check that `.planning/quick/001-full-review-code-quality-security-docs/REPORT.md` exists and contains all required sections.
  </verify>
  <done>
    A comprehensive, actionable review report is generated.
  </done>
</task>

</tasks>

<output>
After completion, the user will have a detailed `REPORT.md` to guide future improvements.
</output>
