package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ThomasZN/v2sub/ping"
	"github.com/ThomasZN/v2sub/template"
	"github.com/ThomasZN/v2sub/types"
	"github.com/modood/table"
	"os"
	"os/exec"
	"sort"
	"time"
)

const (
	v2subConfig  = "/etc/v2sub.json"
	v2rayConfig  = "/etc/v2ray.json"
	trojanConfig = "/etc/trojan.json"
	duration     = 5 * time.Second // 建议至少 5s
	//ruleUrl     = "https://raw.githubusercontent.com/PaPerseller/chn-iplist/master/v2ray-config_rule.txt"

	version = "1.2.0"
)

var (
	flags = struct {
		sub         bool
		sort        bool
		version     bool
		ping        bool
		quick       bool
		global      bool
		url         string
		v2rayConfig string
	}{}
)

func main() {
	flag.BoolVar(&flags.sub, "sub", false, "是否刷新订阅")
	flag.StringVar(&flags.url, "url", "", "订阅地址")
	flag.BoolVar(&flags.ping, "ping", true, "是否对所有节点测试延迟")
	flag.BoolVar(&flags.sort, "sort", false, "是否按延迟排序")
	flag.BoolVar(&flags.global, "global", false, "是否全局代理")
	flag.BoolVar(&flags.quick, "q", false, "是否快速切换")
	flag.StringVar(&flags.v2rayConfig, "config", v2rayConfig, "v2ray 配置文件")
	flag.BoolVar(&flags.version, "version", false, "显示版本")

	flag.Parse()

	if flags.version {
		fmt.Printf("v2sub v%s\n", version)
		return
	}

	if flags.quick {
		flags.ping = false
	}

	//本地配置文件读取
	if exist := FileExist(v2subConfig); !exist {
		fmt.Printf("首次运行 v2sub, 将创建 %s\n", v2subConfig)
	}
	cfg, err := ReadConfig(v2subConfig)
	if err != nil {
		fmt.Printf("v2sub 配置文件损坏: %v\n", err)
	}

	//获取节点
	var nodes = func() types.Nodes {
		if !flags.sub && flags.url == "" && len(cfg.Nodes) != 0 {
			fmt.Println("使用缓存的订阅信息, 如需刷新请指定 -sub")
			return cfg.Nodes
		}

		if flags.url != "" {
			cfg.SubUrl = flags.url
		}

		if cfg.SubUrl == "" {
			fmt.Print("输入订阅地址:")
			_, _ = fmt.Scan(&cfg.SubUrl)
		} else {
			fmt.Printf("订阅地址: %s\n", cfg.SubUrl)
		}

		fmt.Println("开始解析订阅信息...")

		var nodes types.Nodes
		subCh := make(chan []string, 1)
		go GetSub(cfg.SubUrl, subCh)
		defer close(subCh)
		select {
		case <-time.After(duration):
			fmt.Printf("%s 后仍未获取到订阅信息, 请检查订阅地址和网络状况\n", duration.String())
			os.Exit(0)
		case data := <-subCh:
			if data == nil {
				fmt.Println("base64 解码错误, 请核实订阅编码")
				os.Exit(0)
			}
			nodes, data = ParseNodes(data)
			if len(data) != 0 {
				fmt.Println("无法解析下列节点:")
				for i := range data {
					fmt.Println(data[i])
				}
			}
		}

		cfg.Nodes = nodes
		return nodes
	}()

	if flags.ping {
		fmt.Printf("正在测试延迟, 等待 %s...\n", duration.String())
		ping.Ping(nodes, duration)

		if flags.sort {
			sort.Sort(nodes)
		}
	}

	//表格打印
	{
		var tableData []types.TableRow
		for i := range nodes {
			tableData = append(tableData, types.TableRow{
				Index: i,
				Name:  nodes[i].Name,
				Addr:  nodes[i].Addr,
				Port:  nodes[i].Port,
				Ping:  nodes[i].Ping})
		}
		table.Output(tableData)
	}

	//节点选择
	node := func(nodes types.Nodes) *types.Node {
		for {
			fmt.Print("输入节点序号:")
			var nodeIndex int
			_, _ = fmt.Scan(&nodeIndex)
			if nodeIndex < 0 || nodeIndex >= len(nodes) {
				fmt.Println("没有此节点")
			} else {
				fmt.Printf("[%s] Ping: %dms\n", nodes[nodeIndex].Name, nodes[nodeIndex].Ping)
				return nodes[nodeIndex]
			}
		}
	}(nodes)

	var v2rayOutboundProtocol string
	var outboundSetting interface{}
	var streamSetting types.StreamSetting //v2ray.streamSettings
	switch node.Protocol {
	case vmessProtocol:
		v2rayOutboundProtocol = vmessProtocol
		outboundSetting = &types.VnextOutboundSetting{VNext: []types.VNextConfig{
			{
				Address: node.Addr,
				Port:    node.Port,
				Users: []struct {
					ID string `json:"id"`
				}{{ID: node.UID}},
			},
		}}
		streamSetting.Network = node.Net
		streamSetting.Security = node.TLS

	case trojanProtocol:
		v2rayOutboundProtocol = socksProtocol

		// 启动 trojan
		trojan := template.TrojanTemplate // 是否需要从本地读取 trojan config?
		trojan.Password = []string{node.UID}
		trojan.RemoteAddr = node.Addr
		trojan.RemotePort = node.Port
		if trojanRaw, err := json.Marshal(trojan); err != nil {
			panic(err) // ?
		} else {
			if err = WriteFile(trojanConfig, trojanRaw); err != nil {
				fmt.Printf("写入 trojan 配置文件错误: %v\n", err)
				return
			}
		}
		fmt.Println("重启 trojan 服务...")
		if err = exec.Command("systemctl", "restart", "trojan.service").Run(); err != nil {
			fmt.Printf("重启失败: %v\n", err)
			return
		}
		fmt.Println("trojan 启动完成")

		outboundSetting = &types.SocksOutboundSetting{Servers: []types.SocksServerConfig{
			{
				Address: trojan.LocalAddr,
				Port:    trojan.LocalPort,
			},
		}}
		streamSetting.Network = "tcp"
		streamSetting.Security = "none"

	default:
		panic("unexpected protocol: " + node.Protocol) // ?
	}

	if setting, err := json.Marshal(outboundSetting); err != nil {
		panic(err) // ?
	} else {
		var rawSetting json.RawMessage = setting
		cfg.V2rayConfig.OutboundConfigs = append([]types.OutboundConfig{
			{
				Protocol:       v2rayOutboundProtocol,
				Settings:       &rawSetting,
				Tag:            "proxy",
				StreamSettings: &streamSetting,
			},
		}, template.DefaultOutboundConfigs...)
	}

	if flags.global {
		setGlobalProxy(&cfg.V2rayConfig)
	} else {
		setRuleProxy(&cfg.V2rayConfig)
	}

	if data, err := json.Marshal(cfg); err != nil {
		panic(err) // ?
	} else {
		if err = WriteFile(v2subConfig, data); err != nil {
			fmt.Printf("写入 v2sub 配置文件错误: %v\n", err)
			return
		}
	}

	if v2rayCfgData, err := json.Marshal(&cfg.V2rayConfig); err != nil {
		panic(err) // ?
	} else {
		if err = WriteFile(flags.v2rayConfig, v2rayCfgData); err != nil {
			fmt.Printf("写入 v2ray 配置文件错误: %v\n", err)
			return
		}
		fmt.Println("重启 v2ray 服务...")
		if err = exec.Command("systemctl", "restart", "v2ray.service").Run(); err != nil {
			fmt.Printf("重启失败: %v\n", err)
			return
		}
	}

	fmt.Println("All done.")
}
