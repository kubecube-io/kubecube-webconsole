package handler

import (
	"bytes"
	"flag"
	"io"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"k8s.io/client-go/tools/remotecommand"
)

const (
	//todo
	NamespaceKey = "namespace"
	AccountIdKey = "x-auth-accountId"
	JwtTokenKey  = "x-auth-token"

	// pod is the object skiff-webconsole connect to
	ResourceTypePod = "pod"
	// connect to pod treated as the 'list' operation defined in skiff
	OperationTypeList  = "List"
	ServiceModeTypeNCS = "ncs"

	ClusterModeSkiff  = "skiff"
	ClusterModeNative = "native"

	PlatformAuthUrl    = "/authority?Action=BatchAuthentication&Version=2018-08-09"
	PlatformPrjInfoUrl = "/authority?Action=DescribePermissionScope&Version=2018-08-09&PermissionScopeId="

	// 集群的操作是管理员级别的，因此采取该资源+操作实现鉴权
	ClusterRes = "cluster"
	ClusterAdd = "Add"

	SystemScopeId  = "1"
	ParentOfSystem = "0"

	SkiffChrootShPath         = "/skiff-chroot.sh"
	NCS_CLOUD_SHELL_LABEL_KEY = "nce-app"

	PlatformSkiff = "skiff"
)

const (
	// golang中即使是定义常量也使用驼峰
	OsLinux = "linux"
	OsWin   = "windows"
	OsMac   = "darwin"

	ResourceContainer = "container"
	IoStdin           = "stdin"
	IoStdout          = "stdout"
	IoStderr          = "stderr"
	TTY               = "tty"

	// namespace类型独占（只能关联一个项目）或共享（可关联多个项目）
	NamespaceTypeKey           = "system/type"
	NamespaceTypeSingleProject = "singleproject"
)

