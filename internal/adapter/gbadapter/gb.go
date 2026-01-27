package gbadapter

import (
	"context"

	"github.com/gowvp/owl/internal/core/ipc"
	"github.com/gowvp/owl/internal/core/sms"
	"github.com/gowvp/owl/pkg/gbs"
)

var _ ipc.Protocoler = (*Adapter)(nil)

type Adapter struct {
	adapter ipc.Adapter
	gbs     *gbs.Server
	smsCore sms.Core
}

// DeleteDevice implements ipc.Protocoler.
func (a *Adapter) DeleteDevice(ctx context.Context, device *ipc.Device) error {
	return nil
}

func NewAdapter(adapter ipc.Adapter, gbs *gbs.Server, smsCore sms.Core) *Adapter {
	return &Adapter{adapter: adapter, gbs: gbs, smsCore: smsCore}
}

// InitDevice implements ipc.Protocoler.
func (a *Adapter) InitDevice(ctx context.Context, device *ipc.Device) error {
	return nil
}

// OnStreamChanged implements ipc.Protocoler.
// 流注销时停止播放并更新播放状态（仅在 regist=false 时由 zlm_webhook 调用）
// GB28181 协议的 stream 就是 channel.ID，app 固定为 rtp
func (a *Adapter) OnStreamChanged(ctx context.Context, app, stream string) error {
	ch, err := a.adapter.GetChannel(ctx, stream)
	if err != nil {
		return err
	}
	// 更新播放状态为 false
	if err := a.adapter.EditPlayingByID(ctx, ch.ID, false); err != nil {
		return err
	}
	return a.gbs.StopPlay(ctx, &gbs.StopPlayInput{Channel: ch})
}

// OnStreamNotFound implements ipc.Protocoler.
func (a *Adapter) OnStreamNotFound(ctx context.Context, app string, stream string) error {
	ch, err := a.adapter.GetChannel(ctx, stream)
	if err != nil {
		return err
	}

	dev, err := a.adapter.GetDevice(ctx, ch.DID)
	if err != nil {
		return err
	}

	svr, err := a.smsCore.GetMediaServer(ctx, sms.DefaultMediaServerID)
	if err != nil {
		return err
	}

	return a.gbs.Play(&gbs.PlayInput{
		Channel:    ch,
		StreamMode: dev.StreamMode,
		SMS:        svr,
	})
}

// QueryCatalog implements ipc.Protocoler.
func (a *Adapter) QueryCatalog(ctx context.Context, device *ipc.Device) error {
	return a.gbs.QueryCatalog(device.DeviceID)
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
	return nil
}

// PTZControl implements ipc.Protocoler.
func (a *Adapter) PTZControl(ctx context.Context, channel *ipc.Channel, direction int, speed int, horizontal int, vertical int, zoom int) error {
	return a.gbs.PTZControl(ctx, &gbs.PTZInput{
		Channel:    channel,
		Direction:  gbs.PTZDirection(direction),
		Speed:      speed,
		Horizontal: horizontal,
		Vertical:   vertical,
		Zoom:       zoom,
	})
}

// PTZPresetControl implements ipc.Protocoler.
func (a *Adapter) PTZPresetControl(ctx context.Context, channel *ipc.Channel, command string, presetID int) error {
	return a.gbs.PTZPresetControl(ctx, &gbs.PTZPresetInput{
		Channel:  channel,
		Command:  command,
		PresetID: presetID,
	})
}

