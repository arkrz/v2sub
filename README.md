# v2sub

Go 编写的用于 linux 下订阅并简单配置 [v2ray](https://github.com/v2ray/v2ray-core) 的小工具。

程序会创建 `/etc/v2sub.json` 文件用于存储订阅信息。

使用前需确保 v2ray.service 服务已注册且默认 v2ray 配置文件路径为 `/etc/v2ray.json`。