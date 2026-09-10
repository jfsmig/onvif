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

package device

import (
	"context"
	"github.com/jfsmig/onvif/networking"
)

// Call_SetAccessPolicy forwards the call to dev.CallMethod() then parses the payload of the reply as a SetAccessPolicyResponse.
func Call_SetAccessPolicy(ctx context.Context, dev *networking.Client, request SetAccessPolicy) (SetAccessPolicyResponse, error) {
	type Envelope struct {
		Header struct{}
		Body   struct {
			SetAccessPolicyResponse SetAccessPolicyResponse
		}
	}
	reply := Envelope{}
	httpReply, err := dev.CallMethod(ctx, request)
	if httpReply != nil {
		defer httpReply.Body.Close()
	}
	if err != nil {
		return reply.Body.SetAccessPolicyResponse, err
	} else {
		err = networking.ReadAndParse(httpReply, &reply, "SetAccessPolicy")
		return reply.Body.SetAccessPolicyResponse, err
	}
}
