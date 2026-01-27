package gbs

import (
	"context"
	"fmt"

	"github.com/gowvp/owl/internal/core/ipc"
	"github.com/gowvp/owl/pkg/gbs/sip"
)

// PTZDirection PTZ 方向枚举
type PTZDirection int

const (
	PTZDirectionUp         PTZDirection = 0 // 上
	PTZDirectionDown       PTZDirection = 1 // 下
	PTZDirectionLeft       PTZDirection = 2 // 左
	PTZDirectionRight      PTZDirection = 3 // 右
	PTZDirectionUpLeft     PTZDirection = 4 // 左上
	PTZDirectionUpRight    PTZDirection = 5 // 右上
	PTZDirectionDownLeft   PTZDirection = 6 // 左下
	PTZDirectionDownRight  PTZDirection = 7 // 右下
	PTZDirectionZoomIn     PTZDirection = 8 // 焦距变大(倍率变大)
	PTZDirectionZoomOut    PTZDirection = 9 // 焦距变小(倍率变小)
	PTZDirectionFocusNear  PTZDirection = 10 // 焦点前调
	PTZDirectionFocusFar   PTZDirection = 11 // 焦点后调
	PTZDirectionIrisOpen   PTZDirection = 12 // 光圈扩大
	PTZDirectionIrisClose  PTZDirection = 13 // 光圈缩小
)

// PTZInput PTZ 控制输入参数
type PTZInput struct {
	Channel   *ipc.Channel
	Direction PTZDirection // 方向: 0-上, 1-下, 2-左, 3-右, 4-左上, 5-右上, 6-左下, 7-右下, 8-放大, 9-缩小, 10-焦点前调, 11-焦点后调, 12-光圈扩大, 13-光圈缩小
	Speed     int          // 速度: 0-255, 其中 0 表示停止
	Horizontal int          // 水平速度: 0-255 (可选，0表示停止)
	Vertical   int          // 垂直速度: 0-255 (可选，0表示停止)
	Zoom       int          // 变倍速度: 0-15 (可选，0表示停止)
}

// PTZPresetInput PTZ 预置位输入参数
type PTZPresetInput struct {
	Channel  *ipc.Channel
	Command  string // 命令: SetPreset-设置, GotoPreset-调用, RemovePreset-删除
	PresetID int    // 预置位编号: 1-255
}

// PTZCruiseInput PTZ 巡航控制输入参数
type PTZCruiseInput struct {
	Channel *ipc.Channel
	Command string // 命令: SetCruise-设置巡航, GotoCruise-调用巡航, RemoveCruise-删除巡航
	CruiseID int   // 巡航组编号: 1-255
	PresetID int   // 预置位编号: 1-255 (仅SetCruise需要)
	Speed    int   // 速度: 0-255 (仅SetCruise需要)
	StayTime int   // 停留时间，单位秒 (仅SetCruise需要)
}

// PTZScanInput PTZ 扫描控制输入参数
type PTZScanInput struct {
	Channel *ipc.Channel
	Command string // 命令: SetScan-设置扫描, GotoScan-调用扫描, RemoveScan-删除扫描
	ScanID  int    // 扫描组编号: 1-255
	Speed   int    // 速度: 0-255 (仅SetScan需要)
}

