package template

import "github.com/ThomasZN/v2sub/types"

var TrojanTemplate = &types.Trojan{
	RunType:    "client",
	LocalAddr:  ListenOnLocalAddr,
	LocalPort:  1085,
	RemoteAddr: "",
	RemotePort: 0,
	Password:   []string{},
}
