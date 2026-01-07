package api

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gowvp/gb28181/internal/conf"
	"github.com/gowvp/gb28181/internal/rpc"
	"github.com/ixugo/goddd/pkg/conc"
	"github.com/ixugo/goddd/pkg/system"
	"github.com/ixugo/goddd/pkg/web"
)

// AIWebhookAPI 处理 AI 分析服务的回调请求
type AIWebhookAPI struct {
	log     *slog.Logger
	conf    *conf.Bootstrap
	aiTasks *conc.Map[string, struct{}]
	ai      *rpc.AIClient
}

// NewAIWebhookAPI 创建 AI Webhook API 实例
func NewAIWebhookAPI(conf *conf.Bootstrap) AIWebhookAPI {
	return AIWebhookAPI{
		log:     slog.With("hook", "ai"),
		conf:    conf,
		ai:      rpc.NewAIClient("127.0.0.1:50051"),
		aiTasks: conc.NewMap[string, struct{}](),
	}
}

// registerAIWebhookAPI 注册 AI 回调路由，接收来自 Python AI 服务的各类事件通知
func registerAIWebhookAPI(r gin.IRouter, api AIWebhookAPI, handler ...gin.HandlerFunc) {
	group := r.Group("/ai", handler...)
	group.POST("/keepalive", web.WrapH(api.onKeepalive))
	group.POST("/started", web.WrapH(api.onStarted))
	group.POST("/events", web.WrapH(api.onEvents))
	group.POST("/stopped", web.WrapH(api.onStopped))
}

// onKeepalive 接收 AI 服务心跳，用于监控 AI 服务存活状态
func (a AIWebhookAPI) onKeepalive(c *gin.Context, in *AIKeepaliveInput) (AIWebhookOutput, error) {
	var activeStreams int
	var uptimeSeconds int64
	if in.Stats != nil {
		activeStreams = in.Stats.ActiveStreams
		uptimeSeconds = in.Stats.UptimeSeconds
	}
	a.log.InfoContext(c.Request.Context(), "ai keepalive",
		"timestamp", in.Timestamp,
		"message", in.Message,
		"active_streams", activeStreams,
		"uptime_seconds", uptimeSeconds,
	)
	return newAIWebhookOutputOK(), nil
}

// onStarted 接收 AI 服务启动通知，确认 AI 服务已就绪
func (a AIWebhookAPI) onStarted(c *gin.Context, in *AIStartedInput) (AIWebhookOutput, error) {
	a.log.InfoContext(c.Request.Context(), "ai started",
		"timestamp", in.Timestamp,
		"message", in.Message,
	)
	return newAIWebhookOutputOK(), nil
}

// onEvents 接收 AI 检测事件，将快照保存到临时目录
func (a AIWebhookAPI) onEvents(c *gin.Context, in *AIDetectionInput) (AIWebhookOutput, error) {
	a.log.InfoContext(c.Request.Context(), "ai detection event",
		"camera_id", in.CameraID,
		"timestamp", in.Timestamp,
		"detection_count", len(in.Detections),
		"snapshot_size", fmt.Sprintf("%dx%d", in.SnapshotWidth, in.SnapshotHeight),
	)

	for i, det := range in.Detections {
		a.log.InfoContext(c.Request.Context(), "detection detail",
			"index", i,
			"label", det.Label,
			"confidence", det.Confidence,
			"box", fmt.Sprintf("(%d,%d)-(%d,%d)", det.Box.XMin, det.Box.YMin, det.Box.XMax, det.Box.YMax),
			"area", det.Area,
		)
	}

	if in.Snapshot != "" {
		if err := saveSnapshot(in.CameraID, in.Timestamp, in.Snapshot); err != nil {
			a.log.ErrorContext(c.Request.Context(), "save snapshot failed", "err", err)
		}
	}

	return newAIWebhookOutputOK(), nil
}

// onStopped 接收 AI 任务停止通知，记录停止原因
func (a AIWebhookAPI) onStopped(c *gin.Context, in *AIStoppedInput) (AIWebhookOutput, error) {
	a.log.InfoContext(c.Request.Context(), "ai task stopped",
		"camera_id", in.CameraID,
		"timestamp", in.Timestamp,
		"reason", in.Reason,
		"message", in.Message,
	)
	a.aiTasks.Delete(in.CameraID)
	return newAIWebhookOutputOK(), nil
}

// saveSnapshot 将 Base64 编码的快照保存到临时目录
func saveSnapshot(cameraID string, timestamp int64, snapshotB64 string) error {
	tmpDir := filepath.Join(system.Getwd(), "configs", "demo")

	data, err := base64.StdEncoding.DecodeString(snapshotB64)
	if err != nil {
		return fmt.Errorf("decode base64: %w", err)
	}

	t := time.UnixMilli(timestamp)
	filename := fmt.Sprintf("%s_%s.jpg", cameraID, t.Format("20060102_150405"))
	filePath := filepath.Join(tmpDir, filename)
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return fmt.Errorf("create tmp dir: %w", err)
	}

	if err := os.WriteFile(filePath, data, 0o644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	slog.Info("snapshot saved", "path", filePath, "size", len(data))
	return nil
}
