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
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"
)

type CodegenSdkEnv struct {
	Package     string
	TypeReply   string
	TypeRequest string
}

const generatedSuffix = "_auto.go"

func codegenSdk(pkg, source string) error {
	const mainTemplate = `// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
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

// Code generated : DO NOT EDIT.

package {{.Package}}

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_{{.TypeRequest}} forwards the call to dev.CallMethod() then parses the payload of the reply as a {{.TypeReply}}.
func Call_{{.TypeRequest}}(ctx context.Context, dev *networking.Client, request {{.TypeRequest}}) ({{.TypeReply}}, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			{{.TypeReply}} {{.TypeReply}}
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.{{.TypeReply}}, err
	} else {
		err = networking.ReadAndParse(httpReply, &reply, "{{.TypeRequest}}")
		return reply.Body.{{.TypeReply}}, err
	}
}
`

	outDir, err := outputDir(pkg, source)
	if err != nil {
		return err
	}

	methods, err := getMethods(source)
	if err != nil {
		return err
	}

	body, err := template.New("body").Parse(mainTemplate)
	if err != nil {
		return fmt.Errorf("BUG: invalid template: %w", err)
	}

	for _, method := range methods {
		env := CodegenSdkEnv{
			Package:     pkg,
			TypeRequest: method.Name,
			TypeReply:   method.Name + "Response",
		}
		path := filepath.Join(outDir, env.TypeRequest+generatedSuffix)
		if err := writeMethod(path, body, &env); err != nil {
			return err
		}
	}

	// One line per package rather than one per method: `go generate ./...` runs this four
	// times over 205 methods, and the previous per-method line said nothing a failure would
	// not have said better.
	Logger.Info().Str("pkg", pkg).Int("methods", len(methods)).Str("dir", outDir).Msg("Generated")

	return reportOrphans(outDir, methods)
}

// outputDir returns the directory the generated files belong in, which is the one holding
// the source file rather than the process working directory.
//
// The previous code created the files in the working directory and trusted the caller to
// have chdir'ed there. That holds under `go generate`, which does chdir into the package,
// and fails silently everywhere else: run from the repository root, the tool wrote a whole
// package's files into the root, and `git diff` does not report untracked files, so CI
// stayed green. The package name is checked against the directory name because the two
// disagreeing is always a mistake — the generated files declare `package pkg`.
func outputDir(pkg, source string) (string, error) {
	dir := filepath.Dir(source)
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolving %s: %w", dir, err)
	}
	if base := filepath.Base(abs); base != pkg {
		return "", fmt.Errorf("package %q does not match the directory holding %s (%q)", pkg, source, base)
	}
	return dir, nil
}

// writeMethod renders one wrapper. The close is deferred and its error reported: a failure
// at flush time otherwise left a silently truncated file behind.
func writeMethod(path string, body *template.Template, env *CodegenSdkEnv) (err error) {
	fout, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating %s: %w", path, err)
	}
	defer func() {
		if cerr := fout.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("closing %s: %w", path, cerr)
		}
	}()

	if err := body.Execute(fout, env); err != nil {
		return fmt.Errorf("generating %s: %w", path, err)
	}
	return nil
}

// reportOrphans fails when the target directory holds a generated file whose method is no
// longer listed in the source.
//
// Such a file is stale but still compiles, and being tracked and unchanged it is invisible
// to the `git diff --quiet` check that CI relies on, so nothing noticed. It is reported
// rather than deleted: a generator that removes files it was not asked about is a worse
// failure than the one being fixed, and the message says exactly what to remove.
func reportOrphans(outDir string, methods []Method) error {
	found, err := filepath.Glob(filepath.Join(outDir, "*"+generatedSuffix))
	if err != nil {
		return fmt.Errorf("listing %s: %w", outDir, err)
	}

	expected := make(map[string]bool, len(methods))
	for _, method := range methods {
		expected[method.Name+generatedSuffix] = true
	}

	orphans := make([]string, 0)
	for _, path := range found {
		if name := filepath.Base(path); !expected[name] {
			orphans = append(orphans, name)
		}
	}
	if len(orphans) > 0 {
		sort.Strings(orphans)
		return fmt.Errorf("%s holds %d generated file(s) with no entry in the source, remove them: %s",
			outDir, len(orphans), strings.Join(orphans, " "))
	}

	return nil
}
