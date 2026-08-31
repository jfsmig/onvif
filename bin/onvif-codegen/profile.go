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

package main

import (
	"bufio"
	"bytes"
	"fmt"
	"go/format"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

// serviceSuffix maps a service package to the suffix used when an operation name is
// ambiguous. It is a fixed table rather than a derivation because the suffix appears in
// exported method names: inferring it would let a rename elsewhere silently rewrite the API.
// A service absent from this table is an error, not a guess.
var serviceSuffix = map[string]string{
	"device": "Device",
	"media":  "Media",
	"ptz":    "PTZ",
	"event":  "Event",
}

// requirementText renders a manifest requirement flag for a doc comment.
var requirementText = map[string]string{
	"M":  "mandatory for clients",
	"C":  "conditional",
	"O":  "optional",
	"M*": "mandatory for clients, subject to the footnote of that section",
}

type profileOperation struct {
	Service   string // Go package holding the request type and the Call_ wrapper
	Operation string // ONVIF operation name, as the WSDL spells it
	Method    string // Go method name: Operation, suffixed when the name is ambiguous
	Require   string // M, C, O or M*
	Section   string // section of the Profile specification
	Feature   string // title of that section
}

type profileManifest struct {
	Letter     string   // S, T, G...
	Type       string   // ProfileS
	Source     string   // manifest path, for error messages
	Mandatory  []string // services gating the constructor
	Contingent []string // services declared conditional
	Services   []string // every service referenced, sorted
	Operations []profileOperation
}

// codegenProfile generates one Profile client per manifest in manifestDir.
//
// Every manifest is read in one run because the whole set decides how a method is named:
// see collisions below.
func codegenProfile(pkg, manifestDir string) error {
	outDir, root, err := profileOutputDir(pkg, manifestDir)
	if err != nil {
		return err
	}

	paths, err := filepath.Glob(filepath.Join(manifestDir, "*.profile"))
	if err != nil {
		return fmt.Errorf("listing %s: %w", manifestDir, err)
	}
	if len(paths) == 0 {
		return fmt.Errorf("%s: no *.profile manifest", manifestDir)
	}
	sort.Strings(paths)

	// The operations each service actually implements. This is both the validation set and
	// the namespace the collision rule is computed over.
	known, err := loadServiceOperations(root)
	if err != nil {
		return err
	}
	collisions := ambiguousOperations(known)

	manifests := make([]*profileManifest, 0, len(paths))
	for _, path := range paths {
		manifest, err := parseProfileManifest(path, known, collisions)
		if err != nil {
			return err
		}
		manifests = append(manifests, manifest)
	}

	expected := make(map[string]bool, len(manifests))
	for _, manifest := range manifests {
		name := "profile_" + manifest.Letter + generatedSuffix
		expected[name] = true
		if err := writeProfile(filepath.Join(outDir, name), pkg, manifest); err != nil {
			return err
		}
		Logger.Info().Str("profile", manifest.Letter).Int("operations", len(manifest.Operations)).
			Str("file", name).Msg("Generated")
	}

	return reportProfileOrphans(outDir, expected)
}

// profileOutputDir resolves where the generated files go -- the parent of the manifest
// directory, which must be the named package -- and the repository root above it, where the
// service packages live. The manifests sit in a sub-directory so that they do not look like
// Go source.
//
// Both are derived from the absolute path: filepath.Dir("." ) is "." , not the parent, so a
// relative walk upwards silently stays put.
func profileOutputDir(pkg, manifestDir string) (outDir, root string, err error) {
	outDir = filepath.Dir(manifestDir)
	abs, err := filepath.Abs(outDir)
	if err != nil {
		return "", "", fmt.Errorf("resolving %s: %w", outDir, err)
	}
	if base := filepath.Base(abs); base != pkg {
		return "", "", fmt.Errorf("package %q does not match the directory holding %s (%q)", pkg, manifestDir, base)
	}
	return outDir, filepath.Dir(abs), nil
}

// loadServiceOperations reads every <service>/calls.txt under root.
func loadServiceOperations(root string) (map[string]map[string]bool, error) {
	out := make(map[string]map[string]bool, len(serviceSuffix))
	for service := range serviceSuffix {
		methods, err := getMethods(filepath.Join(root, service, "calls.txt"))
		if err != nil {
			return nil, err
		}
		set := make(map[string]bool, len(methods))
		for _, method := range methods {
			set[method.Name] = true
		}
		out[service] = set
	}
	return out, nil
}

// ambiguousOperations reports the operation names exported by more than one service.
//
// The count is taken over every service's whole calls.txt rather than over what the
// manifests happen to mention. Scoping it to the manifests would mean that adding an
// unrelated Profile could rename a method on an existing one -- an API break produced by
// editing a different file. Over calls.txt the answer is fixed for as long as the services
// are: today GetServiceCapabilities, SendAuxiliaryCommand and SetSynchronizationPoint.
func ambiguousOperations(known map[string]map[string]bool) map[string]bool {
	count := make(map[string]int)
	for _, operations := range known {
		for operation := range operations {
			count[operation]++
		}
	}
	out := make(map[string]bool)
	for operation, n := range count {
		if n > 1 {
			out[operation] = true
		}
	}
	return out
}

func parseProfileManifest(path string, known map[string]map[string]bool, collisions map[string]bool) (*profileManifest, error) {
	letter := strings.TrimSuffix(filepath.Base(path), ".profile")
	if !token.IsIdentifier("P"+letter) || !token.IsExported(letter) {
		return nil, fmt.Errorf("%s: %q is not usable as a Profile name", path, letter)
	}

	fin, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("opening %s: %w", path, err)
	}
	defer func() { _ = fin.Close() }()

	manifest := &profileManifest{Letter: letter, Type: "Profile" + letter, Source: path}
	requirement := make(map[string]string) // service -> M|C
	features := make(map[string]string)    // section -> title
	seen := make(map[string]bool)          // service+"."+operation

	scanner := bufio.NewScanner(fin)
	for lineno := 1; scanner.Scan(); lineno++ {
		line := strings.TrimSpace(scanner.Text())
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		where := fmt.Sprintf("%s:%d", path, lineno)

		switch fields[0] {
		case "service":
			if len(fields) != 3 {
				return nil, fmt.Errorf("%s: expected `service <package> <M|C>`", where)
			}
			if _, ok := serviceSuffix[fields[1]]; !ok {
				return nil, fmt.Errorf("%s: unknown service %q", where, fields[1])
			}
			if fields[2] != "M" && fields[2] != "C" {
				return nil, fmt.Errorf("%s: service requirement must be M or C, got %q", where, fields[2])
			}
			if _, dup := requirement[fields[1]]; dup {
				return nil, fmt.Errorf("%s: service %q declared twice", where, fields[1])
			}
			requirement[fields[1]] = fields[2]

		case "feature":
			if len(fields) < 3 {
				return nil, fmt.Errorf("%s: expected `feature <section> <title>`", where)
			}
			features[fields[1]] = strings.Join(fields[2:], " ")

		default:
			if len(fields) != 4 {
				return nil, fmt.Errorf("%s: expected `<service> <Operation> <M|C|O|M*> <section>`", where)
			}
			service, operation, require, section := fields[0], fields[1], fields[2], fields[3]
			if _, ok := serviceSuffix[service]; !ok {
				return nil, fmt.Errorf("%s: unknown service %q", where, service)
			}
			if _, ok := requirementText[require]; !ok {
				return nil, fmt.Errorf("%s: requirement must be M, C, O or M*, got %q", where, require)
			}
			if !token.IsIdentifier(operation) || !token.IsExported(operation) {
				return nil, fmt.Errorf("%s: %q is not an exported Go identifier", where, operation)
			}
			// The check that makes a curation slip fail the build rather than ship: the
			// specification's tables use informal names (7.5 says "Reboot" for
			// SystemReboot) and list entries that are features rather than operations.
			if !known[service][operation] {
				return nil, fmt.Errorf("%s: %s has no operation %q -- see %s/calls.txt", where, service, operation, service)
			}
			if key := service + "." + operation; seen[key] {
				return nil, fmt.Errorf("%s: %s listed twice", where, key)
			} else {
				seen[key] = true
			}
			method := operation
			if collisions[operation] {
				method = operation + serviceSuffix[service]
			}
			manifest.Operations = append(manifest.Operations, profileOperation{
				Service: service, Operation: operation, Method: method,
				Require: require, Section: section,
			})
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	if len(manifest.Operations) == 0 {
		return nil, fmt.Errorf("%s: no operation listed", path)
	}

	// Resolve feature titles and check every cited section was declared.
	for i, operation := range manifest.Operations {
		title, ok := features[operation.Section]
		if !ok {
			return nil, fmt.Errorf("%s: %s cites section %s, which no `feature` line declares",
				path, operation.Operation, operation.Section)
		}
		manifest.Operations[i].Feature = title
	}

	// Every service used must be declared, and every declared service used.
	used := make(map[string]bool)
	for _, operation := range manifest.Operations {
		used[operation.Service] = true
		if _, ok := requirement[operation.Service]; !ok {
			return nil, fmt.Errorf("%s: %s is used but has no `service` line", path, operation.Service)
		}
	}
	for service, level := range requirement {
		if !used[service] {
			return nil, fmt.Errorf("%s: service %s is declared but never used", path, service)
		}
		if level == "M" {
			manifest.Mandatory = append(manifest.Mandatory, service)
		} else {
			manifest.Contingent = append(manifest.Contingent, service)
		}
		manifest.Services = append(manifest.Services, service)
	}

	// Byte-stable output: sort everything the template ranges over. sort.Strings compares
	// bytes, so the result does not move with the locale.
	sort.Strings(manifest.Mandatory)
	sort.Strings(manifest.Contingent)
	sort.Strings(manifest.Services)
	sort.Slice(manifest.Operations, func(i, j int) bool {
		return manifest.Operations[i].Method < manifest.Operations[j].Method
	})

	return manifest, nil
}

func writeProfile(path, pkg string, manifest *profileManifest) (err error) {
	body, err := template.New("profile").Funcs(template.FuncMap{
		"suffix":      func(s string) string { return serviceSuffix[s] },
		"requirement": func(r string) string { return requirementText[r] },
	}).Parse(profileTemplate)
	if err != nil {
		return fmt.Errorf("BUG: invalid template: %w", err)
	}

	var buffer bytes.Buffer
	if err := body.Execute(&buffer, struct {
		Package string
		*profileManifest
	}{pkg, manifest}); err != nil {
		return fmt.Errorf("generating %s: %w", path, err)
	}

	// Formatted rather than hand-aligned: the import block and the method order both follow
	// the manifest, so alignment cannot be maintained in the template the way the per-call
	// generator does it, and any drift would flap the `go generate && git diff` gate.
	formatted, err := format.Source(buffer.Bytes())
	if err != nil {
		return fmt.Errorf("generating %s: %w (the template produced invalid Go)", path, err)
	}

	return os.WriteFile(path, formatted, 0o644)
}

// reportProfileOrphans fails on a generated Profile file whose manifest is gone.
func reportProfileOrphans(outDir string, expected map[string]bool) error {
	found, err := filepath.Glob(filepath.Join(outDir, "profile_*"+generatedSuffix))
	if err != nil {
		return fmt.Errorf("listing %s: %w", outDir, err)
	}
	orphans := make([]string, 0)
	for _, path := range found {
		if name := filepath.Base(path); !expected[name] {
			orphans = append(orphans, name)
		}
	}
	if len(orphans) > 0 {
		sort.Strings(orphans)
		return fmt.Errorf("%s holds %d generated Profile file(s) with no manifest, remove them: %s",
			outDir, len(orphans), strings.Join(orphans, " "))
	}
	return nil
}
