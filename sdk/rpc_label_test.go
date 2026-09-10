// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>.

package sdk

// The Fetch* methods return a struct and not (struct, error): AGENTS.md makes sdk swallow
// every per-call error on purpose, so that one unsupported operation cannot lose a whole
// dump. The consequence is that the trace line is the *only* thing an operator ever sees
// about a failed sub-call, and two ways of getting it wrong had both happened:
//
//   - eight sites named the wrong operation, four of them naming an operation that exists
//     in no calls.txt at all -- sdk/media.go logged "GetAnalyticsConfiguration" and
//     "GetAnalyticsConfigurations" for GetVideoAnalyticsConfiguration(s) and, in
//     FetchMediaAudio, for GetAudioOutputs and GetAudioOutputConfigurations;
//   - three sites in loadProfilePTZ (GetStatus, GetPresets, GetPresetTours) dropped the
//     error with no else branch at all, while the two configuration fetches beside them
//     logged.
//
// Both tests below are written against the AST rather than against a list of known sites,
// so they constrain the next operation added as much as the ones that were wrong.

import (
	"bufio"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// callWrapper matches the generated per-operation wrapper, media.Call_GetProfiles and
// friends. The submatch is the ONVIF operation name.
var callWrapper = regexp.MustCompile(`^Call_([A-Za-z0-9_]+)$`)

// sdkFiles parses the hand-written sources of this package. Generated and test files are
// skipped: profile_S_auto.go delegates without logging, which is correct for it.
func sdkFiles(t *testing.T) (*token.FileSet, []*ast.File) {
	t.Helper()

	names, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}

	fset := token.NewFileSet()
	var files []*ast.File
	for _, name := range names {
		if strings.HasSuffix(name, "_test.go") || strings.HasSuffix(name, "_auto.go") {
			continue
		}
		f, err := parser.ParseFile(fset, name, nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("ParseFile %s: %v", name, err)
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		t.Fatal("no hand-written source parsed; the glob or the package layout moved")
	}
	return fset, files
}

// rpcLabel returns the single Str("rpc", …) literal reachable inside node, and whether
// exactly one was found. The call sits at the end of a zerolog chain, so it is looked for
// anywhere below rather than at a fixed depth.
func rpcLabel(node ast.Node) (string, bool) {
	var found []string
	ast.Inspect(node, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || len(call.Args) != 2 {
			return true
		}
		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok || sel.Sel.Name != "Str" {
			return true
		}
		key, ok := call.Args[0].(*ast.BasicLit)
		if !ok || key.Kind != token.STRING {
			return true
		}
		if k, err := strconv.Unquote(key.Value); err != nil || k != "rpc" {
			return true
		}
		if value, ok := call.Args[1].(*ast.BasicLit); ok && value.Kind == token.STRING {
			if v, err := strconv.Unquote(value.Value); err == nil {
				found = append(found, v)
			}
		}
		return true
	})
	if len(found) != 1 {
		return "", false
	}
	return found[0], true
}

// wrappedOperation returns the ONVIF operation name if expr is a call to a generated
// Call_<Op> wrapper.
func wrappedOperation(expr ast.Expr) (string, bool) {
	call, ok := expr.(*ast.CallExpr)
	if !ok {
		return "", false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return "", false
	}
	if m := callWrapper.FindStringSubmatch(sel.Sel.Name); m != nil {
		return m[1], true
	}
	return "", false
}

// assignedOperation returns the operation name if stmt is `_, err := <svc>.Call_<Op>(…)`.
func assignedOperation(stmt ast.Stmt) (string, bool) {
	assign, ok := stmt.(*ast.AssignStmt)
	if !ok || len(assign.Rhs) != 1 {
		return "", false
	}
	return wrappedOperation(assign.Rhs[0])
}

