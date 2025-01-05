package api

import (
	"fmt"
	"net/http"

	"my-wol/internal/model"
	"my-wol/internal/service"

	"crypto/md5"
	"encoding/hex"

	"my-wol/internal/config"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	pcService      *service.PCService
	clientService  *service.ClientService
	forwardService *service.ForwardService
	config         *config.Config
}

func NewHandler(pcService *service.PCService, clientService *service.ClientService, forwardService *service.ForwardService, config *config.Config) *Handler {
	return &Handler{
		pcService:      pcService,
		clientService:  clientService,
		forwardService: forwardService,
		config:         config,
	}
}

func (h *Handler) generateClientID(c *gin.Context) string {
	// 获取各种可用信息
	ip := c.ClientIP()
	userAgent := c.GetHeader("User-Agent")
	pageID := c.GetHeader("X-Page-ID")
	// X-Forwarded-For 可能包含多个IP，取最后一个
	forwardedFor := c.GetHeader("X-Forwarded-For")
	// X-Real-IP 通常是真实客户端IP
	realIP := c.GetHeader("X-Real-IP")

	// 组合所有信息生成唯一ID（不包含端口）
	idStr := fmt.Sprintf("%s|%s|%s|%s|%s", ip, pageID, forwardedFor, realIP, userAgent)
	// 使用 MD5 生成固定长度的 ID
	hash := md5.Sum([]byte(idStr))
	return hex.EncodeToString(hash[:])
}

func (h *Handler) GetHosts(c *gin.Context) {
	hosts := h.pcService.GetHosts()
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    hosts,
	})
}

func (h *Handler) GetHostStatus(c *gin.Context) {
	hostName := c.Param("hostName")
	keepAwake := c.Query("keepAwake") == "true"

	// 只有在保持唤醒的请求中才更新客户端状态
	if keepAwake {
		clientId := h.generateClientID(c)
		h.clientService.UpdateClient(
			clientId,
			c.GetHeader("User-Agent"),
			c.ClientIP(),
			"",
			hostName,
		)
	}

	status, err := h.pcService.GetHostStatus(hostName, keepAwake)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.Response{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    status,
	})
}

func (h *Handler) GetHostClients(c *gin.Context) {
	hostName := c.Param("hostName")
	clients := h.clientService.GetHostClients(hostName)
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    clients,
	})
}

func (h *Handler) GetHostChannels(c *gin.Context) {
	hostName := c.Param("hostName")
	channels := h.forwardService.GetHostChannels(hostName)
	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data:    channels,
	})
}

func (h *Handler) GetConfig(c *gin.Context) {
	refreshInterval := h.config.HTTP.RefreshInterval
	if refreshInterval <= 0 {
		refreshInterval = 30
	}

	c.JSON(http.StatusOK, model.Response{
		Success: true,
		Data: gin.H{
			"refreshInterval": refreshInterval,
		},
	})
}
