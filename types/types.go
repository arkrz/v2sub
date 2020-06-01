package types

import (
	"encoding/json"
)

type Config struct {
	SubUrl      string `json:"subUrl"`
	Nodes       Nodes  `json:"nodes"`
	V2rayConfig V2ray  `json:"v2rayConfig"`
}

type V2ray struct {
	DNSConfigs      *DNSConfig       `json:"dns"`
	RouterConfig    *RouterConfig    `json:"routing"`
	OutboundConfigs []OutboundConfig `json:"outbounds"`
	InboundConfigs  []InboundConfig  `json:"inbounds"`
}

type DNSConfig struct {
	Servers []json.RawMessage `json:"servers"`
}

type RouterConfig struct {
	RuleList       []json.RawMessage `json:"rules"`
	DomainStrategy string            `json:"domainStrategy"`
}

type OutboundConfig struct {
	Protocol string           `json:"protocol"`
	Settings *json.RawMessage `json:"settings"`
	Tag      string           `json:"tag"`
}

type InboundConfig struct {
	Protocol string `json:"protocol"`
	Port     uint32 `json:"port"`
	ListenOn string `json:"listen"`
}

type OutboundSetting struct {
	VNext []VNextConfig `json:"vnext"`
}

type VNextConfig struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
	Tag     string `json:"tag"`
	Users   []struct {
		ID string `json:"id"`
	} `json:"users"`
}

type Node struct {
	Name string `json:"ps"`
	Addr string `json:"add"`
	Port int    `json:"port"`
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
