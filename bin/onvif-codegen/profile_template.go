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

// profileTemplate renders one Profile client. The output is passed through go/format, so
// indentation here only has to be legal, not pretty.
const profileTemplate = `// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
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

// Code generated from {{.Source}} : DO NOT EDIT.

package {{.Package}}

import (
"context"

"github.com/jfsmig/onvif/networking"
{{range .Services}}"github.com/jfsmig/onvif/{{.}}"
{{end}})

// {{.Type}} offers the operations that the ONVIF Profile {{.Letter}} specification lists for
// a client, over one connection to one appliance.
//
// It is not a conformance statement. This library is not certified, the ONVIF conformance
// test tool is not run against it, and an appliance advertising the required services may
// still reject any of these operations. Membership is curated by hand from the
// specification, one cited section per line, in {{.Source}}.
//
// Not to be confused with an ONVIF *media* profile, which is a per-stream configuration on
// the appliance and is reached through the media service.
type {{.Type}} struct {
client *networking.Client
}

// New{{.Type}} builds the client and reports whether the appliance advertises the services
// that Profile {{.Letter}} requires unconditionally ({{range $i, $s := .Mandatory}}{{if $i}}, {{end}}{{$s}}{{end}}).
//
// It reports on the endpoints already learnt when the appliance was loaded, so it sends
// nothing and a true result means "worth trying", not "will succeed".
{{if .Contingent}}//
// The remaining services are conditional in this Profile ({{range $i, $s := .Contingent}}{{if $i}}, {{end}}{{$s}}{{end}}), so their absence
// does not make the client unusable: their operations return utils.ErrNoService, and the
// Has* predicates below let a caller check first.{{end}}
func New{{.Type}}(client *networking.Client) (*{{.Type}}, bool) {
{{range .Mandatory}}if _, ok := client.HasEndpoint("{{.}}"); !ok {
return nil, false
}
{{end}}return &{{.Type}}{client: client}, true
}
{{range .Contingent}}
// Has{{suffix .}} reports whether the appliance advertises the {{.}} service, which Profile
// {{$.Letter}} treats as conditional.
func (p *{{$.Type}}) Has{{suffix .}}() bool {
_, ok := p.client.HasEndpoint("{{.}}")
return ok
}
{{end}}{{range .Operations}}
// {{.Method}} performs the ONVIF {{.Service}} operation {{.Operation}}.
//
// Profile {{$.Letter}} {{.Section}} {{.Feature}}, {{requirement .Require}}.
func (p *{{$.Type}}) {{.Method}}(ctx context.Context, request {{.Service}}.{{.Operation}}) ({{.Service}}.{{.Operation}}Response, error) {
return {{.Service}}.Call_{{.Operation}}(ctx, p.client, request)
}
{{end}}`
