package api

import (
	"fmt"
	"log"
	"os"

	"my-wol/internal/config"
	"my-wol/internal/service"

	"github.com/gin-gonic/gin"
)

type Server struct {
	cfg     *config.Config
	handler *Handler
	engine  *gin.Engine
}

func NewServer(cfg *config.Config) *Server {
	gin.SetMode(gin.ReleaseMode)

	// 设置日志级别
	logLevel := cfg.Log.Level
	if logLevel == "" {
		logLevel = "info"
	}
	gin.DebugPrintRouteFunc = func(httpMethod, absolutePath, handlerName string, nuHandlers int) {
		if logLevel == "debug" {
			log.Printf("%-6s %-25s --> %s (%d handlers)", httpMethod, absolutePath, handlerName, nuHandlers)
		}
	}

	r := gin.New()

	// 使用自定义日志中间件
	r.Use(gin.LoggerWithConfig(gin.LoggerConfig{
		SkipPaths: []string{"/api/pc/*/status"}, // 跳过状态检查的日志
		Output:    os.Stdout,
		Formatter: func(param gin.LogFormatterParams) string {
			if logLevel == "debug" || param.StatusCode >= 400 {
				return fmt.Sprintf("[GIN] %v | %3d | %13v | %15s | %-7s %s\n%s",
					param.TimeStamp.Format("2006/01/02 - 15:04:05"),
					param.StatusCode,
					param.Latency,
					param.ClientIP,
					param.Method,
					param.Path,
					param.ErrorMessage,
				)
			}
			return ""
		},
	}))
	r.Use(gin.Recovery())

	// 添加前端静态文件支持
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.StaticFile("/favicon.ico", "./web/dist/favicon.ico")

	pcService := service.NewPCService(cfg)
	clientService := service.NewClientService()
	forwardService := service.NewForwardService(cfg, pcService)
	handler := NewHandler(pcService, clientService, forwardService)

	api := r.Group("/api")
	{
		pc := api.Group("/pc")
		{
			pc.GET("/hosts", handler.GetHosts)
			pc.GET("/:hostName/status", handler.GetHostStatus)
			pc.GET("/:hostName/client_info", handler.GetHostClients)
			pc.GET("/:hostName/forward_channels", handler.GetHostChannels)
		}
	}

	return &Server{
		cfg:     cfg,
		handler: handler,
		engine:  r,
	}
}

func (s *Server) Run() error {
	addr := fmt.Sprintf(":%s", s.cfg.HTTP.Port)
	return s.engine.Run(addr)
}

func (s *Server) Close() {
	s.handler.clientService.Close()
	s.handler.forwardService.Close()
}
