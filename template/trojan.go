package template

import "github.com/ThomasZN/v2sub/types"

var TrojanTemplate = &types.Trojan{
	RunType:    "client",
	LocalAddr:  "127.0.0.1",
	LocalPort:  1085,
	RemoteAddr: "",
	RemotePort: 0,
	Password:   []string{},
}
