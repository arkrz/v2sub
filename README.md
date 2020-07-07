# v2sub

Go 编写的用于 linux 下订阅并简单配置 [v2ray](https://github.com/v2ray/v2ray-core) 的命令行工具。

程序会创建 `/etc/v2sub.json` 文件用于存储订阅信息。

使用前需确保 v2ray.service 服务已注册且默认 v2ray 配置文件路径为 `/etc/v2ray.json`。

## Features

+ 内置直接可用的配置文件和代理规则， 监听本地 socks \[1081\] 和 http \[1082\]
+ 并发测试节点延迟 (ping)
+ 表格形式打印所有节点
+ ~~可更新代理规则~~ 内置代理规则使用DNS分流+白名单
+ 支持 [trojan](https://github.com/trojan-gfw/trojan) 订阅与配置

![v2sub](https://github.com/ThomasZN/v2sub/raw/master/v2sub.png)

## Usage

因 ping 与 服务重启 权限需要，以 root 权限运行:

```shell script
sudo ./v2sub
```

快速切换节点：

```shell script
sudo ./v2sub -q
```

更多帮助：

```shell script
./v2sub -help
```

## Note

trojan 功能通过 v2ray 转发实现， 因此可以使用规则代理。 使用前需确保 trojan.service 服务已注册且默认配置文件路径为 `/etc/trojan.json`。

## Warning

程序会覆盖 v2ray 配置文件。

This tool will truncate your v2ray config.