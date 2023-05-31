/*
 * Copyright 1999-2020 Alibaba Group Holding Ltd.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package transport

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/chaosblade-io/chaos-agent/pkg/tools"
)

const (
	SignData       = "sd"
	AccessKey      = "ak"
	SignKey        = "sn"
	TimestampKey   = "ts"
	MaxInvalidTime = 60 * 1000 * time.Millisecond
)

type RequestInterceptor interface {
	Handle(request *Request) (*Response, bool)
	Invoke(request *Request) (*Response, bool)
}

type doRequestInterceptor interface {
	doHandler(request *Request) (*Response, bool)
	doInvoker(request *Request) (*Response, bool)
}

type requestInterceptorChain struct {
	chain RequestInterceptor
	RequestInterceptor
	doRequestInterceptor
}

// Handle interceptor. return nil,true if passed, otherwise return response of fail and false
func (interceptor *requestInterceptorChain) Handle(request *Request) (*Response, bool) {
	if response, ok := interceptor.doHandler(request); !ok {
		return response, ok
	}

	if interceptor != nil && interceptor.chain != nil {
		if response, ok := interceptor.chain.Handle(request); !ok {
			return response, ok
		}
	}
	return nil, true
}

// Invoke interceptor.
func (interceptor *requestInterceptorChain) Invoke(request *Request) (*Response, bool) {
	if response, ok := interceptor.doInvoker(request); !ok {
		return response, ok
	}
	if interceptor.chain != nil {
		if response, ok := interceptor.chain.Invoke(request); !ok {
			return response, ok
		}
	}
	return nil, true
}

type authInterceptor struct {
	requestInterceptorChain
}

func (authInterceptor *authInterceptor) doHandler(request *Request) (*Response, bool) {
	// check sign
	sign := request.Headers[SignKey]
	if sign == "" {
		return ReturnFail(Forbidden, "missing sign"), false
	}
	accessKey := request.Headers[AccessKey]
	if accessKey != "" && accessKey != tools.GetAccessKey() {
		return ReturnFail(Forbidden, "accessKey not matched"), false
	}
	signData := request.Headers[SignData]
	if signData == "" {
		bytes, err := json.Marshal(request.Params)
		if err != nil {
			return ReturnFail(Forbidden, "invalid request parameters"), false
		}
		signData = string(bytes)
	}
	if !tools.Auth(sign, signData) {
		return ReturnFail(Forbidden, "illegal request"), false
	}
	return nil, true
}

func (authInterceptor *authInterceptor) doInvoker(request *Request) (*Response, bool) {
	accessKey := tools.GetAccessKey()
	secureKey := tools.GetSecureKey()
	if accessKey == "" || secureKey == "" {
		return ReturnFail(TokenNotFound), false
	}
	request.AddHeader(AccessKey, accessKey)
	signData := request.Headers[SignData]
	if signData == "" {
		bytes, err := json.Marshal(request.Params)
		if err != nil {
			return ReturnFail(EncodeError, err.Error()), false
		}
		signData = string(bytes)
	}
	sign := tools.Sign(signData)
	request.AddHeader(SignKey, sign)
	return nil, true
}

type timestampInterceptor struct {
	requestInterceptorChain
}

func (interceptor *timestampInterceptor) doHandler(request *Request) (*Response, bool) {
	// check timestamp
	requestTime := request.Params[TimestampKey]
	if requestTime == "" {
		return ReturnFail(InvalidTimestamp), false
	}
	_, err := strconv.ParseInt(requestTime, 10, 64)
	if err != nil {
		return ReturnFail(InvalidTimestamp), false
	}

	return nil, true
}

func (interceptor *timestampInterceptor) doInvoker(request *Request) (*Response, bool) {
	// add timestamp
	currTime := getCurrentTimeInMillis()
	request.AddParam(TimestampKey, strconv.FormatInt(currTime, 10))
	return nil, true
}

func getCurrentTimeInMillis() int64 {
	return time.Now().UnixNano() / 1000
}
