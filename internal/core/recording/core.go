package recording

import (
	"strings"

	"github.com/gowvp/owl/internal/conf"
)

// Storer data persistence
type Storer interface {
	Recording() RecordingStorer
}

// SMSProvider 流媒体服务提供者接口，解耦录制领域与 sms 领域
type SMSProvider interface {
	StartRecord(app, stream, customPath string, maxSecond int) error
	StopRecord(app, stream string) error
}

// Core business domain
type Core struct {
	store       Storer
	conf        *conf.ServerRecording
	smsProvider SMSProvider
}

type Option func(*Core)

// WithSMSProvider 注入流媒体服务提供者，用于控制录制
func WithSMSProvider(provider SMSProvider) Option {
	return func(c *Core) {
		c.smsProvider = provider
	}
}

// WithConfig 注入录制配置
func WithConfig(conf *conf.ServerRecording) Option {
	return func(c *Core) {
		c.conf = conf
	}
}

// NewCore create business domain
func NewCore(store Storer, opts ...Option) Core {
	c := Core{store: store}
	for _, opt := range opts {
		opt(&c)
	}
	return c
}

// IsEnabled 检查是否启用录制（全局开关）
// 使用反转逻辑：Disabled=false 表示启用录制
func (c Core) IsEnabled() bool {
	return c.conf != nil && !c.conf.Disabled
}

// GetFullPath 获取录像文件的完整路径
// relativePath 可能是相对于 StorageDir 的路径，也可能是完整路径
func (c Core) GetFullPath(relativePath string) string {
	if c.conf == nil || c.conf.StorageDir == "" {
		return relativePath
	}
	// 如果 relativePath 已经包含 StorageDir，直接返回
	if len(relativePath) > 0 && (relativePath[0] == '/' || strings.HasPrefix(relativePath, c.conf.StorageDir)) {
		return relativePath
	}
	return c.conf.StorageDir + "/" + relativePath
}
