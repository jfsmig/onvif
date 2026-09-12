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
| `onvif-protocol` | Bytes on the wire: SOAP 1.2, WS-Security, WS-Addressing, namespaces and `XMLName` tags against `docs/wsdl/*.wsdl`. |
| `cli-ux` | The operator's experience of `bin/onvif-cli`: subcommand naming, help text, the three output shapes, stdout/stderr discipline, and whether `README.md` still describes the tool. |
| `go-architect` | Design coherence and the judgement half of the `AGENTS.md` rulebook, including the deliberate decisions (no error returns from `sdk.Fetch*`, `WaitGroup` over `errgroup`, package names as endpoint routing). |
| `runtime-safety` | Concurrency — goroutine lifecycle, object ownership, shared state, races, bounds, timeouts — and secret hygiene, chiefly the `json:"-"` + `xsd/onvif/redaction_test.go` invariant. |
| `codegen-guard` | Whether a `sdk/profiles/*.profile` line cites a section that says what it claims, against the Profile PDFs in `docs/`. |

**All are read-only** (`disallowedTools: Write, Edit, NotebookEdit`), so they are safe to
run in parallel on one diff — launch them in a single message. They report; the main
session applies the fixes.

### The mechanical rules are not a reviewer's job

`scripts/repo-check.sh` decides, deterministically, every rule a script can decide: the
AGPL notice and the blank line after it, the MIT provenance set, the ban on the standard
logger below `sdk`, every self-reference naming the module path `go.mod` declares,
`calls.txt` 1:1 with the wrappers at 89/79/28/9, and a template change carrying its
regenerated output. A `Stop` hook in `.claude/settings.json` runs it on every
turn.

**Do not ask an agent to check any of those.** An agent re-deriving fixed shell costs a
whole context for an answer a second of `bash` already has. The reviewers above are told
the script covers them and to spend their tokens on judgement instead.

### Before launching the panel

Compute the shared inputs **once**, in this session, and hand them down. Subagents do not
share context — that isolation is what makes them cheap — so anything each would otherwise
re-derive is paid for N times.

```sh
B=<session scratchpad>/review    # the scratchpad this session was given, never the repo
mkdir -p "$B"
git diff --name-only HEAD > "$B/changed.txt"
git diff HEAD              > "$B/diff.patch"
scripts/repo-check.sh      > "$B/mechanical.txt"
```

Then give every agent the same two paths, and tell it plainly:

> The diff is at `$B/diff.patch` and the mechanical results at `$B/mechanical.txt`. Read
> those first. Do not run `git`. Open a source file only when the diff is not enough.

### Which to reach for

Launch only what the changed-path set calls for. The cheapest agent is the one never
launched.

| A changed path matches | Launch |
| --- | --- |
| `networking/ device/ media/ ptz/ event/ xsd/` | `onvif-protocol` |
| `bin/onvif-cli/ README.md` | `cli-ux` |
| `sdk/` | `go-architect` and `runtime-safety` |
| `sdk/profiles/*.profile` | `codegen-guard` |
| `*_auto.go`, `calls.txt`, `bin/onvif-codegen/` | the script alone — no agent unless it fails |
| any new `wg.Go`, or a struct shared across goroutines | `runtime-safety` |
| any new field named password, key, passphrase, secret, token or private key | `runtime-safety`, always |
| a new package, interface, or dependency | `go-architect` |

Non-trivial diffs get whatever the table selects, which is often two or three rather than
all five. Do not invoke one to rubber-stamp work already reviewed by another — they are
deliberately non-overlapping, and each will say so and redirect if handed something outside
its remit.

### Agent files describe remit, never the tree

An agent definition states what the reviewer owns, what it must not touch, and how to
report. It does **not** carry a snapshot of the code: no `path:line` citations, no lists of
known defects, no file or test counts. Every one of those rotted the last time they were
tried, and a stale citation costs more than no citation — the agent follows it, finds
unrelated code, re-derives anyway, and sometimes reports the wrong thing with confidence.

Cite a symbol and let the agent grep for it. Where the tool can be run instead of read, say
so: `cli-ux` is told to run `onvif-cli --help`, which is a few hundred bytes and always
current, rather than read the sixty kilobytes of source behind it.

**A decision belongs in a prompt; a defect does not.** The two were once listed side by
side here, and the difference in how they aged settles it. Of the five "standing
violations" `go-architect` carried, **all five had been fixed** while the prompt still
called them live: `ReadAndParse`'s `tag` parameter is used four times now, `sdk/appliance.go`
handles the `io.ReadAll` error, `xsd/built_in.go` no longer calls `log.Fatalln`,
`FetchStreamURI` sorts its profile tokens, and the "commented-out code" is real code. Of
the deliberate decisions listed alongside them, **every one is still exactly true**. The
mechanism is self-defeating: a defect gets listed *because* someone found it, and finding
it is what leads to fixing it, so the list is most wrong precisely when it was most useful.

So keep the sections that say *this is on purpose, do not "fix" it* — `go-architect`'s
"Deliberate designs", `runtime-safety`'s "Standing architectural limits", `cli-ux`'s "What
is settled, and must stay settled". They stop a reviewer proposing a regression, and they
do not age. Record a live defect where it cannot rot instead: a test, which CI keeps honest
and the fix deletes, or a `TODO` at the site. The repository already had the durable
version of exactly this record — `sdk/stream_uri_test.go` pins the `FetchStreamURI`
credential leak the prompt also listed, and only the test survived the fix.

### A finding arrives as a test

This repository has a habit worth keeping: every bug found so far became a small
regression test citing the WSDL line or specification clause it was verified against —
`media/wsdl_conformance_test.go`, `device/namespace_test.go`,
`networking/redirect_test.go`, `xsd/onvif/redaction_test.go`, `sdk/stream_uri_test.go`.
They are the only automated defence this codebase has.
`bin/onvif-cli/interfaces_test.go` is the odd one out: it pins an operator default rather
than a wire format.

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

That file is tracked, and holds two things and nothing else: the attribution block, and
the `Stop` hook that runs `scripts/repo-check.sh`. Do not reintroduce
`includeCoAuthoredBy`: it is deprecated and can only suppress the default, not reword it.
