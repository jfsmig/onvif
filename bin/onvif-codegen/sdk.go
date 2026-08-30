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
	"context"
	"log"
	"os"
	"text/template"
)

type CodegenSdkEnv struct {
	Package     string
	Source      string
	TypeReply   string
	TypeRequest string

	// The device package requires a special management because it is both a
	// main entry point of the onvif SDK but also a core internal API
	IsNotDevicePackage bool
}

func codegenSdk(ctx context.Context, pkg, source string) error {
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
		err = networking.ReadAndParse(ctx, httpReply, &reply, "{{.TypeRequest}}")
		return reply.Body.{{.TypeReply}}, err
	}
}
`

	env := CodegenSdkEnv{
		Package:            pkg,
		IsNotDevicePackage: pkg != "device",
		Source:             source,
	}

	body, err := template.New("body").Parse(mainTemplate)
	if err != nil {
		Logger.Fatal().Err(err).Msg("BUG: invalid template")
	}

	for _, method := range getMethods(env.Source) {
		env.TypeRequest = method.Name
		env.TypeReply = method.Name + "Response"
		log.Println(env)

		if fout, err := os.Create(env.TypeRequest + "_auto.go"); err != nil {
			Logger.Fatal().Err(err).Str("wd", getwd()).Str("file", env.Source).Msg("file creation failure")
		} else {
			if err = body.Execute(fout, &env); err != nil {
				Logger.Fatal().Err(err).Str("wd", getwd()).Str("file", env.Source).Msg("file generation failure")
			}
			fout.Close()
		}
	}

	return nil
}
