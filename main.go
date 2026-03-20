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
	if serverAddr != "" && apiKey != "" {
		found := false
		for i, s := range cfg.Services {
			if s.Server == serverAddr {
				cfg.Services[i].APIKey = apiKey
				found = true
				break
			}
		}
		if !found {
			cfg.Services = append(cfg.Services, config.ServiceConfig{
				Server: serverAddr,
				APIKey: apiKey,
			})
			if len(cfg.Services) == 1 {
				cfg.DefaultSvc = 1
			}
		}
		config.SaveConfig(configPath, cfg)
	}

	if listService {
		fmt.Println("--- 已保存的服务 (Services) ---")
		for i, s := range cfg.Services {
			isDefault := ""
			if i+1 == cfg.DefaultSvc {
				isDefault = " (默认)"
			}
			fmt.Printf("[%d] %s%s\n", i+1, s.Server, isDefault)
		}
		return
	}

	if listAPI {
		fmt.Println("--- 已连接的服务与其 API Key ---")
		for i, s := range cfg.Services {
			fmt.Printf("[%d] %s: %s\n", i+1, s.Server, s.APIKey)
		}
		return
	}

	if listPaths {
		fmt.Println("--- 当前路径映射 (Paths) ---")
		for _, p := range cfg.Paths {
			svcStr := ""
			if len(cfg.Services) > 1 {
				svcStr = fmt.Sprintf(" (服务 #%d)", p.Service)
			}
			fmt.Printf("- %-10s -> localhost:%d%s\n", p.Name, p.Port, svcStr)
		}
		return
	}

	if setDefault > 0 && setDefault <= len(cfg.Services) {
		cfg.DefaultSvc = setDefault
		config.SaveConfig(configPath, cfg)
		fmt.Printf("已将默认服务设置为 [%d] %s\n", setDefault, cfg.Services[setDefault-1].Server)
		return
	}

	if pathName != "" && localPort != 0 {
		svc := serviceID
		if svc == 0 {
			svc = cfg.DefaultSvc
		}
		if svc <= 0 || svc > len(cfg.Services) {
			fmt.Println("错误：请指定有效的 -S 服务 ID，或通过 -s -k 绑定服务。")
			return
		}
		fmt.Printf("正在添加路径映射: %s -> %d -> 服务 #%d\n", pathName, localPort, svc)
		cfg.Paths = append(cfg.Paths, config.PathConfig{
			Name:    pathName,
			Port:    localPort,
			Service: svc,
		})
		config.SaveConfig(configPath, cfg)
		return
	}

	// 自动连接逻辑
	var targetSvc *config.ServiceConfig
	if len(cfg.Services) == 0 {
		fmt.Println("错误：没有配置任何服务。使用 -s [地址] -k [API KEY] 进行绑定。")
		pflag.Usage()
		os.Exit(1)
	}

	if cfg.DefaultSvc > 0 && cfg.DefaultSvc <= len(cfg.Services) {
		targetSvc = &cfg.Services[cfg.DefaultSvc-1]
	} else {
		targetSvc = &cfg.Services[0]
	}

	fmt.Printf("正在自动连接默认服务: %s\n", targetSvc.Server)
	webrtc.ConnectWithRetry(targetSvc.Server, targetSvc.APIKey)
}