// eachCallSite calls visit for every RPC in this package, in either of the two shapes it is
// written in, handing over the block that must carry the trace label:
//
//	if _, err := <svc>.Call_<Op>(…); err == nil { … } else { <label here> }
//
//	_, err := <svc>.Call_<Op>(…)
//	if err != nil { <label here>; return }
//
// The second shape appeared with the fan-out, where a failed lead call has to abandon the
// whole closure rather than skip one field. Both are checked, so restructuring a call site
// cannot quietly drop it out of this test's reach.
func eachCallSite(files []*ast.File, visit func(pos ast.Node, logBlock ast.Node, operation string)) {
	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Shape one: the call is the if statement's own initialiser.
			if stmt, ok := n.(*ast.IfStmt); ok && stmt.Init != nil {
				if operation, ok := assignedOperation(stmt.Init); ok {
					visit(stmt, stmt.Else, operation)
					return true
				}
			}

			// Shape two: the call is a statement, and the next statement handles its error.
			block, ok := n.(*ast.BlockStmt)
			if !ok {
				return true
			}
			for i, stmt := range block.List {
				operation, ok := assignedOperation(stmt)
				if !ok {
					continue
				}
				// Already covered by shape one when the call sits in an if initialiser.
				if i+1 >= len(block.List) {
					visit(stmt, nil, operation)
					continue
				}
				guard, ok := block.List[i+1].(*ast.IfStmt)
				if !ok {
					visit(stmt, nil, operation)
					continue
				}
				visit(stmt, guard.Body, operation)
			}
			return true
		})
	}
}

// propagatesError reports whether node hands its error back to the caller instead of
// logging it. load() does exactly that -- it returns (Appliance, error), so a failed probe
// is the caller's business -- and that is a complete way of handling the error, not a
// dropped one. Only Fetch*, which returns no error, has to log.
func propagatesError(node ast.Node) bool {
	found := false
	ast.Inspect(node, func(n ast.Node) bool {
		if ret, ok := n.(*ast.ReturnStmt); ok && len(ret.Results) > 0 {
			found = true
		}
		return !found
	})
	return found
}

// TestTraceLabelMatchesTheOperation pins both slips at once: a label naming another
// operation and a missing else branch fail the same assertion, because both mean the
// operator is not told which call failed.
func TestTraceLabelMatchesTheOperation(t *testing.T) {
	fset, files := sdkFiles(t)

	sites := 0
	eachCallSite(files, func(pos ast.Node, logBlock ast.Node, operation string) {
		sites++
		where := fset.Position(pos.Pos())

		if logBlock == nil {
			t.Errorf("%s: Call_%s drops its error with nowhere to report it; "+
				"sdk returns no error, so nothing reports this failure", where, operation)
			return
		}
		label, ok := rpcLabel(logBlock)
		if !ok {
			if propagatesError(logBlock) {
				return // returned to the caller, which is the other correct answer
			}
			t.Errorf("%s: Call_%s neither logs a Str(\"rpc\", …) label nor returns the "+
				"error, so the failure is invisible", where, operation)
			return
		}
		if label != operation {
			t.Errorf("%s: Call_%s logs rpc=%q; a wrong label sends an operator after the "+
				"wrong operation", where, operation, label)
		}
	})

	// A guard on the guard: if the shape of these call sites ever changes, the walk above
	// would silently match nothing and this test would pass while checking zero sites.
	if sites < 70 {
		t.Errorf("matched only %d call sites; the RPC call shape moved and this test no "+
			"longer checks what it claims to", sites)
	}
}

// operationNames reads the calls.txt of every service package. Those files are the
// authority on which operations exist -- the generator emits exactly one wrapper per line
// and refuses anything else -- so they are also the authority on what a label may say.
func operationNames(t *testing.T) map[string]string {
	t.Helper()

	out := make(map[string]string)
	for _, pkg := range []string{"device", "media", "ptz", "event"} {
		path := filepath.Join("..", pkg, "calls.txt")
		f, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		for scanner.Scan() {
			// Same filter as bin/onvif-codegen: blank lines and # comments are not entries.
			line := strings.TrimSpace(scanner.Text())
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			out[strings.Fields(line)[0]] = pkg
		}
		if err := scanner.Err(); err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
	}
	if len(out) == 0 {
		t.Fatal("no operation read from any calls.txt")
	}
	return out
}

// TestTraceLabelsNameRealOperations ties the trace vocabulary to the calls.txt 1:1
// invariant. "GetAnalyticsConfiguration" and "GetAnalyticsConfigurations" were logged for
// years and are in no calls.txt, so an operator grepping the WSDL for a label this library
// printed found nothing at all.
func TestTraceLabelsNameRealOperations(t *testing.T) {
	fset, files := sdkFiles(t)
	operations := operationNames(t)

	for _, file := range files {
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if label, ok := rpcLabel(call); ok {
				if _, exists := operations[label]; !exists {
					t.Errorf("%s: rpc=%q is in no calls.txt, so it names no operation this "+
						"library can issue", fset.Position(call.Pos()), label)
				}
				return false // the label is found; do not report it again from its parents
			}
			return true
		})
	}
}
