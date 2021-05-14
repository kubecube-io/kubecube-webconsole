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
	"sync"
)

const (
	LeaderElectionKey       = "kubecube-webconsole-leader-election-key"
	LeaderElectionNamespace = "kube-system"
	NamespaceKey            = "namespace"
	KubeCubeChrootShPath    = "/kubecube-chroot.sh"
	CloudShellLabelKey      = "system/app"
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
	ClusterId     string `json:"clusterId"`
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
	ServerPort        = flag.Int("serverPort", 9081, "set server port")
	scriptName        = flag.String("scriptName", "/init.sh", "script name with full path in container")
	cloudShellDpName  = flag.String("cloudShellDpName", "kubecube-cloud-shell", "deployment run on control cluster for cloud shell, for example,'kubecube-cloud-shell'")
	appNamespace      = flag.String("appNamespace", "cloud-shell", "namespace of cloud shell deployment, default same as cloud-shell")
	enableAudit       = flag.Bool("enableAudit", false, "enable audit function")
	enableStdoutAudit = flag.Bool("enableStdoutAudit", false, "enable stdout audit")
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
