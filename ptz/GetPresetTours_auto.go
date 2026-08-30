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

// Code generated : DO NOT EDIT.

package ptz

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_GetPresetTours forwards the call to dev.CallMethod() then parses the payload of the reply as a GetPresetToursResponse.
func Call_GetPresetTours(ctx context.Context, dev *networking.Client, request GetPresetTours) (GetPresetToursResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			GetPresetToursResponse GetPresetToursResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.GetPresetToursResponse, err
	} else {
		err = networking.ReadAndParse(ctx, httpReply, &reply, "GetPresetTours")
		return reply.Body.GetPresetToursResponse, err
	}
}
