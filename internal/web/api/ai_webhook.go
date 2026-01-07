package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"math/rand/v2"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gowvp/gb28181/internal/conf"
	"github.com/gowvp/gb28181/internal/core/event"
	"github.com/gowvp/gb28181/internal/core/ipc"
	"github.com/gowvp/gb28181/internal/rpc"
	"github.com/ixugo/goddd/pkg/conc"
	"github.com/ixugo/goddd/pkg/orm"
	"github.com/ixugo/goddd/pkg/system"
	"github.com/ixugo/goddd/pkg/web"
)

// AIWebhookAPI 处理 AI 分析服务的回调请求
type AIWebhookAPI struct {
	log       *slog.Logger
	conf      *conf.Bootstrap
	aiTasks   *conc.Map[string, struct{}]
	limiter   func(identifier string) bool
	ai        *rpc.AIClient
	eventCore event.Core
	ipcCore   ipc.Core
}

// NewAIWebhookAPI 创建 AI Webhook API 实例
func NewAIWebhookAPI(conf *conf.Bootstrap, eventCore event.Core, ipcCore ipc.Core) AIWebhookAPI {
	return AIWebhookAPI{
		log:       slog.With("hook", "ai"),
		conf:      conf,
		ai:        rpc.NewAIClient("127.0.0.1:50051"),
		aiTasks:   conc.NewMap[string, struct{}](),
		eventCore: eventCore,
		ipcCore:   ipcCore,
		limiter:   web.IDRateLimiter(0.2, 1, 3*time.Minute),
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

// onEvents 接收 AI 检测事件，按 label 分别存储到数据库，图片保存到 configs/events 目录
func (a AIWebhookAPI) onEvents(c *gin.Context, in *AIDetectionInput) (AIWebhookOutput, error) {
	if !a.limiter(in.CameraID) {
		return newAIWebhookOutputOK(), nil
	}
	ctx := c.Request.Context()

	a.log.InfoContext(ctx, "ai detection event",
		"camera_id", in.CameraID,
		"timestamp", in.Timestamp,
		"detection_count", len(in.Detections),
		"snapshot_size", fmt.Sprintf("%dx%d", in.SnapshotWidth, in.SnapshotHeight),
	)

	// 获取通道信息以确定 DID
	cid := in.CameraID
	var did string
	channel, err := a.ipcCore.GetChannel(ctx, cid)
	if err == nil && channel != nil {
		did = channel.DID
	}

	// 保存图片并获取相对路径
	var imagePath string
	if in.Snapshot != "" {
		var err error
		imagePath, err = saveEventSnapshot(cid, in.Timestamp, in.Snapshot)
		if err != nil {
			a.log.ErrorContext(ctx, "save snapshot failed", "err", err)
		}
	}

	// 按 label 分别存储事件，每个 label 是一个独立事件
	for i, det := range in.Detections {
		a.log.InfoContext(ctx, "detection detail",
			"index", i,
			"label", det.Label,
			"confidence", det.Confidence,
			"box", fmt.Sprintf("(%d,%d)-(%d,%d)", det.Box.XMin, det.Box.YMin, det.Box.XMax, det.Box.YMax),
			"area", det.Area,
		)

		zonesJSON, _ := json.Marshal(det.Box)

		eventInput := &event.AddEventInput{
			DID:       did,
			CID:       cid,
			StartedAt: in.Timestamp,
			EndedAt:   in.Timestamp,
			Label:     det.Label,
			Score:     float32(det.Confidence),
			Zones:     string(zonesJSON),
			ImagePath: imagePath,
			Model:     "yolo11n",
		}

		if _, err := a.eventCore.AddEvent(ctx, eventInput); err != nil {
			a.log.ErrorContext(ctx, "save event failed",
				"label", det.Label,
				"err", err,
			)
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

// saveEventSnapshot 将 Base64 编码的快照保存到 configs/events/{cid}/ 目录
// 返回相对路径: cid/年月日时分秒_随机6位.jpg
func saveEventSnapshot(cid string, t orm.Time, snapshotB64 string) (string, error) {
	eventsDir := filepath.Join(system.Getwd(), "configs", "events")

	data, err := base64.StdEncoding.DecodeString(snapshotB64)
	if err != nil {
		return "", fmt.Errorf("decode base64: %w", err)
	}

	randomSuffix := fmt.Sprintf("%06d", rand.IntN(1000000))
	filename := fmt.Sprintf("%s_%s.jpg", t.Format("20060102150405"), randomSuffix)

	relativePath := filepath.Join(cid, filename)
	fullPath := filepath.Join(eventsDir, relativePath)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
		return "", fmt.Errorf("create events dir: %w", err)
	}

	if err := os.WriteFile(fullPath, data, 0o644); err != nil {
		return "", fmt.Errorf("write file: %w", err)
	}

	slog.Info("event snapshot saved", "path", fullPath, "size", len(data))
	return relativePath, nil
}
