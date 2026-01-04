package api

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gowvp/gb28181/internal/conf"
	"github.com/ixugo/goddd/pkg/reason"
	"github.com/ixugo/goddd/pkg/web"
)

type UserAPI struct {
	conf   *conf.Bootstrap
	secret *Secret
}

type Secret struct {
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	expiredAt  time.Time
	m          sync.RWMutex
}

// TODO: 有概率存在过期导致登录解密识别
func (s *Secret) GetOrCreatePublicKey() (*rsa.PublicKey, error) {
	s.m.RLock()
	if s.publicKey != nil && time.Now().Before(s.expiredAt) {
		s.m.RUnlock()
		return s.publicKey, nil
	}
	s.m.RUnlock()

	s.m.Lock()
	defer s.m.Unlock()

	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}
	s.privateKey = privateKey
	s.publicKey = &privateKey.PublicKey
	s.expiredAt = time.Now().Add(1 * time.Hour)
	return s.publicKey, nil
}

func (s *Secret) MarshalPKIXPublicKey(key *rsa.PublicKey) []byte {
	publicKeyBytes, _ := x509.MarshalPKIXPublicKey(key)
	return pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
}

func (s *Secret) Decrypt(ciphertext string) ([]byte, error) {
	s.m.RLock()
	pri := s.privateKey
	s.m.RUnlock()
	if pri == nil {
		return nil, fmt.Errorf("请刷新页面后重试")
	}
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	plaintext, err := rsa.DecryptOAEP(sha256.New(), rand.Reader, pri, data, nil)
	if err != nil {
		return nil, err
	}
	return plaintext, nil
}

func NewUserAPI(conf *conf.Bootstrap) UserAPI {
	return UserAPI{
		conf:   conf,
		secret: &Secret{},
	}
}

func RegisterUser(r gin.IRouter, api UserAPI, mid ...gin.HandlerFunc) {
	r.POST("/login", web.WrapH(api.login))
	r.GET("/login/key", web.WrapH(api.getPublicKey))

	group := r.Group("/users", mid...)
	group.PUT("", web.WrapHs(api.updateCredentials, mid...)...)
}

// 登录请求结构体
type loginInput struct {
	// Username string `json:"username" binding:"required"`
	// Password string `json:"password" binding:"required"`
	Data string `json:"data" binding:"required"`
}

// 登录响应结构体
type loginOutput struct {
	Token string `json:"token"`
	User  string `json:"user"`
}

// 登录接口
func (api UserAPI) login(_ *gin.Context, in *loginInput) (*loginOutput, error) {
	body, err := api.secret.Decrypt(in.Data)
	if err != nil {
		return nil, reason.ErrServer.SetMsg(err.Error())
	}
	var credentials struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(body, &credentials); err != nil {
		return nil, reason.ErrServer.SetMsg(err.Error())
	}

	// 验证用户名和密码
	if api.conf.Server.Username == "" && api.conf.Server.Password == "" {
		api.conf.Server.Username = "admin"
		api.conf.Server.Password = "admin"
	}
	if credentials.Username != api.conf.Server.Username || credentials.Password != api.conf.Server.Password {
		return nil, reason.ErrNameOrPasswd
	}

	data := web.NewClaimsData().SetUsername(credentials.Username)

	token, err := web.NewToken(data, api.conf.Server.HTTP.JwtSecret, web.WithExpiresAt(time.Now().Add(3*24*time.Hour)))
	if err != nil {
		return nil, reason.ErrServer.SetMsg("生成token失败: " + err.Error())
	}

	return &loginOutput{
		Token: token,
		User:  credentials.Username,
	}, nil
}

// 修改凭据请求结构体
type updateCredentialsInput struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// 修改凭据接口
func (api UserAPI) updateCredentials(_ *gin.Context, in *updateCredentialsInput) (gin.H, error) {
	// 更新配置中的用户名和密码
	api.conf.Server.Username = in.Username
	api.conf.Server.Password = in.Password

	// 写入配置文件
	if err := conf.WriteConfig(api.conf, api.conf.ConfigPath); err != nil {
		return nil, reason.ErrServer.SetMsg("保存配置失败: " + err.Error())
	}

	return gin.H{"msg": "凭据更新成功"}, nil
}

func (api UserAPI) getPublicKey(_ *gin.Context, _ *struct{}) (gin.H, error) {
	publicKey, err := api.secret.GetOrCreatePublicKey()
	if err != nil {
		return nil, reason.ErrServer.SetMsg(err.Error())
	}
	result := api.secret.MarshalPKIXPublicKey(publicKey)
	return gin.H{"key": base64.StdEncoding.EncodeToString(result)}, nil
}
