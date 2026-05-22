# Context

## Domain Glossary

### Lifecycle plan

A lifecycle plan is the application runtime plan derived from the DI container's service graph. It owns the policy for which services participate in lifecycle hooks, which services are excluded because another runtime module owns them, and the startup and shutdown order used by the App runtime.

The lifecycle plan is distinct from lifecycle execution. Planning decides what should happen and in what order; execution starts, stops, rolls back, logs, and enforces context deadlines.
