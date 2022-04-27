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
	"encoding/json"
	"fmt"
	logger "github.com/astaxie/beego/logs"
	"k8s.io/klog/v2"
	"time"

	"github.com/kubecube-io/kubecube/pkg/clog"
	"gopkg.in/igm/sockjs-go.v2/sockjs"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"strings"
)

// TerminalSize handles pty->process resize events
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Next() *remotecommand.TerminalSize {
	size := <-t.sizeChan
	return &size
}

// Read handles pty->process messages (stdin, resize)
// Called in a loop from remotecommand as long as the process is running
func (t TerminalSession) Read(p []byte) (int, error) {
	m, err := t.sockJSSession.Recv()
	if err != nil {
		return 0, err
	}

	if m == "ping" {
		_ = t.sockJSSession.Send("pong")
		return 0, nil
	}

	var msg TerminalMessage
	if err := json.Unmarshal([]byte(m), &msg); err != nil {
		return 0, err
	}

	switch msg.Op {
	case "stdin":
		logger.Debug("[%v] stdin msg.Data content bytes: %v", t.id, []byte(msg.Data))
		if !*enableAudit {
			return copy(p, msg.Data), nil
		}

		// Audit function
		// user enters the enter key
		if strings.HasSuffix(msg.Data, "\r") {
			// if no command in buffer, no need to send
			if t.stdinBuffer.String() != "" {
				t.stdinBuffer.WriteString(strings.TrimSuffix(msg.Data, "\r"))
				go func(cmd string) {
					auditMsg := t.buildAuditMsg(cmd, "stdin")
					payload, err := json.Marshal(auditMsg)
					if err != nil {
						klog.Errorf("marshal stdin audit message failed, %v", err)
						return
					}
					AuditAdapter.Publish(string(payload), t.id)
				}(t.stdinBuffer.String())
			}

			// clean buffer
			t.stdinBuffer.Reset()
		} else {
			t.stdinBuffer.WriteString(msg.Data)
		}
		return copy(p, msg.Data), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{Width: msg.Cols, Height: msg.Rows}
		return 0, nil
	default:
		return 0, fmt.Errorf("unknown message type '%s'", msg.Op)
	}
}

// Write handles process->pty stdout
// Called from remotecommand whenever there is any output
func (t TerminalSession) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "OCI runtime exec failed") && strings.Contains(string(p), "exec: \\\"/bin/bash\\\"") {
		return 0, nil
	}
	msg, err := json.Marshal(TerminalMessage{
		Op:   "stdout",
		Data: string(p),
	})
	if err != nil {
		return 0, err
	}

	if err = t.sockJSSession.Send(string(msg)); err != nil {
		return 0, err
	}

	// auditing is not enabled, or stdout auditing is not required, return directly
	if !*enableAudit || !*enableStdoutAudit {
		return len(p), nil
	}

	go func(data string) {
		auditMsg := t.buildAuditMsg(data, "stdout")
		payload, err := json.Marshal(auditMsg)
		if err != nil {
			klog.Errorf("marshal stdout audit message failed, %v", err)
			return
		}
		AuditAdapter.Publish(string(payload), t.id)
	}(string(p))

	return len(p), nil
}

// Close shuts down the SockJS connection and sends the status code and reason to the client
// Can happen if the process exits or if there is an error starting up the process
// For now the status code is unused and reason is shown to the user (unless "")
func (t TerminalSession) Close(status uint32, reason string) {
	t.sockJSSession.Close(status, reason)
}

// handleTerminalSession is Called by net/http for any new /api/sockjs connections
func handleTerminalSession(session sockjs.Session) {
	var (
		buf             string
		err             error
		msg             TerminalMessage
		terminalSession TerminalSession
	)

	if buf, err = session.Recv(); err != nil {
		logger.Error("handleTerminalSession: can't Recv: %v", err)
		return
	}

	if err = json.Unmarshal([]byte(buf), &msg); err != nil {
		logger.Error("handleTerminalSession: can't UnMarshal (%v): %s", err, buf)
		return
	}

	if msg.Op != "bind" {
		logger.Error("handleTerminalSession: expected 'bind' message, got: %s", buf)
		return
	}

	restClient, cfg, info, err := getConfigs(msg.SessionID)
	if err != nil {
		logger.Error("get rest client failed. Error msg: " + err.Error())
		return
	}

	terminalSession = TerminalSession{
		id:            msg.SessionID,
		sockJSSession: session,
		sizeChan:      make(chan remotecommand.TerminalSize),
		stdinBuffer:   bytes.NewBufferString(""),
		cInfo:         info,
	}

	logger.Info("connect to container with cluster: %s, namespace: %s, pod name: %s, container name: %s, session id: %s", info.ClusterName, info.Namespace, info.PodName, info.ContainerName, msg.SessionID)
	if err = connectToContainer(restClient, cfg, info, terminalSession); err != nil {
		logger.Error("connect to container failed, session id: %v , error message: %v", msg.SessionID, err.Error())
		terminalSession.Close(2, err.Error())
		return
	}
	terminalSession.Close(1, "process exited")
}

