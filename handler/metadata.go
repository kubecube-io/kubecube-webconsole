/*
Copyright 2021 KubeCube Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package handler

import (
	"bytes"
	"flag"
	"github.com/patrickmn/go-cache"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	"io"
	"k8s.io/client-go/tools/remotecommand"
	"net/http"
	"sync"
	"time"
)

const (
	LeaderElectionKey       = "kubecube-webconsole-leader-election-key"
	LeaderElectionNamespace = "kube-system"
	NamespaceKey            = "namespace"
	KubeCubeChrootShPath    = "/kubecube-chroot.sh"
	CloudShellLabelKey      = "kubecube.io/app"
)

const (
	ResourceContainer = "container"
	IoStdin           = "stdin"
	IoStdout          = "stdout"
	IoStderr          = "stderr"
	TTY               = "tty"
)

// TerminalResponse is sent by handleExecShell. The Id is a random session id that binds the original REST request and the SockJS connection.
// Any clientapi in possession of this Id can hijack the terminal session.
type TerminalResponse struct {
	Id      string `json:"id,omitempty"`
	Message string `json:"message,omitempty"`
}

// ConnInfo stores container-connect related information
type ConnInfo struct {
	ClusterName   string `json:"clusterName"`
	PodName       string `json:"podName"`
	ContainerName string `json:"containerName"`
	Namespace     string `json:"namespace"`
	ScriptUser    string `json:"scriptUser"` // the username used by the script in the container
	ScriptUID     string `json:"scriptUID"`  // the userid used by the script in the container
	// the user permission level used by the script in the container is customized by the user, such as dev, ops, admin
	ScriptUserAuth   string        `json:"scriptUserAuth"`
	UserName         string        `json:"userName"`
	IsControlCluster bool          `json:"isControlCluster"`
	AuditRawInfo     *AuditRawInfo `json:"audit_raw_info,omitempty"`
	Header           http.Header   `json:"header,omitempty"`
}

type AuditRawInfo struct {
	RemoteIP  string `json:"remote_ip,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	WebUser   string `json:"web_user,omitempty"`
	Platform  string `json:"platform,omitempty"`
}

var (
	configMap *cache.Cache // store kubeconfig information
	// store the information needed to connect to the container,
	// such as cluster name, namespace, pod name, container name, userinfo in the container, etc.
	connMap sync.Map

	CloudShellDpName string
	CloudShellNs     string
)

var (
	ServerPort             = flag.Int("serverPort", 9081, "set server port")
	scriptName             = flag.String("scriptName", "/init.sh", "script name with full path in container")
	cloudShellDpName       = flag.String("cloudShellDpName", "kubecube-cloud-shell", "deployment run on control cluster for cloud shell, for example,'kubecube-cloud-shell'")
	appNamespace           = flag.String("appNamespace", "kubecube-system", "namespace of cloud shell deployment, default same as kubecube-system")
	enableAudit            = flag.Bool("enableAudit", true, "enable audit function")
	enableStdoutAudit      = flag.Bool("enableStdoutAudit", false, "enable stdout audit")
	auditURL               = flag.String("auditURL", "http://audit.kubecube-system:8888/api/v1/cube/audit/cube", "send audit message to the url")
	auditMethod            = flag.String("auditMethod", "POST", "send audit message request method")
	auditHeader            = flag.String("auditHeader", "Content-Type=application/json;charset=UTF-8", "send audit message request header")
	authUrl                = flag.String("authUrl", "https://kubecube-ultimate.test203-qingzhou.com/api/v1/cube-ultimate/auth", "")
	authMethod             = flag.String("authMethod", "GET", "")
	authScheme             = flag.String("authScheme", "https", "")
	authInsecureSkipVerify = flag.Bool("authInsecureSkipVerify", true, "")
	authTLSCert            = flag.String("authTLSCert", "", "")
	authTLSKey             = flag.String("authTLSKey", "", "")
)

// TerminalSession implements PtyHandler (using a SockJS connection)
type TerminalSession struct {
	id            string
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
