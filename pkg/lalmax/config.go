package lalmax

type GetServerConfigResponse struct {
	ConfVersion                 string `json:"conf_version"`
	CheckSessionDisposeInterval uint32 `json:"check_session_dispose_interval"`
	UpdateSessionStateInterval  uint32 `json:"update_session_state_interval"`
	ManagerChanSize             uint32 `json:"manager_chan_size"`
	AdjustPts                   bool   `json:"adjust_pts"`
	MaxOpenFiles                uint64 `json:"max_open_files"`
	// 关键帧存储路径
	KeyFramePath      string            `json:"key_frame_path"`
	GopCacheConfig    GopCacheConfig    `json:"gop_cache_config"`
	RtmpConfig        RtmpConfig        `json:"rtmp"`
	InSessionConfig   InSessionConfig   `json:"in_session"`
	DefaultHttpConfig DefaultHttpConfig `json:"default_http"`
	HttpflvConfig     HttpflvConfig     `json:"httpflv"`
	HlsConfig         HlsConfig         `json:"hls"`
	HttptsConfig      HttptsConfig      `json:"httpts"`
	RtspConfig        RtspConfig        `json:"rtsp"`
	DashConfig        DashConfig        `json:"dash"`
	RtcConfig         RtcConfig         `json:"rtc"`
	Gb28181Config     SipConfig         `json:"gb28181"`
	// OnvifConfig           onvif.Config          `json:"onvif"`
	HttpFmp4Config        HttpFmp4Config        `json:"httpfmp4"`
	RoomConfig            RoomConfig            `json:"room"`
	RecordConfig          RecordConfig          `json:"record"`
	PlaybackConfig        PlaybackConfig        `json:"playback"`
	MetricsConfig         MetricsConfig         `json:"metrics"`
	RelayPushConfig       RelayPushConfig       `json:"relay_push"`
	StaticRelayPullConfig StaticRelayPullConfig `json:"static_relay_pull"`

	HttpApiConfig    HttpApiConfig    `json:"http_api"`
	ServerId         string           `json:"server_id"`
	HttpNotifyConfig HttpNotifyConfig `json:"http_notify"`
	SimpleAuthConfig SimpleAuthConfig `json:"simple_auth"`
	PprofConfig      PprofConfig      `json:"pprof"`
	// LogConfig        nazalog.Option   `json:"log"`
	DebugConfig DebugConfig `json:"debug"`

	ReportStatWithFrameRecord bool `json:"report_stat_with_frame_record"`
}

type SipConfig struct {
	Enable            bool   `json:"enable"`             // gb28181使能标志
	ListenAddr        string `json:"listen_addr"`        // gb28181监听地址
	SipIP             string `json:"sip_ip"`             // sip 服务器公网IP
	SipPort           uint16 `json:"sip_port"`           // sip 服务器端口，默认 5060
	Serial            string `json:"serial"`             // sip 服务器 id, 默认 34020000002000000001
	Realm             string `json:"realm"`              // sip 服务器域，默认 3402000000
	Username          string `json:"username"`           // sip 服务器账号
	Password          string `json:"password"`           // sip 服务器密码
	KeepaliveInterval int    `json:"keepalive_interval"` // 心跳包时长
	QuickLogin        bool   `json:"quick_login"`        // 快速登陆,有keepalive就认为在线
	SipLogClose       bool   `json:"sip_log_close"`      // 关闭sip日志
	// MediaConfig       MediaConfig `json:"media_config"`       // 媒体服务器配置
}

type GopCacheConfig struct {
	GopNum               int `json:"gop_cache_num"`
	SingleGopMaxFrameNum int `json:"single_gop_max_frame_num"`
}

type RtmpConfig struct {
	Enable                  bool   `json:"enable"`
	Addr                    string `json:"addr"`
	RtmpsEnable             bool   `json:"rtmps_enable"`
	RtmpsAddr               string `json:"rtmps_addr"`
	RtmpOverQuicEnable      bool   `json:"rtmp_over_quic_enable"`
	RtmpOverQuicAddr        string `json:"rtmp_over_quic_addr"`
	RtmpsCertFile           string `json:"rtmps_cert_file"`
	RtmpsKeyFile            string `json:"rtmps_key_file"`
	RtmpOverKcpEnable       bool   `json:"rtmp_over_kcp_enable"`
	RtmpOverKcpAddr         string `json:"rtmp_over_kcp_addr"`
	RtmpOverKcpDataShards   int    `json:"rtmp_over_kcp_data_shards"`
	RtmpOverKcpParityShards int    `json:"rtmp_over_kcp_parity_shards"`

	MergeWriteSize int    `json:"merge_write_size"`
	PubTimeoutSec  uint32 `json:"pub_timeout_sec"`
	PullTimeoutSec uint32 `json:"pull_timeout_sec"`
}

