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
	"context"
	"io/ioutil"
	"k8s.io/klog/v2"
	"net/http"
	"strings"
	"time"
)

const (
	MaxRetry       = 5
	MaxIdlePerHost = 100
	RequestTimeout = 3 // second
)

var AuditAdapter *auditAdapter

type auditAdapter struct {
	URL        string
	Method     string
	Header     string
	HttpClient *http.Client
}

type Response struct {
	Code      int    `json:"code"`
	Data      string `json:"data"`
	ErrorCode string `json:"errorCode"`
}

func (adapter *auditAdapter) Publish(payload string, id string) {

	klog.Infof("[%v] audit message: %s", id, payload)

	// The initial wait interval, hard-coded to 80ms
	interval := 80 * time.Millisecond
	for i := 0; i < MaxRetry; i++ {
		if i != 0 {
			klog.Warningf("[%v] send audit message failed, retry after %v", id, interval)
			time.Sleep(interval)
			interval = time.Duration(int64(float32(interval) * 2.5))
		}
		err := adapter.sendWithRetry(payload, id)
		if err != nil {
			klog.Errorf("[%v] unexpected error when request audit svc, error: %v", id, err)
			continue
		}
		klog.Infof("[%v] send audit message to audit svc success.", id)
		return
	}

}

func (adapter *auditAdapter) sendWithRetry(payload string, id string) error {
	request, err := http.NewRequest(adapter.Method, adapter.URL, strings.NewReader(payload))
	if err != nil {
		klog.Error("[%v] create http request error: %v", id, err)
		return err
	}
	headers := strings.Split(adapter.Header, ";")
	for _, header := range headers {
		kv := strings.Split(header, "=")
		if len(kv) != 2 {
			continue
		}
		request.Header.Set(kv[0], kv[1])
	}

	resp, err := adapter.HttpClient.Do(request.WithContext(context.TODO()))
	if err != nil {
		klog.Errorf("[%v] send message to audit service error: %s", id, err.Error())
		return err
	}
	defer resp.Body.Close()

	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		klog.Errorf("[%v] read response body error: %s", id, err.Error())
		return err
	}

	klog.Warningf("[%v] response from audit service, statusCode=%d, body=%s", id, resp.StatusCode, content)

	//result := &Response{}
	//err = json.Unmarshal(content, result)
	//if err != nil {
	//	klog.Errorf("[%v] unmarshal response json error: %s", id, err.Error())
	//	return err
	//}
	//
	//if resp.StatusCode != http.StatusOK || result.Code != http.StatusOK {
	//	return errors.New("bad response status code")
	//}
	return nil
}
