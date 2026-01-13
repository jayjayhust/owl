package rtspadapter

import (
	"context"

	"github.com/gowvp/owl/internal/core/ipc"
	"github.com/gowvp/owl/internal/core/sms"
)

var _ ipc.Protocoler = (*Adapter)(nil)

type Adapter struct {
	ipcCore ipc.Core
	smsCore sms.Core
}

// DeleteDevice implements ipc.Protocoler.
func (a *Adapter) DeleteDevice(ctx context.Context, device *ipc.Device) error {
	return nil
}

func NewAdapter(ipcCore ipc.Core, smsCore sms.Core) *Adapter {
	return &Adapter{
		ipcCore: ipcCore,
		smsCore: smsCore,
	}
}

// InitDevice implements ipc.Protocoler.
func (a *Adapter) InitDevice(ctx context.Context, device *ipc.Device) error {
	panic("unimplemented")
}

// OnStreamChanged implements ipc.Protocoler.
func (a *Adapter) OnStreamChanged(ctx context.Context, stream string) error {
	return nil
}

// OnStreamNotFound implements ipc.Protocoler.
// 当流不存在时，从 Channel 获取配置并启动拉流代理
func (a *Adapter) OnStreamNotFound(ctx context.Context, app string, stream string) error {
	ch, err := a.ipcCore.GetChannel(ctx, stream)
	if err != nil {
		return err
	}

	svr, err := a.smsCore.GetMediaServer(ctx, sms.DefaultMediaServerID)
	if err != nil {
		return err
	}
	resp, err := a.smsCore.AddStreamProxy(svr, sms.AddStreamProxyRequest{
		App:     ch.App,
		Stream:  ch.Stream,
		URL:     ch.Config.SourceURL,
		RTPType: ch.Config.Transport,
	})
	if err != nil {
		return err
	}

	// 更新 StreamKey 和 IsOnline（用于后续关闭拉流代理）
	_, err = a.ipcCore.EditChannelConfigAndOnline(ctx, ch.ID, true, func(cfg *ipc.StreamConfig) {
		cfg.StreamKey = resp.Data.Key
	})

	return err
}

// QueryCatalog implements ipc.Protocoler.
func (a *Adapter) QueryCatalog(ctx context.Context, device *ipc.Device) error {
	panic("unimplemented")
}

// StartPlay implements ipc.Protocoler.
func (a *Adapter) StartPlay(ctx context.Context, device *ipc.Device, channel *ipc.Channel) (*ipc.PlayResponse, error) {
	panic("unimplemented")
}

// StopPlay implements ipc.Protocoler.
func (a *Adapter) StopPlay(ctx context.Context, device *ipc.Device, channel *ipc.Channel) error {
	panic("unimplemented")
}

// ValidateDevice implements ipc.Protocoler.
func (a *Adapter) ValidateDevice(ctx context.Context, device *ipc.Device) error {
	panic("unimplemented")
}
