#!/usr/bin/env bash
#
# Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <https://www.gnu.org/licenses/>.

# The deterministic half of the review panel.
#
# Every rule here is one an LLM used to re-derive from a prompt on each review, at the
# cost of a whole agent context. They are all mechanical, so they live in shell: the
# reviewers in .claude/agents/ are left with the judgement that actually needs a model.
#
# Default mode is read-only, which is what makes it safe to run from a hook. --regen
# reproduces the CircleCI generate-then-diff gate, which rewrites the working tree.
#
# Modes:
#   (none)    run the read-only checks, exit 1 on failure
#   --regen   also run the generate-then-diff gate; rewrites the working tree
#   --hook    as (none), but exit 2 on failure so Claude Code feeds the output back to
#             the model instead of only printing it. See the Stop hook in
#             .claude/settings.json.

set -uo pipefail

cd "$(git rev-parse --show-toplevel)" || exit 2

REGEN=0
HOOK=0
case "${1:-}" in
  --regen) REGEN=1 ;;
  --hook)  HOOK=1 ;;
esac

# A Stop hook that exits 2 blocks the stop and hands its output to the model. Claude Code
# sets stop_hook_active when it is already continuing from one, and honouring that is what
# keeps a rule this script cannot fix from looping forever.
if [ "$HOOK" -eq 1 ]; then
  if grep -q '"stop_hook_active"[[:space:]]*:[[:space:]]*true' <<< "$(cat)"; then
    exit 0
  fi
fi

FAILED=0
pass() { printf 'PASS  %s\n' "$1"; }
fail() { printf 'FAIL  %s\n' "$1"; FAILED=1; }

# Source files, generated ones included: the licence rules apply to every .go file.
gofiles() { find . -name '*.go' -not -path './.git/*' | LC_ALL=C sort; }

# ---------------------------------------------------------------------------
# 1. The AGPL notice on every .go file.
#
# The notice carries no SPDX identifier, so the prose is the only marker. The window is
# 40 lines rather than the 14 the notice occupies, because the twelve files with an MIT
# provenance block push the AGPL paragraphs further down.
# ---------------------------------------------------------------------------
check_licence_header() {
  local out
  out=$(gofiles | while read -r f; do
    head -40 "$f" | grep -q 'GNU Affero General Public License' || echo "  $f"
  done)
  if [ -n "$out" ]; then
    fail "licence header: missing the AGPL notice"; echo "$out"
  else
    pass "licence header on every .go file"
  fi
}

# ---------------------------------------------------------------------------
# 2. The blank line after the licence notice.
#
# Without it Go takes the notice as the package doc comment, silently. The test anchors
# on the end of the leading comment block, never on the line before `package`: doc.go and
# credentials/resolver.go correctly put a package doc comment there, and a naive check
# reports both as violations.
# ---------------------------------------------------------------------------
check_licence_blank_line() {
  local out
  out=$(gofiles | while read -r f; do
    local n nxt
    n=$(awk 'NR==1 && !/^\/\//{exit} /^\/\//{last=NR; next} {exit} END{print last}' "$f")
    [ -z "$n" ] && continue
    nxt=$(sed -n "$((n + 1))p" "$f")
    [ -n "$nxt" ] && echo "  $f:$((n + 1)): expected a blank line, found: ${nxt:0:60}"
  done)
  if [ -n "$out" ]; then
    fail "licence notice must be followed by a blank line"; echo "$out"
  else
    pass "blank line between the notice and what follows"
  fi
}