// TerminalResponse is sent by handleExecShell. The Id is a random session id that binds the original REST request and the SockJS connection.
// Any clientapi in possession of this Id can hijack the terminal session.
type TerminalResponse struct {
	Id      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

// ConnInfo stores container-connect related information
type ConnInfo struct {
	TenantId      string `json:"tenantId"`
	ClusterName   string `json:"clusterName"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	Namespace     string `json:"namespace"`
	//ScriptName     string `json:"scriptName"`     // 容器中的执行脚本的路径+脚本名
	ScriptUser     string `json:"scriptUser"`     // 容器中脚本使用的用户名
	ScriptUID      string `json:"scriptUID"`      // 容器中脚本使用的用户uid
	ScriptUserAuth string `json:"scriptUserAuth"` // 容器中脚本使用的用户权限等级，由用户自定义，如dev、ops、admin
	// 额外扩展的两个字段，为了在管控集群上运行kubectl,需要账户信息，操作的计算集群id
	AccountId        string        `json:"accountId"`
	ClusterId        string        `json:"clusterId"`
	IsControlCluster bool          `json:"isControlCluster"`
	AuditRawInfo     *AuditRawInfo `json:"audit_raw_info,omitempty"`
}

type AuditRawInfo struct {
	RemoteIP  string `json:"remote_ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	WebUser   string `json:"web_user,omitempty"`
	Platform  string `json:"platform,omitempty"` // 通过什么平台传入的，如严选SNet\严选Opera或者轻舟页面
}

var (
	configMap      *cache.Cache // 存放kubeconfig信息
	connMap        sync.Map     // 存放连接到容器所需要的信息，集群名、namespace、pod名、容器名、租户id、容器中的user信息等
	clusterModeMap *cache.Cache // 使用cache存放集群类型信息（原生/轻舟），并配置上超时时间

	// 管控集群上运行的后端dp名字
	CloudShellDpName string
	CloudShellNs     string
	// 管控集群的名字，初始随机生成一个，获取不到则通过数据库获取，再更新此字段, 可能存在同时修改的情况，不考虑加锁，没必要
	ControlClusterName = "###random-init##!!"
)

var (
	authEndpoint      = flag.String("authEndpoint", "http://platform-user-auth.qa-ci.service.163.org", "auth platform endpoint")
	sentinelAddrs     = flag.String("sentinelAddrs", "10.173.32.51:26379,10.173.32.51:26380", "redis server endpoint format ip:port")
	redisPassword     = flag.String("redisPassword", "123", "redis password")
	redisMasterName   = flag.String("redisMasterName", "mymaster", "redis master name")
	redisDbIdx        = flag.Int("redisDbIdx", 0, "redis db index")
	ServerPort        = flag.Int("serverPort", 9081, "set server port")
	host              = flag.String("dbHost", "localhost:20002", "database host with format xx.xx.xx.xx:xxxx")
	dbName            = flag.String("dbName", "ncs", "database name")
	username          = flag.String("userName", "qzmysql", "database username")
	password          = flag.String("password", "uAXy8kHyEcqRJYNx", "database password")
	maxOpenConns      = flag.Int("maxOpenConns", 200, "max open connections")
	maxIdleConns      = flag.Int("maxIdleConns", 64, "max idle connections")
	scriptName        = flag.String("scriptName", "/init.sh", "script name with full path in container")
	cloudShellDpName  = flag.String("cloudShellDpName", "skiff-cloud-shell", "deployment run on control cluster for cloud shell, for example,'skiff-cloud-shell'")
	appNamespace      = flag.String("appNamespace", "dev", "namespace of skiff cloud shell deployment, default same as skiff-ncs")
	enableAudit       = flag.Bool("enableAudit", false, "enable audit function")
	enableStdoutAudit = flag.Bool("enableStdoutAudit", false, "enable stdout audit")
	auditConfigPath   = flag.String("auditConfigPath", "testconf/auditconf.yaml", "audit config file path")
)

// TerminalSession implements PtyHandler (using a SockJS connection)
type TerminalSession struct {
	id string
	//bound         chan error
	sockJSSession sockjs.Session
	sizeChan      chan remotecommand.TerminalSize
	stdinBuffer   *bytes.Buffer
	cInfo         *ConnInfo
}

// TerminalMessage is the messaging protocol between ShellController and TerminalSession.
//
// OP      DIRECTION  FIELD(S) USED  DESCRIPTION
// ---------------------------------------------------------------------
// bind    fe->be     SessionID      Id sent back from TerminalResponse
// stdin   fe->be     Data           Keystrokes/paste buffer
// resize  fe->be     Rows, Cols     New terminal size
// stdout  be->fe     Data           Output from the process
// toast   be->fe     Data           OOB message to be shown to the user
type TerminalMessage struct {
	Op, Data, SessionID string
	Rows, Cols          uint16
}

// PtyHandler is what remotecommand expects from a pty
type PtyHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

type AuthInfo struct {
	Jwt               string `json:"JWT"`
	Operation         string `json:"OperationType"`
	ResourceType      string `json:"ResourceType"`
	Index             uint64 `json:"Index"`
	ParentId          string `json:"ParentId"`
	PermissionScopeId string `json:"PermissionScopeId"`
	AccountId         string `json:"AccountId"`
}

type AuthReq struct {
	AuthenticationParams []AuthInfo `json:"AuthenticationParams"`
	ServiceModule        string     `json:"ServiceModule"`
}

type AuthResult struct {
	Code      string `json:"Code"`
	HasRole   bool   `json:"HasRole"`
	Index     uint64 `json:"Index"`
	AccountId string `json:"AccountId"`
}

type AuthResp struct {
	AuthenticationResults []AuthResult `json:"AuthenticationResults"`
}

type ProjectInfo struct {
	Id                    uint64 `json:"Id"`
	ParentId              uint64 `json:"ParentId"`
	RequestId             string `json:"RequestId"`
	CreateTime            uint64 `json:"CreateTime"`
	UpdateTime            uint64 `json:"UpdateTime"`
	PermissionScopeType   string `json:"PermissionScopeType"`
	PermissionScopeName   string `json:"PermissionScopeName"`
	PermissionScopeEnName string `json:"PermissionScopeEnName"`
}

// AuditMsg stores info sent to audit server
type AuditMsg struct {
	SessionID     string    `json:"session_id"`
	CreateTime    time.Time `json:"create_time"`
	PodName       string    `json:"pod_name,omitempty"`
	Namespace     string    `json:"namespace,omitempty"`
	ClusterName   string    `json:"cluster_name,omitempty"`
	Data          string    `json:"data"`
	DataType      string    `json:"data_type"` //stdin, stdout
	RemoteIP      string    `json:"remote_ip,omitempty"`
	UserAgent     string    `json:"user_agent,omitempty"`
	ContainerUser string    `json:"container_user,omitempty"`
	WebUser       string    `json:"web_user,omitempty"`
	Platform      string    `json:"platform,omitempty"` // 通过什么平台传入的，如严选SNest\严选Opera或者轻舟页面
}
