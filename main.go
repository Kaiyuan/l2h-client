package main

import (
	"fmt"
	"os"

	"github.com/Kaiyuan/l2h-client/internal/config"
	"github.com/Kaiyuan/l2h-client/internal/webrtc"
	"github.com/spf13/pflag"
)

func main() {
	var (
		serverAddr  string
		apiKey      string
		configPath  string
		pathName    string
		localPort   int
		serviceID   int
		listPaths   bool
		listAPI     bool
		listService bool
		setDefault  int
		help        bool
	)

	pflag.StringVarP(&serverAddr, "server", "s", "", "服务器地址 (例如 https://l2h.host:443)")
	pflag.StringVarP(&apiKey, "key", "k", "", "用户 API Key")
	pflag.StringVarP(&configPath, "config", "c", config.GetConfigPath(), "配置文件路径")
	pflag.StringVarP(&pathName, "path", "p", "", "映射的路径名称")
	pflag.IntVarP(&localPort, "local", "l", 0, "要代理的本地端口")
	pflag.IntVarP(&serviceID, "service", "S", 0, "服务 ID")
	pflag.BoolVar(&listPaths, "list", false, "列出所有路径映射")
	pflag.BoolVar(&listAPI, "list-api", false, "列出所有 API Key")
	pflag.BoolVar(&listService, "list-service", false, "列出所有服务")
	pflag.IntVarP(&setDefault, "default", "d", 0, "设置默认服务")
	pflag.BoolVarP(&help, "help", "h", false, "显示帮助信息")

	pflag.Parse()

	if help {
		pflag.Usage()
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("加载配置出错: %v\n", err)
		os.Exit(1)
	}

	// Update config with flags if provided
	if serverAddr != "" {
		cfg.Server = serverAddr
	}
	if apiKey != "" {
		cfg.APIKey = apiKey
	}

	if listService {
		fmt.Println("正在列出服务...")
		// TODO: Implementation (实现)
		return
	}

	if pathName != "" && localPort != 0 {
		fmt.Printf("正在添加路径映射: %s -> %d\n", pathName, localPort)
		cfg.Paths = append(cfg.Paths, config.PathConfig{
			Name:    pathName,
			Port:    localPort,
			Service: serviceID,
		})
		config.SaveConfig(configPath, cfg)
		return
	}

	if cfg.Server == "" || cfg.APIKey == "" {
		fmt.Println("错误：需要服务器地址和 API Key。使用 -s 和 -k 进行绑定。")
		pflag.Usage()
		os.Exit(1)
	}

	fmt.Printf("正在连接服务器: %s\n", cfg.Server)
	webrtc.ConnectWithRetry(cfg.Server, cfg.APIKey)
}