# ---------------------------------------------------------------------------
# 3. No standard logger below sdk.
#
# networking, xsd, utils and credentials return errors and log nothing. Comments are
# stripped first: xsd/built_in.go documents the log.Fatalln it no longer calls, and a
# plain grep reports that prose as a breach.
# ---------------------------------------------------------------------------
check_std_logger() {
  local out
  out=$(find networking xsd utils credentials -name '*.go' 2>/dev/null | LC_ALL=C sort | while read -r f; do
    awk -v F="$f" '
      BEGIN { inblock = 0 }
      {
        line = $0
        if (inblock) { if (line ~ /\*\//) inblock = 0; next }
        if (line ~ /\/\*/) { inblock = 1; next }
        sub(/\/\/.*/, "", line)
        if (line ~ /(^|[^.[:alnum:]_])log\.(Print|Fatal|Panic)/)
          printf "  %s:%d:%s\n", F, NR, line
      }' "$f"
  done)
  if [ -n "$out" ]; then
    fail "standard logger used below sdk"; echo "$out"
  else
    pass "no standard logger in networking, xsd, utils, credentials"
  fi
}

# ---------------------------------------------------------------------------
# 4. The MIT provenance block sits on exactly the twelve files AGENTS.md names.
# ---------------------------------------------------------------------------
check_mit_provenance() {
  local expected actual out
  expected=$(printf '%s\n' \
    doc.go imaging/types.go analytics/types.go device/types.go media/types.go \
    ptz/types.go event/types.go event/operation.go networking/networking.go \
    xsd/built_in.go xsd/onvif/onvif.go xsd/iso8601/iso8601_duration.go | LC_ALL=C sort)
  actual=$(grep -rl 'LICENSE.MIT' --include='*.go' . | sed 's|^\./||' | LC_ALL=C sort)
  out=$(diff <(echo "$expected") <(echo "$actual"))
  if [ -n "$out" ]; then
    fail "MIT provenance block is not on the twelve files AGENTS.md names"
    echo "$out" | sed 's/^</  unexpectedly absent: /; s/^>/  unexpectedly present: /'
  else
    pass "MIT provenance block on exactly the twelve named files"
  fi
}

# ---------------------------------------------------------------------------
# 5. calls.txt and the generated wrappers, 1:1.
#
# A matching total says nothing about which names match, so the sets are compared. It is
# a rename or a duplicate this exists to catch.
# ---------------------------------------------------------------------------
check_calls_wrappers() {
  local ok=1 tmp
  tmp=$(mktemp -d) || return
  local -A want=([device]=89 [media]=79 [ptz]=28 [event]=9)
  for p in device media ptz event; do
    grep -vE '^[[:space:]]*(#|$)' "$p/calls.txt" | tr -d '\r' | LC_ALL=C sort > "$tmp/calls"
    ls "$p"/*_auto.go 2>/dev/null | sed "s|$p/||; s|_auto.go||" | LC_ALL=C sort > "$tmp/auto"
    local nc na dup only_calls only_auto
    nc=$(wc -l < "$tmp/calls"); na=$(wc -l < "$tmp/auto")
    dup=$(uniq -d < "$tmp/calls" | tr '\n' ' ')
    only_calls=$(LC_ALL=C comm -23 "$tmp/calls" "$tmp/auto" | tr '\n' ' ')
    only_auto=$(LC_ALL=C comm -13 "$tmp/calls" "$tmp/auto" | tr '\n' ' ')
    [ "$nc" -ne "${want[$p]}" ] && { ok=0; echo "  $p: calls.txt has $nc entries, expected ${want[$p]}"; }
    [ "$na" -ne "${want[$p]}" ] && { ok=0; echo "  $p: $na wrappers, expected ${want[$p]}"; }
    [ -n "$dup" ]        && { ok=0; echo "  $p: duplicated in calls.txt: $dup"; }
    [ -n "$only_calls" ] && { ok=0; echo "  $p: in calls.txt with no wrapper (regenerate): $only_calls"; }
    [ -n "$only_auto" ]  && { ok=0; echo "  $p: wrapper with no calls.txt entry (stale, delete): $only_auto"; }
  done
  rm -rf "$tmp"
  if [ "$ok" -eq 1 ]; then
    pass "calls.txt 1:1 with the wrappers at 89/79/28/9"
  else
    fail "calls.txt and the generated wrappers disagree"
  fi
}

# ---------------------------------------------------------------------------
# 6. A template change must carry its regenerated output.
#
# One template drives every wrapper, so editing it without regenerating is a red build.
# ---------------------------------------------------------------------------
check_template_regen() {
  local changed tmpl auto
  changed=$(git diff --name-only HEAD; git diff --name-only --cached; git ls-files -m)
  changed=$(echo "$changed" | LC_ALL=C sort -u | sed '/^$/d')
  [ -z "$changed" ] && { pass "template change carries its regenerated output (no diff)"; return; }
  tmpl=$(echo "$changed" | grep -E 'bin/onvif-codegen/(sdk|profile_template)\.go$')
  auto=$(echo "$changed" | grep -E '_auto\.go$')
  if [ -n "$tmpl" ] && [ -z "$auto" ]; then
    fail "a generator template changed with no regenerated file in the diff"
    echo "$tmpl" | sed 's/^/  /'
    echo "  run: go generate ./... && git add -A"
  else
    pass "template change carries its regenerated output"
  fi
}

# ---------------------------------------------------------------------------
# 7. The CircleCI gate, made usable on a dirty tree.
#
# CI runs `go generate && git diff --exit-code` against a fresh checkout, so any diff there
# is the generator's. Here the tree usually carries unrelated work in progress, and
# reporting that as generator drift is noise. So the comparison is before-and-after
# generating, not against HEAD: it fails only on what this run actually rewrote.
# ---------------------------------------------------------------------------
snapshot_generated() {
  find . -name '*_auto.go' -not -path './.git/*' -print0 \
    | LC_ALL=C sort -z | xargs -0 md5sum 2>/dev/null
}

check_regen() {
  local before after out
  before=$(snapshot_generated)
  if ! go generate ./... 2>&1 | sed 's/^/  /'; then
    fail "go generate failed"; return
  fi
  after=$(snapshot_generated)

  if [ "$before" = "$after" ]; then
    pass "go generate rewrites nothing -- the committed output is current"
    return
  fi

  fail "go generate rewrote files -- commit them"
  out=$(diff <(echo "$before") <(echo "$after") | grep -E '^[<>]' | awk '{print $3}' | LC_ALL=C sort -u)
  echo "$out" | sed 's/^/  /'
}

# ---------------------------------------------------------------------------
# 5. A package directory is named after its package.
#
# CallMethod routes a request by the last segment of its struct's PkgPath, lowercased, so
# device/, media/, ptz/ and event/ are entries in a routing table rather than organisation.
# Go allows a package clause to differ from its directory, and this repository shipped one
# that did: Imaging/ held `package imaging`, which meant onvif-codegen refused to generate
# into it -- its own directory-matches-package check -- while README promised the wrappers
# were all that was missing.
#
# The rule is mechanical, so it belongs here rather than in a reviewer's head or in a Go test
# that cannot see a directory it was not told about. sdk/routing_test.go covers the other
# half, that the four service names still resolve to an endpoint.
#
# Two exemptions, both real: a main package is named for its command and not its directory,
# and the module root's basename is wherever the repository happens to be cloned.
# ---------------------------------------------------------------------------
check_package_dirs() {
  local out
  out=$(gofiles | xargs -n1 dirname | LC_ALL=C sort -u | while read -r d; do
    [ "$d" = "." ] && continue
    local pkg base
    # head -1 after the grep, not grep -m1 alone: -m1 stops per file, so a directory of
    # ninety files yielded ninety package clauses and every comparison below was against
    # a multi-line string.
    pkg=$(grep -h '^package ' "$d"/*.go 2>/dev/null | head -1 | awk '{print $2}')
    if [ -z "$pkg" ] || [ "$pkg" = "main" ]; then
      continue
    fi
    base=$(basename "$d")
    if [ "$pkg" != "$base" ]; then
      echo "  $d: directory $base holds package $pkg"
    fi
  done)
  if [ -n "$out" ]; then
    fail "every package directory is named after its package"; echo "$out"
  else
    pass "every package directory is named after its package"
  fi
}

# ---------------------------------------------------------------------------
# 8. Every self-reference names the module path go.mod declares.
#
# The module is at a major version, so `/v2` is part of the import path rather than a
# property of the tag. That puts the module path in four kinds of place instead of one:
# go.mod, every import, the //go:generate lines that `go run` the generator, and the two
# templates that *write* imports. A bump that reaches the first three and not the templates
# builds, vets and tests green -- until the next `go generate` writes the old path into the
# new tree, and then nothing builds, reported by `git diff --quiet` as uncommitted generator
# output rather than as the stale path it actually is.
#
# Inside a .go file every occurrence is an import path, a //go:generate argument, or a
# template that writes one, so the whole file is scanned. README.md is not: it also names
# the repository in badge URLs, which are not import paths and must stay unversioned, and one
# sentence deliberately names the v1 path. So only the three shapes there that are certainly
# import paths are checked -- the pkg.go.dev links, the `go get` line, and the example import.
# A stale path in a README breaks no build, which is exactly why it is the one that survives
# a bump and sends a reader to a module that does not exist.
#
# It belongs here rather than in a Go test because two of the four places are a string
# constant and a comment, which the compiler never resolves and no Go test can see.
# ---------------------------------------------------------------------------
check_module_path() {
  local module out
  module=$(awk '$1 == "module" { print $2; exit }' go.mod)
  if [ -z "$module" ]; then
    fail "go.mod declares a module path"; return
  fi
  out=$(gofiles | xargs grep -no 'github\.com/jfsmig/onvif[A-Za-z0-9_./-]*' 2>/dev/null \
    | awk -v m="$module" -F: '{
        if ($3 != m && index($3, m "/") != 1)
          printf "  %s:%s: %s\n", $1, $2, $3
      }')
  out=$out$(grep -noE '(pkg\.go\.dev/|go get |import ")github\.com/jfsmig/onvif[A-Za-z0-9_./-]*' README.md \
    | sed -E 's#:(pkg\.go\.dev/|go get |import ")#:#' \
    | awk -v m="$module" -F: '{
        if ($2 != m && index($2, m "/") != 1)
          printf "  README.md:%s: %s\n", $1, $2
      }')
  if [ -n "$out" ]; then
    fail "a self-reference does not name the module path $module"
    echo "$out"
    echo "  go.mod says $module; the templates in bin/onvif-codegen/, the //go:generate"
    echo "  lines and the README links carry it too, and are bumped with it"
  else
    pass "every self-reference names the module path $module"
  fi
}

run_checks() {
  echo "== licence rules =="
  check_licence_header
  check_licence_blank_line
  check_mit_provenance
  echo "== logging rules =="
  check_std_logger
  echo "== layout rules =="
  check_package_dirs
  check_module_path
  echo "== generator invariants =="
  check_calls_wrappers
  check_template_regen
  if [ "$REGEN" -eq 1 ]; then
    echo "== generate-then-diff (CircleCI gate) =="
    check_regen
  fi
}

# Buffer to a file rather than a command substitution: $(...) runs in a subshell, so the
# FAILED the checks set there would never reach the test below, and every run would report
# success. The hook must stay silent when everything passes -- it fires on every turn, and
# eight PASS lines a turn is noise nobody reads, which is how a real failure gets missed.
BUFFER=$(mktemp) || exit 2
trap 'rm -f "$BUFFER"' EXIT
run_checks > "$BUFFER"

if [ "$FAILED" -eq 0 ]; then
  [ "$HOOK" -eq 1 ] && exit 0
  cat "$BUFFER"
  echo
  echo "all checks passed"
  exit 0
fi

# On failure both modes speak. Hook mode uses stderr and exit 2, which is what makes Claude
# Code hand the findings to the model rather than only printing them.
if [ "$HOOK" -eq 1 ]; then
  {
    echo "scripts/repo-check.sh found breaches of the rules in AGENTS.md:"
    cat "$BUFFER"
    echo
    echo "Fix them before finishing. Re-run: scripts/repo-check.sh"
  } >&2
  exit 2
fi

cat "$BUFFER"
echo
echo "SOME CHECKS FAILED"
exit 1
