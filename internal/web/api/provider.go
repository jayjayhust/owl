package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/wire"
	"github.com/gowvp/owl/internal/adapter/gbadapter"
	"github.com/gowvp/owl/internal/adapter/onvifadapter"
	"github.com/gowvp/owl/internal/adapter/rtspadapter"
	"github.com/gowvp/owl/internal/conf"
	"github.com/gowvp/owl/internal/core/event"
	"github.com/gowvp/owl/internal/core/ipc"
	"github.com/gowvp/owl/internal/core/ipc/store/ipccache"
	"github.com/gowvp/owl/internal/core/ipc/store/ipcdb"
	"github.com/gowvp/owl/internal/core/proxy"
	"github.com/gowvp/owl/internal/core/push"
	"github.com/gowvp/owl/internal/core/push/store/pushdb"
	"github.com/gowvp/owl/internal/core/sms"
	"github.com/gowvp/owl/pkg/gbs"
	"github.com/ixugo/goddd/domain/uniqueid"
	"github.com/ixugo/goddd/domain/uniqueid/store/uniqueiddb"
	"github.com/ixugo/goddd/domain/version/versionapi"
	"github.com/ixugo/goddd/pkg/orm"
	"github.com/ixugo/goddd/pkg/web"
	"gorm.io/gorm"
)

var (
	ProviderVersionSet = wire.NewSet(versionapi.NewVersionCore)
	ProviderSet        = wire.NewSet(
		wire.Struct(new(Usecase), "*"),
		NewHTTPHandler,
		versionapi.New,
		NewSMSCore, NewSmsAPI,
		NewWebHookAPI,
		NewUniqueID,
		NewPushCore, NewPushAPI,
		gbs.NewServer,
		NewIPCStore, NewProtocols, NewIPCCore, NewIPCAPI, NewGBAdapter,
		NewProxyAPI, NewProxyCore,
		NewConfigAPI,
		NewUserAPI,
		NewAIWebhookAPIWithDeps,
		NewEventCore, NewEventAPI,
	)
)

type Usecase struct {
	Conf       *conf.Bootstrap
	DB         *gorm.DB
	Version    versionapi.API
	SMSAPI     SmsAPI
	WebHookAPI WebHookAPI
	UniqueID   uniqueid.Core
	MediaAPI   PushAPI
	GB28181API IPCAPI
	ProxyAPI   ProxyAPI
	ConfigAPI  ConfigAPI

	SipServer    *gbs.Server
	UserAPI      UserAPI
	AIWebhookAPI AIWebhookAPI

	EventAPI EventAPI
}

// NewHTTPHandler 生成Gin框架路由内容
func NewHTTPHandler(uc *Usecase) http.Handler {
	cfg := uc.Conf.Server
	if cfg.HTTP.JwtSecret == "" {
		uc.Conf.Server.HTTP.JwtSecret = orm.GenerateRandomString(32)
	}
	if !cfg.Debug {
		gin.SetMode(gin.ReleaseMode)
	}
	g := gin.New()
	g.NoRoute(func(c *gin.Context) {
		c.JSON(404, "来到了无人的荒漠")
	})
	// 如果启用了 Pprof，设置 Pprof 监控
	if cfg.HTTP.PProf.Enabled {
		web.SetupPProf(g, &cfg.HTTP.PProf.AccessIps) // 设置 Pprof 监控
	}

	setupRouter(g, uc) // 设置路由处理函数
	uc.Version.RecordVersion()
	return g // 返回配置好的 Gin 实例作为 http.Handler
}

// NewUniqueID 唯一 id 生成器
func NewUniqueID(db *gorm.DB) uniqueid.Core {
	return uniqueid.NewCore(uniqueiddb.NewDB(db).AutoMigrate(orm.GetEnabledAutoMigrate()), 5)
}

func NewPushCore(db *gorm.DB, uni uniqueid.Core) push.Core {
	return push.NewCore(pushdb.NewDB(db).AutoMigrate(orm.GetEnabledAutoMigrate()), uni)
}

func NewIPCStore(db *gorm.DB) ipc.Storer {
	return ipccache.NewCache(ipcdb.NewDB(db).AutoMigrate(orm.GetEnabledAutoMigrate()))
}

func NewGBAdapter(store ipc.Storer, uni uniqueid.Core) ipc.Adapter {
	return ipc.NewAdapter(
		store,
		uni,
	)
}

// NewProtocols 创建协议适配器映射
func NewProtocols(adapter ipc.Adapter, sms sms.Core, proxyCore *proxy.Core, gbs *gbs.Server) map[string]ipc.Protocoler {
	protocols := make(map[string]ipc.Protocoler)
	protocols[ipc.TypeOnvif] = onvifadapter.NewAdapter(adapter, sms)
	protocols[ipc.TypeRTSP] = rtspadapter.NewAdapter(proxyCore, sms)
	protocols[ipc.TypeGB28181] = gbadapter.NewAdapter(adapter, gbs, sms)
	return protocols
}

// NewAIWebhookAPIWithDeps 创建带依赖的 AI Webhook API
func NewAIWebhookAPIWithDeps(conf *conf.Bootstrap, eventCore event.Core, ipcCore ipc.Core) AIWebhookAPI {
	return NewAIWebhookAPI(conf, eventCore, ipcCore)
}