type InSessionConfig struct {
	AddDummyAudioEnable      bool `json:"add_dummy_audio_enable"`
	AddDummyAudioWaitAudioMs int  `json:"add_dummy_audio_wait_audio_ms"`
}

type DefaultHttpConfig struct {
	CommonHttpAddrConfig
}

type HttpflvConfig struct {
	CommonHttpServerConfig
}

type HttptsConfig struct {
	CommonHttpServerConfig
}

type HlsConfig struct {
	CommonHttpServerConfig

	UseMemoryAsDiskFlag bool `json:"use_memory_as_disk_flag"`
	DiskUseMmapFlag     bool `json:"disk_use_mmap_flag"`
	UseM3u8MemoryFlag   bool `json:"use_m3u8_memory_flag"`
	// hls.MuxerConfig
	SubSessionTimeoutMs  int                    `json:"sub_session_timeout_ms"`
	SubSessionHashKey    string                 `json:"sub_session_hash_key"`
	Fmp4HttpServerConfig CommonHttpServerConfig `json:"fmp4"`
}

type DashConfig struct {
	CommonHttpServerConfig
	UseMemoryAsDiskFlag bool `json:"use_memory_as_disk_flag"`
	DiskUseMmapFlag     bool `json:"disk_use_mmap_flag"`
	UseMpdMemoryFlag    bool `json:"use_mpd_memory_flag"`
	// dash.Config
}

type RtcConfig struct {
	PubTimeoutSec uint32 `json:"pub_timeout_sec"`
	CommonHttpServerConfig
	// rtc.ICEConfig
}

type RtspConfig struct {
	Enable                            bool   `json:"enable"`
	Addr                              string `json:"addr"`
	RtspsEnable                       bool   `json:"rtsps_enable"`
	RtspsAddr                         string `json:"rtsps_addr"`
	RtspsCertFile                     string `json:"rtsps_cert_file"`
	RtspsKeyFile                      string `json:"rtsps_key_file"`
	OutWaitKeyFrameFlag               bool   `json:"out_wait_key_frame_flag"`
	WsRtspEnable                      bool   `json:"ws_rtsp_enable"`
	WsRtspAddr                        string `json:"ws_rtsp_addr"`
	PubTimeoutSec                     uint32 `json:"pub_timeout_sec"`
	PullTimeoutSec                    uint32 `json:"pull_timeout_sec"`
	RtspRemuxerAddSpsPps2KeyFrameFlag bool   `json:"add_sps_pps_to_key_frame_flag"`
	// rtsp.ServerAuthConfig
}

type Jt1078Config struct {
	Enable        bool   `json:"enable"`
	ListenIp      string `json:"listen_ip"`
	ListenPort    int    `json:"listen_port"`
	PortNum       uint16 `json:"port_num"` // 范围 ListenPort至ListenPort+PortNum
	PubTimeoutSec uint32 `json:"pub_timeout_sec"`
	Intercom      struct {
		Enable     bool
		IP         string `json:"ip"`           // 固定外网ip
		Port       int    `json:"port"`         // 固定外网udp端口
		AudioPorts [2]int `json:"audio_ports"`  // 范围 AudioPorts[0]至AudioPorts[1]
		OnJoinURL  string `json:"on_join_url"`  // 设备对讲连接上了的url回调
		OnLeaveURL string `json:"on_leave_url"` // 设备对讲断开了的url回调
	} `json:"intercom"`
}

type HttpFmp4Config struct {
	CommonHttpServerConfig
}

type RecordConfig struct {
	EnableFlv            bool   `json:"enable_flv"`
	FlvOutPath           string `json:"flv_out_path"`
	EnableMpegts         bool   `json:"enable_mpegts"`
	MpegtsOutPath        string `json:"mpegts_out_path"`
	EnableFmp4           bool   `json:"enable_fmp4"`
	Fmp4OutPath          string `json:"fmp4_out_path"`
	RecordInterval       int    `json:"record_interval"`        // 固定间隔录制一个文件，单位秒
	EnableRecordInterval bool   `json:"enable_record_interval"` // 是否开启固定间隔录制
}

type PlaybackConfig struct {
	CommonHttpServerConfig
}

