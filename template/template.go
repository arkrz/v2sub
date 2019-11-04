package template

import (
	"github.com/ThomasZN/v2sub/types"
	"v2ray.com/core/common/net"
	"v2ray.com/core/infra/conf"
)

var domainStrategy = "ipondemand"

var ConfigTemplate = &types.Config{
	SubUrl: "",
	V2rayConfig: conf.Config{
		RouterConfig: &conf.RouterConfig{
			RuleList:       nil,
			DomainStrategy: &domainStrategy,
		},
		OutboundConfigs: []conf.OutboundDetourConfig{
			{
				Protocol: "vmess",
				Settings: nil,
			},
		},
		InboundConfigs: []conf.InboundDetourConfig{
			{
				Protocol: "socks",
				PortRange: &conf.PortRange{ // [from, to]
					From: 1080,
					To:   1080,
				}, // https://github.com/v2ray/v2ray-core/blob/v4.21.3/app/proxyman/inbound/always.go#L91
				ListenOn: &conf.Address{Address: net.ParseAddress("127.0.0.1")},
			},
		},
	},
}
