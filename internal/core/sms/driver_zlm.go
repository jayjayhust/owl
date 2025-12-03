package sms

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gowvp/gb28181/pkg/zlm"
)

var _ Driver = (*ZLMDriver)(nil)

type ZLMDriver struct {
	engine zlm.Engine
}

// GetStreamLiveAddr implements Driver.
func (d *ZLMDriver) GetStreamLiveAddr(ctx context.Context, ms *MediaServer, httpPrefix, host, app, stream string) StreamLiveAddr {
	var out StreamLiveAddr
	wsPrefix := strings.Replace(strings.Replace(httpPrefix, "https", "wss", 1), "http", "ws", 1)
	out.WSFLV = fmt.Sprintf("%s/proxy/sms/%s.live.flv", wsPrefix, stream)
	out.HTTPFLV = fmt.Sprintf("%s/proxy/sms/%s.live.flv", httpPrefix, stream)
	out.HLS = fmt.Sprintf("%s/proxy/sms/%s/hls.fmp4.m3u8", httpPrefix, stream)
	rtcPrefix := strings.Replace(strings.Replace(httpPrefix, "https", "webrtc", 1), "http", "webrtc", 1)
	out.WebRTC = fmt.Sprintf("%s/proxy/sms/index/api/webrtc?app=%s&stream=%s&type=play", rtcPrefix, app, stream)
	out.RTMP = fmt.Sprintf("rtmp://%s:%d/%s", host, ms.Ports.RTMP, stream)
	out.RTSP = fmt.Sprintf("rtsp://%s:%d/%s", host, ms.Ports.RTSP, stream)
	return out
}

func NewZLMDriver() *ZLMDriver {
	return &ZLMDriver{
		engine: zlm.NewEngine(),
	}
}

func (d *ZLMDriver) Protocol() string {
	return ProtocolZLMediaKit
}

func (d *ZLMDriver) withConfig(ms *MediaServer) zlm.Engine {
	url := fmt.Sprintf("http://%s:%d", ms.IP, ms.Ports.HTTP)
	return d.engine.SetConfig(zlm.Config{
		URL:    url,
		Secret: ms.Secret,
	})
}

func (d *ZLMDriver) Connect(ctx context.Context, ms *MediaServer) error {
	engine := d.withConfig(ms)
	resp, err := engine.GetServerConfig()
	if err != nil {
		return err
	}
	if len(resp.Data) == 0 {
		return fmt.Errorf("ZLM 服务节点配置为空")
	}

	// 更新端口信息等
	// 注意：这里我们不直接修改数据库，而是修改传入的 ms 对象，调用者负责持久化或使用
	zlmConfig := resp.Data[0]
	http := ms.Ports.HTTP
	ms.Ports.FLV = http
	ms.Ports.WsFLV = http
	ms.Ports.HTTPS = zlmConfig.HTTPSslport
	ms.Ports.RTMP = zlmConfig.RtmpPort
	ms.Ports.RTMPs = zlmConfig.RtmpSslport
	ms.Ports.RTSP = zlmConfig.RtspPort
	ms.Ports.RTSPs = zlmConfig.RtspSslport
	ms.Ports.RTPPorxy = zlmConfig.RtpProxyPort
	ms.Ports.FLVs = zlmConfig.HTTPSslport
	ms.Ports.WsFLVs = zlmConfig.HTTPSslport
	ms.HookAliveInterval = 10
	ms.Status = true

	return nil
}