type MetricsConfig struct {
	Enable         bool   `json:"enable"`
	PushgatewayURL string `json:"pushgateway_url"`
	JobName        string `json:"job_name"`
	InstanceName   string `json:"instance_name"`
	PushInterval   int    `json:"push_interval"`
}

type RelayPushConfig struct {
	Enable   bool     `json:"enable"`
	AddrList []string `json:"addr_list"`
}

type StaticRelayPullConfig struct {
	Enable bool   `json:"enable"`
	Addr   string `json:"addr"`
}

type HttpApiConfig struct {
	CommonHttpServerConfig
}

type HttpNotifyConfig struct {
	Enable            bool   `json:"enable"`
	UpdateIntervalSec int    `json:"update_interval_sec"`
	OnServerStart     string `json:"on_server_start"`
	OnUpdate          string `json:"on_update"`
	OnPubStart        string `json:"on_pub_start"`
	OnPubStop         string `json:"on_pub_stop"`
	OnSubStart        string `json:"on_sub_start"`
	OnSubStop         string `json:"on_sub_stop"`
	OnPushStart       string `json:"on_push_start"`
	OnPushStop        string `json:"on_push_stop"`
	OnRelayPullStart  string `json:"on_relay_pull_start"`
	OnRelayPullStop   string `json:"on_relay_pull_stop"`
	OnRtmpConnect     string `json:"on_rtmp_connect"`
	OnHlsMakeTs       string `json:"on_hls_make_ts"`
	OnHlsMakeFmp4     string `json:"on_hls_make_fmp4"`
	OnReportStat      string `json:"on_report_stat"`
	OnReportFrameInfo string `json:"on_report_frame_info"`
	MaxTaskLen        int    `json:"max_task_len"` // 最大任务数
	ClientSize        int    `json:"client_size"`  // 并发客户端
	// NotifyTimeoutSec  int    `json:"notify_timeout_sec"` // 通知超时时间
	DiscardInterval uint32 `json:"discard_interval"` // 丢弃间隔，当队列满的时候，丢弃数量达到此值，下一个一定保留
}

type SimpleAuthConfig struct {
	Key                string `json:"key"`
	DangerousLalSecret string `json:"dangerous_lal_secret"`
	PubRtmpEnable      bool   `json:"pub_rtmp_enable"`
	SubRtmpEnable      bool   `json:"sub_rtmp_enable"`
	SubHttpflvEnable   bool   `json:"sub_httpflv_enable"`
	SubHttptsEnable    bool   `json:"sub_httpts_enable"`
	PubRtspEnable      bool   `json:"pub_rtsp_enable"`
	SubRtspEnable      bool   `json:"sub_rtsp_enable"`
	HlsM3u8Enable      bool   `json:"hls_m3u8_enable"`
	PushRtmpEnable     bool   `json:"push_rtmp_enable"`
	PushJt1078Enable   bool   `json:"push_jt1078_enable"`
	PushPsEnable       bool   `json:"push_ps_enable"`
}

type PprofConfig struct {
	CommonHttpServerConfig
}

type DebugConfig struct {
	LogGroupIntervalSec       int `json:"log_group_interval_sec"`
	LogGroupMaxGroupNum       int `json:"log_group_max_group_num"`
	LogGroupMaxSubNumPerGroup int `json:"log_group_max_sub_num_per_group"`
}

type CommonHttpServerConfig struct {
	CommonHttpAddrConfig

	Enable      bool   `json:"enable"`
	EnableHttps bool   `json:"enable_https"`
	EnableHttp3 bool   `json:"enable_http3"`
	UrlPattern  string `json:"url_pattern"`
}

type CommonHttpAddrConfig struct {
	HttpListenAddr  string `json:"http_listen_addr"`
	HttpsListenAddr string `json:"https_listen_addr"`
	Http3ListenAddr string `json:"http3_listen_addr"`
	HttpsCertFile   string `json:"https_cert_file"`
	HttpsKeyFile    string `json:"https_key_file"`
}

type RoomConfig struct {
	Enable    bool   `json:"enable"`
	APIKey    string `json:"api_key"`
	APISecret string `json:"api_secret"`
}

func (e *Engine) GetServerConfig() (*GetServerConfigResponse, error) {
	// var resp GetServerConfigResponse
	// if err := e.post(getServerConfig, nil, &resp); err != nil {
	// 	return nil, err
	// }
	// if err := e.ErrHandle(resp.Code, resp.Msg); err != nil {
	// 	return nil, err
	// }
	// return &resp, nil

	return nil, nil
}
