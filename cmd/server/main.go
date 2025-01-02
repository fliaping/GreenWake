package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"my-wol/internal/api"
	"my-wol/internal/config"
)

func main() {
	// 添加命令行参数
	configPath := flag.String("config", "config.yml", "配置文件路径")
	flag.Parse()

	// 如果没有指定绝对路径，则使用相对于当前工作目录的路径
	if !filepath.IsAbs(*configPath) {
		workDir, err := os.Getwd()
		if err != nil {
			log.Fatalf("获取当前工作目录失败: %v", err)
		}
		*configPath = filepath.Join(workDir, *configPath)
	}

	log.Printf("使用配置文件: %s", *configPath)

	// 加载配置
	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 启动服务器
	server := api.NewServer(cfg)
	if err := server.Run(); err != nil {
		log.Fatalf("服务器启动失败: %v", err)
	}
}