func (d *ZLMDriver) Setup(ctx context.Context, ms *MediaServer, webhookURL string) error {
	engine := d.withConfig(ms)

	// 构造配置请求
	req := zlm.SetServerConfigRequest{
		RtcExternIP:          zlm.NewString(ms.IP),
		GeneralMediaServerID: zlm.NewString(ms.ID),
		HookEnable:           zlm.NewString("1"),
		HookOnFlowReport:     zlm.NewString(""),
		HookOnPlay:           zlm.NewString(fmt.Sprintf("%s/on_play", webhookURL)),

		ProtocolEnableTs:      zlm.NewString("0"),
		ProtocolEnableFmp4:    zlm.NewString("0"),
		ProtocolEnableHls:     zlm.NewString("0"),
		ProtocolEnableHlsFmp4: zlm.NewString("1"),

		HookOnPublish:                  zlm.NewString(fmt.Sprintf("%s/on_publish", webhookURL)),
		HookOnStreamNoneReader:         zlm.NewString(fmt.Sprintf("%s/on_stream_none_reader", webhookURL)),
		GeneralStreamNoneReaderDelayMS: zlm.NewString("30000"),
		HookOnStreamNotFound:           zlm.NewString(fmt.Sprintf("%s/on_stream_not_found", webhookURL)),
		HookOnRecordTs:                 zlm.NewString(""),
		HookOnRtspAuth:                 zlm.NewString(""),
		HookOnRtspRealm:                zlm.NewString(""),
		HookOnShellLogin:               zlm.NewString(""),
		HookOnStreamChanged:            zlm.NewString(fmt.Sprintf("%s/on_stream_changed", webhookURL)),
		HookOnServerKeepalive:          zlm.NewString(fmt.Sprintf("%s/on_server_keepalive", webhookURL)),
		HookTimeoutSec:                 zlm.NewString("20"),
		HookAliveInterval:              zlm.NewString(fmt.Sprint(ms.HookAliveInterval)),
		ProtocolContinuePushMs:         zlm.NewString("3000"),
		RtpProxyPortRange:              &ms.RTPPortRange,
		FfmpegLog:                      zlm.NewString("./fflogs/ffmpeg.log"),
	}

	resp, err := engine.SetServerConfig(&req)
	if err != nil {
		return err
	}
	slog.Info("ZLM 服务节点配置设置成功", "changed", resp.Changed)
	return nil
}

func (d *ZLMDriver) Ping(ctx context.Context, ms *MediaServer) error {
	// 使用 getApiList 或简单的获取配置来探测是否存活
	engine := d.withConfig(ms)
	// 可以使用更轻量级的接口，这里暂时复用 GetServerConfig
	_, err := engine.GetServerConfig()
	return err
}

func (d *ZLMDriver) OpenRTPServer(ctx context.Context, ms *MediaServer, req *zlm.OpenRTPServerRequest) (*zlm.OpenRTPServerResponse, error) {
	engine := d.withConfig(ms)
	return engine.OpenRTPServer(*req)
}

func (d *ZLMDriver) CloseRTPServer(ctx context.Context, ms *MediaServer, req *zlm.CloseRTPServerRequest) (*zlm.CloseRTPServerResponse, error) {
	engine := d.withConfig(ms)
	return engine.CloseRTPServer(*req)
}

func (d *ZLMDriver) AddStreamProxy(ctx context.Context, ms *MediaServer, req *AddStreamProxyRequest) (*zlm.AddStreamProxyResponse, error) {
	engine := d.withConfig(ms)
	return engine.AddStreamProxy(zlm.AddStreamProxyRequest{
		Vhost:         "__defaultVhost__",
		App:           req.App,
		Stream:        req.Stream,
		URL:           req.URL,
		RTPType:       req.RTPType,
		RetryCount:    3,
		TimeoutSec:    PullTimeoutMs / 1000,
		EnableHLSFMP4: zlm.NewBool(true),
		EnableAudio:   zlm.NewBool(true),
		EnableRTSP:    zlm.NewBool(true),
		EnableRTMP:    zlm.NewBool(true),
		AddMuteAudio:  zlm.NewBool(true),
		AutoClose:     zlm.NewBool(true),
	})
}

func (d *ZLMDriver) GetSnapshot(ctx context.Context, ms *MediaServer, req *GetSnapRequest) ([]byte, error) {
	engine := d.withConfig(ms)
	return engine.GetSnap(req.GetSnapRequest)
}