func getConfigs(sessionID string) (*rest.RESTClient, *rest.Config, *ConnInfo, error) {
	var val string
	var err error
	var info *ConnInfo

	v, ok := connMap.Load(sessionID)
	if ok {
		val = v.(string)
	}

	err = json.Unmarshal([]byte(val), &info)
	if err != nil {
		logger.Error("unmarshal container-connect info from failed, key: %v, value: %v, error: %v", sessionID, val, err)
		return nil, nil, nil, err
	}

	v, err = getNonControlCfg(info.ClusterName)

	if err != nil {
		logger.Error("failed to fetch rest.config for cluster [%s], msg: %v", info.ClusterName, err)
		return nil, nil, nil, err
	}

	cfg, _ := (v).(*rest.Config)

	restClient, err := rest.RESTClientFor(cfg)
	if err != nil {
		logger.Error("get rest client failed. Error msg: " + err.Error())
		return nil, nil, nil, err
	}
	return restClient, cfg, info, nil
}

func connectToContainer(k8sClient *rest.RESTClient, cfg *rest.Config, info *ConnInfo, ptyHandler PtyHandler) error {
	namespace := info.Namespace
	podName := info.PodName
	containerName := info.ContainerName

	// connect to control service
	var req *rest.Request
	if info.IsControlCluster {
		cmds := []string{KubeCubeChrootShPath, "-u", info.UserName, "-c", info.ClusterName, "-t", info.Token}

		req = k8sClient.Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("exec").
			Param(ResourceContainer, containerName)

		req = req.VersionedParams(&v1.PodExecOptions{
			Command:   cmds,
			Container: containerName,
			Stdin:     true,
			Stdout:    true,
			Stderr:    true,
			TTY:       true,
		}, scheme.ParameterCodec)

		err := postReq(req, cfg, ptyHandler)
		if err != nil {
			clog.Error("run shell or connect to container error: %s", err)
			return err
		}
		return nil
	}

	cmds := buildCMD(info)
	req = k8sClient.Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param(ResourceContainer, containerName).
		Param(IoStdin, "true").
		Param(IoStdout, "true").
		Param(IoStderr, "true").
		Param(TTY, "true")
	req = req.VersionedParams(&v1.PodExecOptions{
		Command: cmds,
	}, scheme.ParameterCodec)

	// try to run `/bin/bash` after into container
	err := postReq(req, cfg, ptyHandler)
	// if err, run `/bin/sh`
	if err != nil {
		cmds := []string{"/bin/sh"}
		req = k8sClient.Post().
			Resource("pods").
			Name(podName).
			Namespace(namespace).
			SubResource("exec").
			Param(ResourceContainer, containerName).
			Param(IoStdin, "true").
			Param(IoStdout, "true").
			Param(IoStderr, "true").
			Param(TTY, "true")
		req = req.VersionedParams(&v1.PodExecOptions{
			Command: cmds,
		}, scheme.ParameterCodec)
		logger.Info("try to connect to container with cmds: %v", cmds)
		shErr := postReq(req, cfg, ptyHandler)
		if shErr != nil {
			logger.Error("connect to pod %v failed, %v", podName, err)
			return shErr
		}
	}
	return nil
}

func postReq(req *rest.Request, cfg *rest.Config, ptyHandler PtyHandler) error {
	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", req.URL())
	if err != nil {
		logger.Error("new SPDY executor failed, %v", err)
		return err
	}

	// Stream will block the current goroutine
	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:             ptyHandler,
		Stdout:            ptyHandler,
		Stderr:            ptyHandler,
		TerminalSizeQueue: ptyHandler,
		Tty:               true,
	})
	return err
}

func buildCMD(info *ConnInfo) []string {
	userFlag := false
	cmds := []string{*scriptName}
	if info.ScriptUser != "" {
		userFlag = true
		cmds = append(cmds, "-u", info.ScriptUser)
	}
	if info.ScriptUID != "" {
		userFlag = true
		cmds = append(cmds, "-i", info.ScriptUID)
	}
	if info.ScriptUserAuth != "" {
		userFlag = true
		cmds = append(cmds, "-a", info.ScriptUserAuth)
	}

	// if front end does not specify the user in the container, run `/bin/bash` directly in the container
	if !userFlag {
		cmds = []string{"/bin/bash"}
	}

	logger.Info("try to connect to container with cmds: %v", cmds)

	return cmds
}

func (t TerminalSession) buildAuditMsg(cmd string, dataType string) *AuditMsg {
	msg := &AuditMsg{
		SessionID:     t.id,
		Data:          cmd,
		DataType:      dataType,
		CreateTime:    time.Now(),
		PodName:       t.cInfo.PodName,
		Namespace:     t.cInfo.Namespace,
		ClusterName:   t.cInfo.ClusterName,
		ContainerUser: t.cInfo.ScriptUser,
	}
	auditRawInfo := t.cInfo.AuditRawInfo
	if auditRawInfo != nil {
		msg.RemoteIP = auditRawInfo.RemoteIP
		msg.UserAgent = auditRawInfo.UserAgent
		msg.WebUser = auditRawInfo.WebUser
		msg.Platform = auditRawInfo.Platform
	}
	return msg
}
