package main

import (
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/ThomasZN/v2sub/ping"
	"github.com/ThomasZN/v2sub/template"
	"github.com/ThomasZN/v2sub/types"
	"github.com/modood/table"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

const (
	v2subConfig = "/etc/v2sub.json"
	v2rayConfig = "/etc/v2ray.json"
	duration    = 5 * time.Second // 建议至少 5s
	//ruleUrl     = "https://raw.githubusercontent.com/PaPerseller/chn-iplist/master/v2ray-config_rule.txt"

	version = "1.1.0"
)

var (
	flags = struct {
		sub bool
		//rule        bool
		sort        bool
		version     bool
		ping        bool
		quick       bool
		global      bool
		url         string
		v2rayConfig string
	}{}

	//ruleHandler func() <-chan *types.RouterConfig
)

func main() {
	flag.BoolVar(&flags.sub, "sub", false, "是否刷新订阅")
	flag.StringVar(&flags.url, "url", "", "订阅地址")
	flag.BoolVar(&flags.ping, "ping", true, "是否对所有节点测试延迟")
	flag.BoolVar(&flags.sort, "sort", false, "是否按延迟排序")
	flag.BoolVar(&flags.global, "global", false, "是否全局代理")
	//flag.BoolVar(&flags.rule, "rule", true, "是否刷新规则")
	flag.BoolVar(&flags.quick, "q", false, "是否快速切换")
	flag.StringVar(&flags.v2rayConfig, "config", v2rayConfig, "v2ray 配置文件")
	flag.BoolVar(&flags.version, "version", false, "显示版本")

	flag.Parse()

	if flags.version {
		fmt.Printf("v2sub v%s\n", version)
		return
	}

	if flags.quick {
		//flags.ping, flags.rule = false, false
		flags.ping = false
	}

	cfg := ReadConfig(v2subConfig)

	start := time.Now()

	//if !flags.global && flags.rule {
	//	fmt.Println("获取规则...")
	//	ruleCh := make(chan *types.RouterConfig, 1)
	//	ruleHandler = func() <-chan *types.RouterConfig {
	//		return ruleCh
	//	}
	//	go GetRule(ruleUrl, ruleCh)
	//}

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
		select {
		case <-time.After(duration):
			fmt.Printf("%s 后仍未获取到订阅信息, 请检查订阅地址和网络状况\n", duration.String())
			os.Exit(0)
		case data := <-subCh:
			for i := range data {
				data[i] = strings.ReplaceAll(data[i], "vmess://", "")
				if nodeData, err := base64.StdEncoding.DecodeString(data[i]); err != nil {
					fmt.Printf("订阅信息格式错误: %v, 建议咨询服务提供商\n", err)
					fmt.Println(data[i])
				} else {
					var node = &types.Node{}
					if err = json.Unmarshal(nodeData, node); err != nil {
						fmt.Printf("订阅信息格式错误: %v, 建议咨询服务提供商\n", err)
						fmt.Println(string(nodeData))
					} else {
						nodes = append(nodes, node)
					}

				}
			}
		}

		fmt.Printf("订阅信息解析完毕, 用时 %ds\n", time.Now().Second()-start.Second())

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

	var tableData []types.TableRow
	for i := range nodes {
		tableData = append(tableData, types.TableRow{
			Index: i,
			Name:  nodes[i].Name,
			Addr:  nodes[i].Addr,
			Ping:  nodes[i].Ping})
	}
	table.Output(tableData)

	var streamSetting types.StreamSetting

	node := func(nodes types.Nodes) *types.Node {
		for {
			fmt.Print("输入节点序号:")
			var nodeIndex int
			_, _ = fmt.Scan(&nodeIndex)
			if nodeIndex < 0 || nodeIndex >= len(nodes) {
				fmt.Println("没有此节点")
			} else {
				fmt.Printf("[%s] Ping: %dms\n", nodes[nodeIndex].Name, nodes[nodeIndex].Ping)
				if nodes[nodeIndex].Net != "" {
					streamSetting.Network = nodes[nodeIndex].Net
				}
				if nodes[nodeIndex].TLS != "" {
					streamSetting.Security = nodes[nodeIndex].TLS
				}
				return nodes[nodeIndex]
			}
		}
	}(nodes)

	var outboundSetting = types.OutboundSetting{VNext: []types.VNextConfig{
		{
			Address: node.Addr,
			Port:    node.Port,
			Users: []struct {
				ID string `json:"id"`
			}{{ID: node.UID}},
		},
	}}

	if setting, err := json.Marshal(outboundSetting); err != nil {
		panic(err) // fatal
	} else {
		var rawSetting json.RawMessage = setting
		cfg.V2rayConfig.OutboundConfigs = []types.OutboundConfig{
			{
				Protocol:       "vmess",
				Settings:       &rawSetting,
				Tag:            "proxy",
				StreamSettings: &streamSetting,
			},
			{
				Protocol: "freedom",
				Tag:      "direct",
			},
			{
				Protocol: "blackhole",
				Tag:      "block",
			},
		}
	}

	if flags.global {
		cfg.V2rayConfig.DNSConfigs = nil
		cfg.V2rayConfig.RouterConfig = nil
	} else {
		cfg.V2rayConfig.DNSConfigs = template.DefaultDNSConfigs
		cfg.V2rayConfig.RouterConfig = template.DefaultRouterConfigs
		//if flags.rule {
		//	select {
		//	case <-time.After(time.Second):
		//		fmt.Printf("%ds 后仍未获取到规则信息, 将使用内置规则\n", time.Now().Second()-start.Second())
		//		cfg.V2rayConfig.RouterConfig = parseRule(template.RuleTemplate)
		//	case rule := <-ruleHandler():
		//		if rule == nil {
		//			fmt.Println("无法获取规则, 将使用内置规则")
		//			cfg.V2rayConfig.RouterConfig = parseRule(template.RuleTemplate)
		//		} else {
		//			fmt.Printf("已获取规则: %s\n", ruleUrl)
		//			cfg.V2rayConfig.RouterConfig = rule
		//		}
		//	}
		//} else {
		//	if cfg.V2rayConfig.RouterConfig == nil || len(cfg.V2rayConfig.RouterConfig.RuleList) == 0 {
		//		//fmt.Println("使用内置规则")
		//		cfg.V2rayConfig.RouterConfig = parseRule(template.RuleTemplate)
		//	}
		//}
	}

	if data, err := json.Marshal(cfg); err != nil {
		panic(err) // fatal
	} else {
		if err = WriteFile(v2subConfig, data); err != nil {
			fmt.Printf("写入 v2sub 配置文件错误: %v\n", err)
			return
		}
	}

	if v2rayCfgData, err := json.Marshal(&cfg.V2rayConfig); err != nil {
		panic(err) // fatal
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

func ReadConfig(name string) *types.Config {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		fmt.Printf("首次运行 v2sub, 将创建 %s\n", v2subConfig)
		return template.ConfigTemplate
	}

	cfg := &types.Config{}
	if err = json.Unmarshal(data, cfg); err != nil {
		fmt.Printf("配置文件损坏: %v\n", err)
		return template.ConfigTemplate
	}
	return cfg
}

func GetSub(url string, ch chan<- []string) {
	defer close(ch)

	// 拿不到订阅信息程序无法进行
	body, err := httpGet(url)
	if err != nil {
		fmt.Printf("获取订阅信息失败: %v\n", err)
		os.Exit(0)
	}

	res, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		fmt.Printf("订阅信息解析失败: %v\n", err)
		os.Exit(0)
	}

	ch <- strings.Split(string(res[:len(res)-1]), "\n") // 多一个换行符
}

//func GetRule(url string, ch chan<- *types.RouterConfig) {
//	defer close(ch)
//
//	// 拿不到规则信息程序仍可进行
//	body, err := httpGet(url)
//	if err != nil {
//		//fmt.Printf("获取规则信息失败: %v\n", err)
//		return
//	}
//
//	var res = parseRule(body)
//	//if res == nil {
//	//	return
//	//}
//
//	ch <- res
//}

func httpGet(url string) ([]byte, error) {
	data, err := http.Get(url)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = data.Body.Close()
	}()
	return ioutil.ReadAll(data.Body)
}

func WriteFile(name string, data []byte) error {
	file, err := os.Create(name)
	if err != nil {
		return err
	}

	_, err = file.Write(data)
	if err != nil {
		return err
	}

	return file.Close()
}

//func parseRule(body []byte) *types.RouterConfig {
//	body = body[strings.Index(string(body), ":")+1 : strings.LastIndex(string(body), ",")] // ASCII
//
//	var res = &types.RouterConfig{}
//	if err := json.Unmarshal(body, res); err != nil {
//		fmt.Printf("parseRule error: %v\n", err)
//		return nil
//	}
//
//	return res
//}
