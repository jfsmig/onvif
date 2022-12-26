package sdk

import (
	"context"
	"github.com/jfsmig/onvif/ptz"
	"github.com/jfsmig/onvif/xsd/onvif"
)

type Ptz struct {
	Capabilities  ptz.Capabilities
	Nodes         []onvif.PTZNode
	Configuration []onvif.PTZConfiguration
}

func (dw *deviceWrapper) FetchPTZ(ctx context.Context) Ptz {
	out := Ptz{}

	if caps, err := ptz.Call_GetServiceCapabilities(ctx, dw.client, ptz.GetServiceCapabilities{}); err == nil {
		out.Capabilities = caps.Capabilities
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetServiceCapabilities").Msg("ptz")
	}

	if nodes, err := ptz.Call_GetNodes(ctx, dw.client, ptz.GetNodes{}); err == nil {
		for _, n := range nodes.PTZNode {
			if node, err := ptz.Call_GetNode(ctx, dw.client, ptz.GetNode{NodeToken: n.Token}); err == nil {
				out.Nodes = append(out.Nodes, node.PTZNode)
			} else {
				Logger.Trace().Err(err).Str("rpc", "GetNode").Msg("ptz")
			}
		}
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetNodes").Msg("ptz")
	}

	if cfgs, err := ptz.Call_GetConfigurations(ctx, dw.client, ptz.GetConfigurations{}); err == nil {
		out.Configuration = cfgs.PTZConfiguration
	} else {
		Logger.Trace().Err(err).Str("rpc", "GetConfigurations").Msg("ptz")
	}

	return out
}
