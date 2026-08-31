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

package analytics

import (
	"encoding/xml"
	"strings"
	"testing"
)

// The XMLName carried the tev (events) prefix on an analytics operation, while the sibling
// fields correctly used tan. Verified against docs/wsdl/analytics.wsdl, whose
// targetNamespace is http://www.onvif.org/ver20/analytics/wsdl.
func TestCreateAnalyticsModulesUsesTheAnalyticsNamespace(t *testing.T) {
	b, err := xml.Marshal(CreateAnalyticsModules{})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	got := string(b)
	if strings.Contains(got, "tev:") {
		t.Fatalf("analytics operation still carries the events prefix: %s", got)
	}
	if !strings.HasPrefix(got, "<tan:CreateAnalyticsModules>") {
		t.Fatalf("operation element is not tan:CreateAnalyticsModules: %s", got)
	}
}
