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

	pflag.StringVarP(&serverAddr, "server", "s", "", "Server address (e.g., https://l2h.host:443)")
	pflag.StringVarP(&apiKey, "key", "k", "", "API KEY")
	pflag.StringVarP(&configPath, "config", "c", config.GetConfigPath(), "Config file path")
	pflag.StringVarP(&pathName, "path", "p", "", "Path name for mapping")
	pflag.IntVarP(&localPort, "local", "l", 0, "Local port to proxy")
	pflag.IntVarP(&serviceID, "service", "S", 0, "Service ID")
	pflag.BoolVar(&listPaths, "list", false, "List all Path mappings")
	pflag.BoolVar(&listAPI, "list-api", false, "List all API keys")
	pflag.BoolVar(&listService, "list-service", false, "List all services")
	pflag.IntVarP(&setDefault, "default", "d", 0, "Set default service")
	pflag.BoolVarP(&help, "help", "h", false, "Show help")

	pflag.Parse()

	if help {
		pflag.Usage()
		return
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
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
		fmt.Println("Listing services...")
		// TODO: Implementation
		return
	}

	if pathName != "" && localPort != 0 {
		fmt.Printf("Adding path mapping: %s -> %d\n", pathName, localPort)
		cfg.Paths = append(cfg.Paths, config.PathConfig{
			Name:    pathName,
			Port:    localPort,
			Service: serviceID,
		})
		config.SaveConfig(configPath, cfg)
		return
	}

	if cfg.Server == "" || cfg.APIKey == "" {
		fmt.Println("Error: Server address and API KEY are required. Use -s and -k to bind.")
		pflag.Usage()
		os.Exit(1)
	}

	fmt.Printf("Connecting to server: %s\n", cfg.Server)
	err = webrtc.ConnectToServer(cfg.Server, cfg.APIKey)
	if err != nil {
		fmt.Printf("Failed to connect: %v\n", err)
		os.Exit(1)
	}

	// Keep alive
	select {}
}
