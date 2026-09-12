// Copyright (C) 2022-2026 Jean-Francois SMIGIELSKI
//
// Portions of this file derive from the goonvif project, now use-go/onvif,
// originally distributed under the MIT License, see LICENSE.MIT,
// Copyright (c) 2018 Yakovlev Dmitry, Zhorzh Palanjyan, Crazybber.
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
	"github.com/jfsmig/onvif/v2/xsd"
	"github.com/jfsmig/onvif/v2/xsd/onvif"
)

type GetSupportedRules struct {
	XMLName            string               `xml:"tan:GetSupportedRules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
}

type CreateRules struct {
	XMLName            string               `xml:"tan:CreateRules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
	Rule               onvif.Config         `xml:"tan:Rule"`
}

type DeleteRules struct {
	XMLName            string               `xml:"tan:DeleteRules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
	RuleName           xsd.String           `xml:"tan:RuleName"`
}

type GetRules struct {
	XMLName            string               `xml:"tan:GetRules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
}

type GetRuleOptions struct {
	XMLName            string               `xml:"tan:GetRuleOptions"`
	RuleType           xsd.QName            `xml:"tan:RuleType"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
}

type ModifyRules struct {
	XMLName            string               `xml:"tan:ModifyRules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
	Rule               onvif.Config         `xml:"tan:Rule"`
}

type GetServiceCapabilities struct {
	XMLName string `xml:"tan:GetServiceCapabilities"`
}

type GetSupportedAnalyticsModules struct {
	XMLName            string               `xml:"tan:GetSupportedAnalyticsModules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
}

type GetAnalyticsModuleOptions struct {
	XMLName            string               `xml:"tan:GetAnalyticsModuleOptions"`
	Type               xsd.QName            `xml:"tan:Type"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
}

type CreateAnalyticsModules struct {
	XMLName            string               `xml:"tan:CreateAnalyticsModules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
	AnalyticsModule    onvif.Config         `xml:"tan:AnalyticsModule"`
}

type DeleteAnalyticsModules struct {
	XMLName             string               `xml:"tan:DeleteAnalyticsModules"`
	ConfigurationToken  onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
	AnalyticsModuleName xsd.String           `xml:"tan:AnalyticsModuleName"`
}

type GetAnalyticsModules struct {
	XMLName            string               `xml:"tan:GetAnalyticsModules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
}

type ModifyAnalyticsModules struct {
	XMLName            string               `xml:"tan:ModifyAnalyticsModules"`
	ConfigurationToken onvif.ReferenceToken `xml:"tan:ConfigurationToken"`
	AnalyticsModule    onvif.Config         `xml:"tan:AnalyticsModule"`
}
