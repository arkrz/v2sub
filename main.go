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
	"sort"
	"strings"
	"time"
)

const (
	v2subConfig = "v2sub.json"
	duration    = 5 * time.Second // 建议至少 5s

	version = "1.0.0"
)

var flags = struct {
	sub     bool
	sort    bool
	url     string
	version bool
}{}

func main() {
	flag.BoolVar(&flags.sub, "sub", false, "是否刷新订阅")
	flag.StringVar(&flags.url, "url", "", "订阅地址")
	flag.BoolVar(&flags.sort, "sort", false, "是否按延迟排序")
	flag.BoolVar(&flags.version, "version", false, "显示版本")

	flag.Parse()

	if flags.version {
		fmt.Printf("v2sub v%s\n", version)
		return
	}

	var nodes = func() types.Nodes {
		cfg, err := ReadConfig("./" + v2subConfig)
		if err != nil {
			fmt.Printf("无法找到 %s, 将在当前目录下创建\n", v2subConfig)
			cfg = template.ConfigTemplate
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
		subStart := time.Now()

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
					os.Exit(0)
				} else {
					var node = &types.Node{}
					if err = json.Unmarshal(nodeData, node); err != nil {
						fmt.Printf("订阅信息格式错误: %v, 建议咨询服务提供商\n", err)
						os.Exit(0)
					}
					nodes = append(nodes, node)
				}
			}
		}

		fmt.Printf("订阅信息解析完毕, 用时 %ds\n", time.Now().Second()-subStart.Second())

		return nodes
	}()

	fmt.Printf("正在测试延迟, 等待 %s...\n", duration.String())
	ping.Ping(nodes, duration)

	if flags.sort {
		sort.Sort(nodes)
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

}

func ReadConfig(name string) (*types.Config, error) {
	data, err := ioutil.ReadFile(name)
	if err != nil {
		return nil, err
	}

	cfg := &types.Config{}
	err = json.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func GetSub(url string, ch chan<- []string) {
	defer close(ch)

	data, err := http.Get(url)
	if err != nil {
		fmt.Printf("GetSub error: %v\n", err)
		return
	}

	body, err := ioutil.ReadAll(data.Body)
	if err != nil {
		fmt.Printf("GetSub error: %v\n", err)
		return
	} else {
		_ = data.Body.Close()
	}

	res, err := base64.StdEncoding.DecodeString(string(body))
	if err != nil {
		fmt.Printf("GetSub error: %v\n", err)
		return
	}

	ch <- strings.Split(string(res[:len(res)-1]), "\n") // 多一个换行符
}
