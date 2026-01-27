package api

import (
	"github.com/gin-gonic/gin"
	"github.com/gowvp/owl/internal/core/ipc"
	"github.com/ixugo/goddd/pkg/reason"
	"github.com/ixugo/goddd/pkg/web"
)

// PTZAPI PTZ 控制 API
type PTZAPI struct {
	ipc ipc.Core
}

// NewPTZAPI 创建 PTZ API
func NewPTZAPI(bundle IPCBundle) PTZAPI {
	return PTZAPI{ipc: bundle.Core}
}

// RegisterPTZ 注册 PTZ 路由
func RegisterPTZ(g gin.IRouter, api PTZAPI, handler ...gin.HandlerFunc) {
	group := g.Group("/channels/:id/ptz", handler...)
	group.POST("/control", web.WrapH(api.ptzControl))        // PTZ 方向控制
	group.POST("/preset", web.WrapH(api.ptzPresetControl))   // PTZ 预置位控制
	group.POST("/stop", web.WrapH(api.ptzStop))              // PTZ 停止
}

// PTZControlInput PTZ 方向控制输入参数
type PTZControlInput struct {
	Direction  int `json:"direction" binding:"required,min=0,max=13"` // 方向: 0-上, 1-下, 2-左, 3-右, 4-左上, 5-右上, 6-左下, 7-右下, 8-放大, 9-缩小, 10-焦点前调, 11-焦点后调, 12-光圈扩大, 13-光圈缩小
	Speed      int `json:"speed" binding:"min=0,max=255"`             // 速度: 0-255
	Horizontal int `json:"horizontal" binding:"min=0,max=255"`        // 水平速度: 0-255 (可选)
	Vertical   int `json:"vertical" binding:"min=0,max=255"`          // 垂直速度: 0-255 (可选)
	Zoom       int `json:"zoom" binding:"min=0,max=15"`               // 变倍速度: 0-15 (可选)
}

// PTZPresetInput PTZ 预置位控制输入参数
type PTZPresetInput struct {
	Command  string `json:"command" binding:"required,oneof=SetPreset GotoPreset RemovePreset"` // 命令: SetPreset-设置, GotoPreset-调用, RemovePreset-删除
	PresetID int    `json:"preset_id" binding:"required,min=1,max=255"`                         // 预置位编号: 1-255
}

// ptzControl PTZ 方向控制
// @Summary PTZ 方向控制
// @Description 控制云台的方向、焦距、光圈等
// @Tags PTZ
// @Accept json
// @Produce json
// @Param id path string true "通道 ID"
// @Param input body PTZControlInput true "PTZ 控制参数"
// @Success 200 {object} gin.H
// @Router /channels/{id}/ptz/control [post]
func (a PTZAPI) ptzControl(c *gin.Context, in *PTZControlInput) (any, error) {
	channelID := c.Param("id")
	ctx := c.Request.Context()

	// 获取通道信息
	channel, err := a.ipc.GetChannel(ctx, channelID)
	if err != nil {
		return nil, reason.ErrBadRequest.SetMsg("通道不存在")
	}

	// 检查协议是否支持 PTZ 控制
	protocol, err := a.ipc.GetProtocol(channel.Type)
	if err != nil {
		return nil, reason.ErrBadRequest.SetMsg("不支持的协议类型")
	}

	ptzController, ok := protocol.(ipc.PTZController)
	if !ok {
		return nil, reason.ErrBadRequest.SetMsg("该协议不支持 PTZ 控制")
	}

	// 执行 PTZ 控制
	if err := ptzController.PTZControl(ctx, channel, in.Direction, in.Speed, in.Horizontal, in.Vertical, in.Zoom); err != nil {
		return nil, reason.ErrInternalServer.SetMsg(err.Error())
	}

	return gin.H{"msg": "PTZ 控制成功"}, nil
}

// ptzPresetControl PTZ 预置位控制
// @Summary PTZ 预置位控制
// @Description 设置、调用或删除预置位
// @Tags PTZ
// @Accept json
// @Produce json
// @Param id path string true "通道 ID"
// @Param input body PTZPresetInput true "预置位控制参数"
// @Success 200 {object} gin.H
// @Router /channels/{id}/ptz/preset [post]
func (a PTZAPI) ptzPresetControl(c *gin.Context, in *PTZPresetInput) (any, error) {
	channelID := c.Param("id")
	ctx := c.Request.Context()

	// 获取通道信息
	channel, err := a.ipc.GetChannel(ctx, channelID)
	if err != nil {
		return nil, reason.ErrBadRequest.SetMsg("通道不存在")
	}

	// 检查协议是否支持 PTZ 控制
	protocol, err := a.ipc.GetProtocol(channel.Type)
	if err != nil {
		return nil, reason.ErrBadRequest.SetMsg("不支持的协议类型")
	}

	ptzController, ok := protocol.(ipc.PTZController)
	if !ok {
		return nil, reason.ErrBadRequest.SetMsg("该协议不支持 PTZ 控制")
	}

	// 执行预置位控制
	if err := ptzController.PTZPresetControl(ctx, channel, in.Command, in.PresetID); err != nil {
		return nil, reason.ErrInternalServer.SetMsg(err.Error())
	}

	return gin.H{"msg": "预置位控制成功"}, nil
}

// ptzStop PTZ 停止
// @Summary PTZ 停止
// @Description 停止 PTZ 运动
// @Tags PTZ
// @Accept json
// @Produce json
// @Param id path string true "通道 ID"
// @Success 200 {object} gin.H
// @Router /channels/{id}/ptz/stop [post]
func (a PTZAPI) ptzStop(c *gin.Context, _ *struct{}) (any, error) {
	channelID := c.Param("id")
	ctx := c.Request.Context()

	// 获取通道信息
	channel, err := a.ipc.GetChannel(ctx, channelID)
	if err != nil {
		return nil, reason.ErrBadRequest.SetMsg("通道不存在")
	}

	// 检查协议是否支持 PTZ 控制
	protocol, err := a.ipc.GetProtocol(channel.Type)
	if err != nil {
		return nil, reason.ErrBadRequest.SetMsg("不支持的协议类型")
	}

	ptzController, ok := protocol.(ipc.PTZController)
	if !ok {
		return nil, reason.ErrBadRequest.SetMsg("该协议不支持 PTZ 控制")
	}

	// 执行停止控制 (速度设为 0)
	if err := ptzController.PTZControl(ctx, channel, 0, 0, 0, 0, 0); err != nil {
		return nil, reason.ErrInternalServer.SetMsg(err.Error())
	}

	return gin.H{"msg": "PTZ 停止成功"}, nil
}
