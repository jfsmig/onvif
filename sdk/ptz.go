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

import (
	"context"
	"sync"

	"github.com/jfsmig/onvif/ptz"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type Ptz struct {
	Capabilities  ptz.Capabilities
	Nodes         []onvif.PTZNode
	Configuration []onvif.PTZConfiguration
}

// FetchPTZ issues its three independent operations concurrently, and the per-node detail
// fetches concurrently within one of them.
//
// Each of the three closures owns one field of out, so no lock is needed. The inner
// per-node fan-out cannot append to out.Nodes -- concurrent appends to one slice race --
// so it writes into a pre-sized slice by index, which also makes the result keep the
// device's node order instead of finishing order.
func (p *ProfileS) FetchPTZ(ctx context.Context) Ptz {
	out := Ptz{}

	var wg sync.WaitGroup

	wg.Go(func() {
		if caps, err := ptz.Call_GetServiceCapabilities(ctx, p.client, ptz.GetServiceCapabilities{}); err == nil {
			out.Capabilities = caps.Capabilities
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("ptz")
		}
	})

	wg.Go(func() {
		nodes, err := ptz.Call_GetNodes(ctx, p.client, ptz.GetNodes{})
		if err != nil {
			Logger.Trace().Err(err).Str("rpc", "GetNodes").Msg("ptz")
			return
		}

		// One slot per node, so each goroutine writes its own element. A node whose detail
		// fetch fails leaves its slot zero and is dropped below, which is what the old
		// append-on-success loop did.
		detailed := make([]onvif.PTZNode, len(nodes.PTZNode))
		filled := make([]bool, len(nodes.PTZNode))

		var inner sync.WaitGroup
		for i, n := range nodes.PTZNode {
			inner.Go(func() {
				if node, err := ptz.Call_GetNode(ctx, p.client, ptz.GetNode{NodeToken: n.Token}); err == nil {
					detailed[i], filled[i] = node.PTZNode, true
				} else {
					Logger.Trace().Err(err).Str("rpc", "GetNode").Msg("ptz")
				}
			})
		}
		inner.Wait()

		for i, ok := range filled {
			if ok {
				out.Nodes = append(out.Nodes, detailed[i])
			}
		}
	})

	wg.Go(func() {
		if cfgs, err := ptz.Call_GetConfigurations(ctx, p.client, ptz.GetConfigurations{}); err == nil {
			out.Configuration = cfgs.PTZConfiguration
		} else {
			Logger.Trace().Err(err).Str("rpc", "GetConfigurations").Msg("ptz")
		}
	})

	wg.Wait()
	return out
}
