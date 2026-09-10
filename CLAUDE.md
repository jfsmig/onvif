# CLAUDE.md

The instructions for this repository are agent-agnostic and live in @AGENTS.md.
Read them first: layout, commands, the code generators, the licence-header rules
and the conventions all apply in full.

## Claude-specific directives

### The review panel

CI never sees a camera. `.circleci/config.yml` builds, vets, tests and checks that the
generated files are current, but nothing exercises the wire protocol — `AGENTS.md` says
it outright: *"a change is verified by building and by running the CLI against a real
camera."* Correctness therefore rests on review, so this repository keeps five
specialised reviewers in `.claude/agents/`:

| Agent | Owns |
| --- | --- |
| `onvif-protocol` | Bytes on the wire: SOAP 1.2, WS-Security, namespaces and `XMLName` tags against `docs/wsdl/*.wsdl`, Profile manifests against the specification PDFs. |
| `cli-ux` | The operator's experience of `bin/onvif-cli`: subcommand naming, help text, machine-parsable stdout, stdout/stderr discipline, and whether `README.md` still describes the tool. |
| `go-architect` | The `AGENTS.md` rulebook and design coherence, including the deliberate decisions (no error returns from `sdk.Fetch*`, `WaitGroup` over `errgroup`, package names as endpoint routing). |
| `runtime-safety` | Concurrency — goroutine lifecycle, object ownership, shared state, races, bounds, timeouts — and secret hygiene, chiefly the `json:"-"` + `xsd/onvif/redaction_test.go` invariant. |
| `codegen-guard` | The generator invariants: no hand-edited `*_auto.go`, `calls.txt` 1:1 at 89/79/28/9, template edits followed by regeneration, manifest lines carrying a spec citation. |

**All five are read-only** (`disallowedTools: Write, Edit, NotebookEdit`), so they are
safe to run in parallel on one diff — launch them in a single message. They report; the
main session applies the fixes.

### Which to reach for

By what the change touches:

- `networking/`, `device/`, `media/`, `ptz/`, `event/`, `xsd/` → `onvif-protocol`, and
  `codegen-guard` if any `calls.txt`, `types.go` or `*_auto.go` moved
- `bin/onvif-cli/`, `README.md` → `cli-ux`
- `sdk/` → `go-architect` and `runtime-safety`
- `bin/onvif-codegen/`, `sdk/profiles/` → `codegen-guard`, then `onvif-protocol` for the
  citations
- any new `wg.Go`, or a struct shared across goroutines → `runtime-safety`
- any new field named password, key, passphrase, secret, token or private key →
  `runtime-safety`, always
- a new package, interface, or dependency → `go-architect`

Non-trivial diffs get the whole panel. Do not invoke one to rubber-stamp work already
reviewed by another — they are deliberately non-overlapping, and each will say so and
redirect if handed something outside its remit.

### A finding arrives as a test

This repository has a habit worth keeping: every bug found so far became a small
regression test citing the WSDL line or specification clause it was verified against —
`media/wsdl_conformance_test.go`, `device/namespace_test.go`,
`networking/redirect_test.go`, `xsd/onvif/redaction_test.go`, `sdk/ptz_token_test.go`.
Fourteen such files across ten packages, and they are the only automated defence this
codebase has. `bin/onvif-cli/interfaces_test.go` is the odd one out: it pins an operator
default rather than a wire format.

So a reviewer's finding is not finished as prose. Where it can be pinned, it comes with
the test written out, in the style of the target package, with a comment saying what the
slip was and what it was verified against. Apply the test with the fix, in the same
change.

### The attribution trailer is configured, not remembered

`AGENTS.md` states the rule for every agent: an LLM-assisted commit ends with
`Assisted-By: <tool>`, and never `Co-authored-by:` or `Authored-by:`. For Claude Code it
is mechanical — `.claude/settings.json` sets `attribution.commit` and `attribution.pr` to
`Assisted-By: Claude Code`, which replaces both shipped defaults, the `Co-Authored-By:`
trailer and the "Generated with Claude Code" pull-request footer.

That file is tracked, and holds the attribution block and nothing else. Do not reintroduce
`includeCoAuthoredBy`: it is deprecated and can only suppress the default, not reword it.
