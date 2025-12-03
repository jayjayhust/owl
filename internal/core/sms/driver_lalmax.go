package sms

import (
	"context"
	"fmt"

	"github.com/gowvp/gb28181/pkg/lalmax"
	"github.com/gowvp/gb28181/pkg/zlm"
)

const (
	PullTimeoutMs = 10000
	PullRetryNum  = 3
)

var _ Driver = (*LalmaxDriver)(nil)

type LalmaxDriver struct {
	engine lalmax.Engine
}

// AddStreamProxy implements Driver.
func (l *LalmaxDriver) AddStreamProxy(ctx context.Context, ms *MediaServer, req *AddStreamProxyRequest) (*zlm.AddStreamProxyResponse, error) {
	engine := l.withConfig(ms)
	resp, err := engine.CtrlStartRelayPull(ctx, lalmax.ApiCtrlStartRelayPullReq{
		StreamName:    req.Stream,
		Url:           req.URL,
		PullTimeoutMs: PullTimeoutMs,
		PullRetryNum:  PullRetryNum,
		RtspMode:      req.RTPType,
	})
	if err != nil {
		return nil, err
	}
	var result zlm.AddStreamProxyResponse
	result.Data.Key = resp.Data.SessionId
	return &result, nil
}

// CloseRTPServer implements Driver.
func (l *LalmaxDriver) CloseRTPServer(ctx context.Context, ms *MediaServer, req *zlm.CloseRTPServerRequest) (*zlm.CloseRTPServerResponse, error) {
	panic("unimplemented")
}

// Connect implements Driver.
func (l *LalmaxDriver) Connect(ctx context.Context, ms *MediaServer) error {
	engine := l.withConfig(ms)
	resp, err := engine.GetServerConfig()
	if err != nil {
		return err
	}
	// if len(resp.Data) == 0 {
	// return fmt.Errorf("Lalmax 服务节点配置为空")
	// }
	_ = resp
	panic("unimplemented")
}

// GetSnapshot implements Driver.
func (l *LalmaxDriver) GetSnapshot(ctx context.Context, ms *MediaServer, req *zlm.GetSnapRequest) ([]byte, error) {
	panic("unimplemented")
}

// OpenRTPServer implements Driver.
func (l *LalmaxDriver) OpenRTPServer(ctx context.Context, ms *MediaServer, req *zlm.OpenRTPServerRequest) (*zlm.OpenRTPServerResponse, error) {
	engine := l.withConfig(ms)
	resp, err := engine.ApiCtrlStartRtpPub(ctx, lalmax.ApiCtrlStartRtpPubReq{
		StreamName:      req.StreamID,
		Port:            req.Port,
		TimeoutMs:       PullTimeoutMs,
		IsTcpFlag:       0,
		IsWaitKeyFrame:  0,
		IsTcpActive:     false,
		DebugDumpPacket: "",
	})
	if err != nil {
		return nil, err
	}
	return &zlm.OpenRTPServerResponse{
		Port: resp.Data.Port,
	}, nil
}

// Ping implements Driver.
func (l *LalmaxDriver) Ping(ctx context.Context, ms *MediaServer) error {
	return nil
}

// Protocol implements Driver.
func (l *LalmaxDriver) Protocol() string {
	return ProtocolLalmax
}

// Setup implements Driver.
func (l *LalmaxDriver) Setup(ctx context.Context, ms *MediaServer, webhookURL string) error {
	panic("unimplemented")
}

func NewLalmaxDriver() *LalmaxDriver {
	return &LalmaxDriver{}
}

func (l *LalmaxDriver) withConfig(ms *MediaServer) lalmax.Engine {
	url := fmt.Sprintf("http://%s:%d", ms.IP, ms.Ports.HTTP)
	return l.engine.SetConfig(lalmax.Config{
		URL:    url,
		Secret: ms.Secret,
	})
}