// PTZControl PTZ 方向控制
// GB/T 28181-2022 9.4 远程控制
func (g *GB28181API) PTZControl(ctx context.Context, in *PTZInput) error {
	ch, ok := g.svr.memoryStorer.GetChannel(in.Channel.DeviceID, in.Channel.ChannelID)
	if !ok {
		return ErrChannelNotExist
	}

	if !ch.device.IsOnline {
		return ErrDeviceOffline
	}

	// 构建 PTZ 控制码
	// 第1-4字节: A5 0F 01 00 (固定头)
	// 第5字节: 组合码(方向+速度)
	// 第6字节: 水平速度
	// 第7字节: 垂直速度
	// 第8字节: 变倍/焦点/光圈控制码(高4位)和速度(低4位)
	ptzCmd := make([]byte, 8)
	ptzCmd[0] = 0xA5
	ptzCmd[1] = 0x0F
	ptzCmd[2] = 0x01
	ptzCmd[3] = 0x00

	// 第5字节: 方向控制
	switch in.Direction {
	case PTZDirectionUp:
		ptzCmd[4] = 0x08
	case PTZDirectionDown:
		ptzCmd[4] = 0x04
	case PTZDirectionLeft:
		ptzCmd[4] = 0x02
	case PTZDirectionRight:
		ptzCmd[4] = 0x01
	case PTZDirectionUpLeft:
		ptzCmd[4] = 0x0A
	case PTZDirectionUpRight:
		ptzCmd[4] = 0x09
	case PTZDirectionDownLeft:
		ptzCmd[4] = 0x06
	case PTZDirectionDownRight:
		ptzCmd[4] = 0x05
	case PTZDirectionZoomIn:
		ptzCmd[4] = 0x10
	case PTZDirectionZoomOut:
		ptzCmd[4] = 0x20
	default:
		ptzCmd[4] = 0x00 // 停止
	}

	// 第6字节: 水平速度 (0-255)
	if in.Horizontal > 0 && in.Horizontal <= 255 {
		ptzCmd[5] = byte(in.Horizontal)
	} else if in.Speed > 0 && in.Speed <= 255 {
		ptzCmd[5] = byte(in.Speed)
	}

	// 第7字节: 垂直速度 (0-255)
	if in.Vertical > 0 && in.Vertical <= 255 {
		ptzCmd[6] = byte(in.Vertical)
	} else if in.Speed > 0 && in.Speed <= 255 {
		ptzCmd[6] = byte(in.Speed)
	}

	// 第8字节: 变倍/焦点/光圈控制 (高4位)和速度(低4位)
	zoom := byte(0)
	if in.Zoom > 0 && in.Zoom <= 15 {
		zoom = byte(in.Zoom) & 0x0F
	} else if in.Speed > 0 && in.Speed <= 15 {
		zoom = byte(in.Speed) & 0x0F
	}

	switch in.Direction {
	case PTZDirectionZoomIn:
		ptzCmd[7] = (1 << 4) | zoom // 焦距变大
	case PTZDirectionZoomOut:
		ptzCmd[7] = (2 << 4) | zoom // 焦距变小
	case PTZDirectionFocusNear:
		ptzCmd[7] = (4 << 4) | zoom // 焦点前调
	case PTZDirectionFocusFar:
		ptzCmd[7] = (8 << 4) | zoom // 焦点后调
	case PTZDirectionIrisOpen:
		ptzCmd[7] = (3 << 4) | zoom // 光圈扩大
	case PTZDirectionIrisClose:
		ptzCmd[7] = (4 << 4) | zoom // 光圈缩小
	default:
		ptzCmd[7] = zoom
	}

	// 发送 PTZ 控制指令
	body := sip.GetPTZControlXML(in.Channel.ChannelID, ptzCmd)
	_, err := g.svr.wrapRequest(ch, sip.MethodMessage, &sip.ContentTypeXML, body)
	return err
}

// PTZPresetControl PTZ 预置位控制
// GB/T 28181-2022 9.4 远程控制
func (g *GB28181API) PTZPresetControl(ctx context.Context, in *PTZPresetInput) error {
	ch, ok := g.svr.memoryStorer.GetChannel(in.Channel.DeviceID, in.Channel.ChannelID)
	if !ok {
		return ErrChannelNotExist
	}

	if !ch.device.IsOnline {
		return ErrDeviceOffline
	}

	// 构建预置位控制码
	ptzCmd := make([]byte, 8)
	ptzCmd[0] = 0xA5
	ptzCmd[1] = 0x0F
	ptzCmd[2] = 0x01
	ptzCmd[3] = 0x00
	ptzCmd[4] = 0x00
	ptzCmd[5] = 0x00
	ptzCmd[6] = 0x00

	// 第8字节: 预置位操作(高4位)和预置位编号(低4位，当高4位不为0时，低4位表示预置位编号的高4位)
	// 第7字节: 预置位编号的低8位
	switch in.Command {
	case "SetPreset":
		ptzCmd[7] = 0x81 // 设置预置位
	case "GotoPreset":
		ptzCmd[7] = 0x82 // 调用预置位
	case "RemovePreset":
		ptzCmd[7] = 0x83 // 删除预置位
	default:
		return fmt.Errorf("invalid preset command: %s", in.Command)
	}

	// 预置位编号
	if in.PresetID < 1 || in.PresetID > 255 {
		return fmt.Errorf("preset ID must be between 1 and 255")
	}
	ptzCmd[6] = byte(in.PresetID)

	// 发送 PTZ 控制指令
	body := sip.GetPTZControlXML(in.Channel.ChannelID, ptzCmd)
	_, err := g.svr.wrapRequest(ch, sip.MethodMessage, &sip.ContentTypeXML, body)
	return err
}

