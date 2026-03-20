# l2h-client (Link to Host Client)

l2h-client 是 l2h 系统的内网反向代理客户端，使用 Go 语言开发。它建立与 l2h-server 的安全 WebRTC 连接，并将公网请求转发至本地端口。

## 功能特性

- **基于 WebRTC (DataChannel)**：无需公网 IP 和端口映射，穿透 NAT 访问。
- **多平台支持**：预编译支持 Linux (x86/ARM)、Windows、macOS。
- **命令行工具 (CLI)**：提供便捷的配置与管理。
- **自动重连**：连接断开后会自动重联至服务器。
- **多服务管理**：在单个客户端上配置并连接至多个不同的服务器。

## 快速安装

### Linux (一键安装)

```bash
curl -fsSL https://raw.githubusercontent.com/Kaiyuan/l2h-client/main/install.sh | bash
```

### Windows

1. 在 [Releases](https://github.com/Kaiyuan/l2h-client/releases) 页面下载 `l2h-cli-windows-amd64.exe`。
2. 重命名为 `l2h-cli.exe` 并将其所在文件夹添加到系统的环境变量 (PATH)。

## 基本使用

```bash
# 首次绑定服务端
l2h-cli -s http://l2h.host:443 -k YOUR_API_KEY

# 列出映射
l2h-cli -p test -l 8080 -S 1
l2h-cli -list

# 开始连接
l2h-cli
```

## 命令行参数

- `-s`：指定服务器地址 (e.g., `http://server:52331`)
- `-k`：用户 API Key
- `-p`：映射路径名
- `-l`：本地端口
- `-list`：显示当前路径映射
- `-list-service`：显示已保存的服务
- `-d`：设置默认服务

## 开发者编译

```bash
go build -o l2h-cli main.go
```

## 开源协议

MIT
