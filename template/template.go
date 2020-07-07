package template

import (
	"encoding/json"
	"github.com/ThomasZN/v2sub/types"
)

var domainStrategy = "ipondemand"

var ConfigTemplate = &types.Config{
	SubUrl: "",
	Nodes:  types.Nodes{},
	V2rayConfig: types.V2ray{
		RouterConfig: &types.RouterConfig{
			RuleList:       nil,
			DomainStrategy: domainStrategy,
		},
		OutboundConfigs: []types.OutboundConfig{},
		InboundConfigs: []types.InboundConfig{
			{
				Protocol: "socks",
				Port:     1081,
				ListenOn: "127.0.0.1",
				//PortRange: &conf.PortRange{ // [from, to]
				//	From: 1080,
				//	To:   1080,
				//}, // https://github.com/v2ray/v2ray-core/blob/v4.21.3/app/proxyman/inbound/always.go#L91
				//ListenOn: &conf.Address{Address: net.ParseAddress("127.0.0.1")},
			},
			{
				Protocol: "http",
				Port:     1082,
				ListenOn: "127.0.0.1",
			},
		},
	},
}

//参考 https://toutyrater.github.io/routing/configurate_rules.html
var DefaultDNSConfigs = &types.DNSConfig{Servers: []json.RawMessage{
	[]byte(`"114.114.114.114"`),
	[]byte(
		`{
			"address": "1.1.1.1",
			"port": 53,
			"domains": [
				"geosite:geolocation-!cn"
			]
		}`),
}}

var DefaultRouterConfigs = &types.RouterConfig{
	RuleList: []json.RawMessage{
		[]byte(
			`{
				"type": "field",
				"outboundTag": "direct",
				"domain": [
					"geosite:cn"
				]
			}`),
		[]byte(
			`{
                "type": "field",
                "outboundTag": "direct",
                "ip": [
                    "geoip:cn",
                    "geoip:private"
                ]
            }`),
		[]byte(
			`{
                "type": "field",
                "outboundTag": "proxy",
                "network": "udp,tcp"
            }`),
	},
	DomainStrategy: domainStrategy,
}

var DefaultOutboundConfigs = []types.OutboundConfig{
	{
		Protocol: "freedom",
		Tag:      "direct",
	},
	{
		Protocol: "blackhole",
		Tag:      "block",
	},
}