// PTZCruiseControl PTZ 巡航控制
// GB/T 28181-2022 9.4 远程控制
func (g *GB28181API) PTZCruiseControl(ctx context.Context, in *PTZCruiseInput) error {
	ch, ok := g.svr.memoryStorer.GetChannel(in.Channel.DeviceID, in.Channel.ChannelID)
	if !ok {
		return ErrChannelNotExist
	}

	if !ch.device.IsOnline {
		return ErrDeviceOffline
	}

	// 构建巡航控制码
	ptzCmd := make([]byte, 8)
	ptzCmd[0] = 0xA5
	ptzCmd[1] = 0x0F
	ptzCmd[2] = 0x01
	ptzCmd[3] = 0x00

	switch in.Command {
	case "SetCruise":
		// 设置巡航: 添加预置位到巡航组
		ptzCmd[4] = 0x00
		ptzCmd[5] = byte(in.PresetID) // 预置位编号
		ptzCmd[6] = byte(in.Speed)    // 速度
		ptzCmd[7] = 0x84 | byte(in.CruiseID) // 设置巡航(0x84) + 巡航组编号
	case "GotoCruise":
		// 调用巡航
		ptzCmd[4] = 0x00
		ptzCmd[5] = 0x00
		ptzCmd[6] = byte(in.CruiseID) // 巡航组编号
		ptzCmd[7] = 0x85 // 调用巡航
	case "RemoveCruise":
		// 删除巡航
		ptzCmd[4] = 0x00
		ptzCmd[5] = 0x00
		ptzCmd[6] = byte(in.CruiseID) // 巡航组编号
		ptzCmd[7] = 0x86 // 删除巡航
	default:
		return fmt.Errorf("invalid cruise command: %s", in.Command)
	}

	// 发送 PTZ 控制指令
	body := sip.GetPTZControlXML(in.Channel.ChannelID, ptzCmd)
	_, err := g.svr.wrapRequest(ch, sip.MethodMessage, &sip.ContentTypeXML, body)
	return err
}

// PTZScanControl PTZ 扫描控制
// GB/T 28181-2022 9.4 远程控制
func (g *GB28181API) PTZScanControl(ctx context.Context, in *PTZScanInput) error {
	ch, ok := g.svr.memoryStorer.GetChannel(in.Channel.DeviceID, in.Channel.ChannelID)
	if !ok {
		return ErrChannelNotExist
	}

	if !ch.device.IsOnline {
		return ErrDeviceOffline
	}

	// 构建扫描控制码
	ptzCmd := make([]byte, 8)
	ptzCmd[0] = 0xA5
	ptzCmd[1] = 0x0F
	ptzCmd[2] = 0x01
	ptzCmd[3] = 0x00
	ptzCmd[4] = 0x00

	switch in.Command {
	case "SetScan":
		// 设置扫描: 设置扫描的起始位置和结束位置
		ptzCmd[5] = byte(in.Speed)    // 速度
		ptzCmd[6] = byte(in.ScanID)   // 扫描组编号
		ptzCmd[7] = 0x87 // 设置扫描
	case "GotoScan":
		// 调用扫描
		ptzCmd[5] = 0x00
		ptzCmd[6] = byte(in.ScanID) // 扫描组编号
		ptzCmd[7] = 0x88 // 调用扫描
	case "RemoveScan":
		// 删除扫描
		ptzCmd[5] = 0x00
		ptzCmd[6] = byte(in.ScanID) // 扫描组编号
		ptzCmd[7] = 0x89 // 删除扫描
	default:
		return fmt.Errorf("invalid scan command: %s", in.Command)
	}

	// 发送 PTZ 控制指令
	body := sip.GetPTZControlXML(in.Channel.ChannelID, ptzCmd)
	_, err := g.svr.wrapRequest(ch, sip.MethodMessage, &sip.ContentTypeXML, body)
	return err
}
