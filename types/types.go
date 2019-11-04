package types

import "v2ray.com/core/infra/conf"

type Config struct {
	SubUrl      string      `json:"subUrl"`
	V2rayConfig conf.Config `json:"v2rayConfig"`
}

type OutboundSetting struct {
	VNext []VNextConfig `json:"vnext"`
}

type VNextConfig struct {
	Address string `json:"address"`
	Port    string `json:"port"`
	Users   []struct {
		ID string `json:"id"`
	} `json:"users"`
}

type RuleConfig struct {
	OutboundTag string   `json:"outboundTag"` // e.g: proxy direct
	Domain      []string `json:"domain"`
	Type        string   `json:"type"` // e.g: field
}

type Node struct {
	Name string `json:"ps"`
	Addr string `json:"add"`
	Port string `json:"port"`
	UID  string `json:"id"`
	AID  string `json:"aid"`
	Net  string `json:"net"`
	Type string `json:"type"`
	Host string `json:"host"`
	TLS  string `json:"tls"`

	Ping int `json:"-"`
}

type Nodes []*Node

func (ns Nodes) Len() int { return len(ns) }
func (ns Nodes) Less(i, j int) bool {
	switch {
	case ns[i].Ping == -1:
		return false
	case ns[j].Ping == -1:
		return true
	default:
		return ns[i].Ping < ns[j].Ping
	}
}
func (ns Nodes) Swap(i, j int) { ns[i], ns[j] = ns[j], ns[i] }

type TableRow struct {
	Index int
	Name  string
	Addr  string
	Ping  int
}
