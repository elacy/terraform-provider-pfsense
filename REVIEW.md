# Code Review Rules

This file is the rubric the reviewer agent grades every PR against, and the
coder agent reads before writing code. Both agents must follow it exactly.
Edit this file to change how reviews behave in this repo — do not ask agents
to "just use best practices."

## Blocking rules (auto-reject — request changes)

1. **Security** — hardcoded secrets/API keys, shell injection, SQL injection,
   path traversal, `eval`/`exec` on user input, unauthenticated access to
   protected endpoints. Secret material in provider resource schemas must be
   marked `Sensitive: true` (e.g. pre-shared keys, private keys, client
   secrets).
2. **Correctness** — logic contradicts the stated intent, off-by-one/race
   conditions, missing error handling on I/O/network/DB calls, code that
   cannot possibly do what the PR description claims. For this provider:
   natural-key ID resolution must be correct (see the comment header in
   `internal/provider/helpers.go`), Delete must tolerate already-absent
   objects, and every mutation on a deferred-apply model
   (firewall/interface/routing/dhcp/dns/vpn) must pass `apply`.
3. **Tests / build** — the code must compile: `gofmt -l .` empty, `go build
   ./...` and `go vet ./...` clean, `go test ./...` passes. A PR whose touched
   files break the build is an automatic request-changes regardless of logic.

## Style / convention rules (warning-level, fix unless justified)

4. Match the repo's existing conventions: formatting, naming, error style,
   import order. The resource pattern is defined by
   `internal/provider/firewall_alias_resource.go` and
   `internal/provider/nat_resources.go` — new resources must follow it
   (model struct, `Schema`, Create/Read/Update/Delete/ImportState, `findByKey`
   family helpers, `applyNow` where required).
5. No dead code: no commented-out blocks, debug prints, or leftover TODOs in
   touched files.
6. Public functions/classes get docstrings/comments explaining *why*, not just
   *what*.
7. Changes are scoped to the task — no drive-by refactors of unrelated code.

## Approve criteria

Approve only when:
- No blocking rules are violated.
- Warnings are resolved or explicitly justified in a reply to the reviewer.
- The diff is small enough to fully review (if too large, request changes and
  ask for it to be split).

## Verdict format

The reviewer returns exactly one of:

- `approve` — no blocking issues; warnings resolved/justified.
- `request-changes` — at least one blocking issue or unaddressed warning,
  with a numbered list of findings: `file:line — severity — what's wrong —
  what to change`.

Do not approve with a caveat. If there is anything blocking, it is
`request-changes`.