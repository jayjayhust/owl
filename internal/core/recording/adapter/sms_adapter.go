package adapter

import (
	"github.com/gowvp/owl/internal/core/recording"
	"github.com/gowvp/owl/internal/core/sms"
	"github.com/gowvp/owl/pkg/zlm"
)

var _ recording.SMSProvider = (*SMSAdapter)(nil)

// SMSAdapter 实现 recording.SMSProvider 接口
// 将 sms.Core 的录制能力适配给 recording 领域使用
type SMSAdapter struct {
	smsCore sms.Core
}

// NewSMSAdapter 创建 SMS 适配器，返回 recording.SMSProvider 接口
// Wire 通过此函数自动绑定 sms.Core -> recording.SMSProvider
func NewSMSAdapter(smsCore sms.Core) recording.SMSProvider {
	return &SMSAdapter{smsCore: smsCore}
}

// StartRecord 启动录制
func (a *SMSAdapter) StartRecord(app, stream, customPath string, maxSecond int) error {
	ms, err := a.smsCore.GetDefaultMediaServer()
	if err != nil {
		return err
	}
	_, err = a.smsCore.StartRecord(ms, zlm.StartRecordRequest{
		Type:       1, // MP4
		Vhost:      "__defaultVhost__",
		App:        app,
		Stream:     stream,
		CustomPath: customPath,
		MaxSecond:  maxSecond,
	})
	return err
}

// StopRecord 停止录制
func (a *SMSAdapter) StopRecord(app, stream string) error {
	ms, err := a.smsCore.GetDefaultMediaServer()
	if err != nil {
		return err
	}
	_, err = a.smsCore.StopRecord(ms, zlm.StopRecordRequest{
		Type:   1, // MP4
		Vhost:  "__defaultVhost__",
		App:    app,
		Stream: stream,
	})
	return err
}
