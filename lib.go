package main

import (
	"encoding/base64"
	"encoding/json"
	"github.com/ThomasZN/v2sub/template"
	"github.com/ThomasZN/v2sub/types"
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
)

func FileExist(name string) bool {
	fileInfo, err := os.Stat(name)
	if err != nil || fileInfo.IsDir() {
		return false
	}
	return true
}

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

// 从url中获取订阅信息并进行base64解码
// http请求错误不发送任何信息; 解码错误发送nil
func GetSub(url string, ch chan<- []string) {
	body, err := httpGet(url)
	if err != nil {
		body, err = httpGet(url) // 尝试两次
		if err != nil {
			return // send none
		}
	}

	res, err := base64.StdEncoding.DecodeString(string(body))
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

// 返回正确解析的节点 以及 无法解析的数据
func ParseNodes(data []string) (types.Nodes, []string) {
	const vmessPrefix = vmessProtocol + "://"
	const trojanPrefix = trojanProtocol + "://"

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
