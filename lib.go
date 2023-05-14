package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/arkrz/v2sub/template"
	"github.com/arkrz/v2sub/types"
	"github.com/modood/table"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
)

const (
	vmessProtocol  = "vmess"
	trojanProtocol = "trojan"
	socksProtocol  = "socks"
	ssProtocol     = "shadowsocks"
)

// ExitWithMsg 输出 msg 并退出
func ExitWithMsg(msg interface{}, code int) {
	fmt.Println(msg)
	os.Exit(code)
}

// FileExist 判断文件是否存在
func FileExist(name string) bool {
	fileInfo, err := os.Stat(name)
	if err != nil || fileInfo.IsDir() {
		return false
	}
	return true
}

// ReadConfig 读取 v2sub 配置文件
func ReadConfig(name string) (*types.Config, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return template.ConfigTemplate, nil // return error?
	}

	cfg := &types.Config{}
	if err = json.Unmarshal(data, cfg); err != nil {
		return template.ConfigTemplate, err
	}
	return cfg, nil
}

// GetSub 从url中获取订阅信息并进行base64解码
// http请求错误不发送任何信息; 解码错误发送nil
func GetSub(url string, ch chan<- []string) {
	body, err := httpGet(url)
	if err != nil {
		body, err = httpGet(url) // 尝试两次
		if err != nil {
			return // send none
		}
	}

	bodyStr := string(body)
	complementLen := (4 - (len(bodyStr) % 4)) % 4

	for i := 0; i < complementLen; i++ {
		bodyStr += "="
	}

	res, err := base64.StdEncoding.DecodeString(bodyStr)
	if err != nil {
		ch <- nil
		return // send nil
	}

	ch <- strings.Split(string(res[:len(res)-1]), "\n") // 多一个换行符
}

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

// ParseNodes 返回正确解析的节点 以及 无法解析的数据
func ParseNodes(data []string) (types.Nodes, []string) {
	const vmessPrefix = vmessProtocol + "://"
	const trojanPrefix = trojanProtocol + "://"
	const ssPrefix = "ss" + "://"

	var nodes types.Nodes
	var retData []string
	for i := range data {
		switch {
		// vmess
		case strings.HasPrefix(data[i], vmessPrefix):
			data[i] = strings.ReplaceAll(data[i], vmessPrefix, "")
			if nodeData, err := base64.StdEncoding.DecodeString(data[i]); err != nil {
				retData = append(retData, data[i])
			} else {
				var node = &types.Node{}
				if err = json.Unmarshal(nodeData, node); err != nil {
					retData = append(retData, data[i])
				} else {
					node.Protocol = vmessProtocol
					nodes = append(nodes, node)
				}
			}

		// trojan
		case strings.HasPrefix(data[i], trojanPrefix):
			data[i] = strings.ReplaceAll(data[i], trojanPrefix, "")
			node, ok := parseTrojanSub(data[i])
			if !ok {
				retData = append(retData, data[i])
			} else {
				node.Protocol = trojanProtocol
				nodes = append(nodes, node)
			}

		// ss
		case strings.HasPrefix(data[i], ssPrefix):
			data[i] = strings.ReplaceAll(data[i], ssPrefix, "")
			node, ok := parseSSSub(data[i])
			if !ok {
				retData = append(retData, data[i])
			} else {
				node.Protocol = ssProtocol
				nodes = append(nodes, node)
			}

		default:
			retData = append(retData, data[i])
		}
	}

	return nodes, retData
}

func parseTrojanSub(data string) (*types.Node, bool) {
	idEnd := strings.Index(data, "@")
	if idEnd < 0 {
		return nil, false
	}
	id := data[:idEnd]
	data = data[idEnd+1:]

	addrEnd := strings.Index(data, ":")
	if addrEnd < 0 {
		return nil, false
	}
	addr := data[:addrEnd]
	data = data[addrEnd+1:]

	portEnd := strings.Index(data, "?")
	if portEnd < 0 {
		return nil, false
	}
	port, err := strconv.Atoi(data[:portEnd])
	if err != nil {
		return nil, false
	}
	data = data[portEnd+1:]

	nameBegin := strings.Index(data, "#") + 1
	if nameBegin <= 0 {
		return nil, false
	}
	name := data[nameBegin:]
	name, err = url.QueryUnescape(name) //URL解码
	if err != nil {
		return nil, false
	}
	name = name[:len(name)-1] //多一个 /r

	return &types.Node{
		Name: name,
		Addr: addr,
		Port: port,
		UID:  id,
	}, true
}

