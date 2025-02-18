ip-blackcate
===

ip黑名单, 用于把扫描ip ban掉。

## 配置

```json
{
    "black_port_list": [ //探测的端口范围, 如果这些范围内的端口被外部访问, 则将其ip拉入黑名单
        "9998-10000"
    ],
    "db_file": "/data/ip.db", //存储扫描ip的db
    "log_config": {
        "level": "debug",
        "console": true
    },
    "user_ip_black_list_dir": "/blacklist", //用户自定义的黑名单列表存储目录, 文件使用`blacklist-`开头, 一行一个ip
    "user_ip_white_list_dir": "/whitelist" //用户自定义的黑名单列表存储目录, 文件使用`whitelist-`开头一行一个ip
}
```

## 运行方式

使用docker运行

```yaml
services:
  ip-blackcage:
    image: xxxsen/ip-blackcage:latest
    container_name: "ip-blackcage"
    privileged: true # 特权模式
    volumes:
      - ./config:/config
      - ./blacklist:/blacklist
      - ./whitelist:/whitelist
      - ./data:/data
    restart: always
    command: --config=/config/config.json
    network_mode: "host"
```

