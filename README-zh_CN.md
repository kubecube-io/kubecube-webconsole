# kubecube-webconsole

### 总体架构

整个webconsole分为两个主要功能点:

- 前端与服务端的实时交互
- 服务端与计算集群的实时交互

其中前端与服务端采用websocket协议进行通信，服务端与Kubernetes集群中的容器使用SPDY协议进行通信。

整体的架构图如下：

![架构图](./images/webconsole架构图.png)

### webconsole连接流程

1. 前端发送一个http的请求至服务端获取websocket连接校验用的sessionID
2. 通过获取的sessionID创建一个websocket连接（连接的方式：前端通过sockjs）之后实现socket交互，获取客户端发送的数据（就是在恰客户端终端输入的命令）
3. 通过client-go发送对应的数据到指定容器，并获取返回数据，回传给客户端终端
4. 客户端终端通过xterm.js/hterm.js做相关的显示。

### 主要接口定义

页面跟webconsole交互过程中，会有两个API接口调用：

- 传入容器负载信息获取sessionId。该连接为普通http连接：
   `/api/v1/{cluster}/namespace/{namespace}/pod/{pod}/shell/{container} ` 返回样例：
   `{  "id": "3a9ae585ceaa6e3b0c72c31b0c215187" }`
- 通过sessionId建立websocket连接。 webconsole服务提供sockjs接口，前端调用sockjs接口时为普通http调用，该接口成功返回后会由sockjs自动建立websocket连接。因此该API需要链路上所有节点均支持websocket。
   后端支持的API： `/api/sockjs/info?3a9ae585ceaa6e3b0c72c31b0c215187&t=1548837417633`
   该API有2个URL参数，第1个是前一步骤获取到的sessionId（不带key），第二个是时间戳（key为t）。
   返回样例：
   `{"websocket":true,"cookie_needed":false,"origins":["*:*"],"entropy":1983920037}`
   其中websocket的值为`true`，表明webconsole中使用的sockjs使能了websocket功能，因此前端sockJs会建立起跟webconsole的websocket连接。

## 讨论与反馈

[FAQ](https://www.kubecube.io/docs/faq/)

## 开源协议

```
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
```