func parseSSSub(data string) (*types.Node, bool) {
	idEnd := strings.Index(data, "@")
	if idEnd < 0 {
		return nil, false
	}
	id := data[:idEnd]
	data = data[idEnd+1:]

	addrEnd := strings.Index(data, ":")
	if addrEnd < 0 {
		return nil, false
	}
	addr := data[:addrEnd]
	data = data[addrEnd+1:]

	portEnd := strings.Index(data, "#")
	if portEnd < 0 {
		return nil, false
	}
	port, err := strconv.Atoi(data[:portEnd])
	if err != nil {
		return nil, false
	}
	data = data[portEnd+1:]

	name, err := url.QueryUnescape(data) //URL解码
	if err != nil {
		return nil, false
	}
	name = name[:len(name)-1] //多一个 /r

	byteID, err := base64.RawURLEncoding.DecodeString(id)
	if err != nil {
		return nil, false
	}
	strID := string(byteID)

	methodEnd := strings.Index(strID, ":")
	if methodEnd < 0 {
		return nil, false
	}

	return &types.Node{
		Name: name,
		Addr: addr,
		Port: port,
		UID:  strID[methodEnd+1:],
		Type: strID[:methodEnd],
	}, true
}

// WriteFile 覆写文件
// 若文件不存在则会创建并写入
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

func setRuleProxy(config *types.V2ray) {
	config.DNSConfigs = template.DefaultDNSConfigs
	config.RouterConfig = template.DefaultRouterConfigs
}

func setGlobalProxy(config *types.V2ray) {
	config.DNSConfigs = nil
	config.RouterConfig = nil
}

func listenOnLocal(config *types.V2ray) {
	for i := range config.InboundConfigs {
		config.InboundConfigs[i].ListenOn = template.ListenOnLocalAddr
	}
}

func listenOnWan(config *types.V2ray) {
	for i := range config.InboundConfigs {
		config.InboundConfigs[i].ListenOn = template.ListenOnWanAddr
	}
}

func listenOnPort(config *types.V2ray) {
	for i := range config.InboundConfigs {
		switch config.InboundConfigs[i].Protocol {
		case template.ListenOnSocksProtocol:
			if flags.socksPort != 0 {
				config.InboundConfigs[i].Port = uint32(flags.socksPort)
			}

		case template.ListenOnHttpProtocol:
			if flags.httpPort != 0 {
				config.InboundConfigs[i].Port = uint32(flags.httpPort)
			}
		}
	}
}

func parsePort(v interface{}) (port int) {
	portStr := fmt.Sprintf("%v", v)
	port, _ = strconv.Atoi(portStr)
	return
}

func printAsTable(nodes types.Nodes) {
	var tableData []types.TableRow
	for i := range nodes {
		tableData = append(tableData, types.TableRow{
			Index:    i,
			Name:     nodes[i].Name,
			Addr:     nodes[i].Addr,
			Port:     parsePort(nodes[i].Port),
			Protocol: nodes[i].Protocol,
			Ping:     nodes[i].Ping})
	}
	table.Output(tableData)
}

func allDone(cfg *types.Config) {
	var msg string
	for _, inboundConfig := range cfg.V2rayConfig.InboundConfigs {
		if len(msg) == 0 {
			msg = "\n开始监听...\n"
		}
		msg += fmt.Sprintf("%s://%s:%d\n", inboundConfig.Protocol, inboundConfig.ListenOn, inboundConfig.Port)
	}
	msg += "\nAll done."
	fmt.Println(msg)
}
