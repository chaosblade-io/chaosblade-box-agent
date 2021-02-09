/*
 * Copyright 1999-2021 Alibaba Group Holding Ltd.
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
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"strconv"
)

//RequestInvoker invoke remote service and return response
type RequestInvoker interface {
	Invoke(uri Uri, request *Request) (*Response, error)
}

// invoker with interceptor
type requestInvoker struct {
	client      *httpClient
	interceptor RequestInterceptor
	RequestInvoker
}

func (invoker *requestInvoker) Invoke(uri Uri, request *Request) (*Response, error) {
	// interceptor
	interceptor := invoker.interceptor
	if interceptor != nil {
		if response, ok := interceptor.Invoke(request); !ok {
			return response, errors.New(response.Error)
		}
	}

	reqBody, _ := json.Marshal(request.GetBody())
	result, err := invoker.client.HttpCall(uri.HandlerName, reqBody)

	if err != nil {
		log.Warningf("Invoke failed. error:%s", err.Error())
		return nil, err
	}
	// decode
	var response Response
	err = json.Unmarshal([]byte(result), &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}

func NewInvoker(client *httpClient, needInterceptor bool) RequestInvoker {
	//  Not need request interceptor when first connect,
	var interceptor RequestInterceptor
	if needInterceptor {
		interceptor = buildInterceptor()
	} else {
		interceptor = nil
	}
	// entry invoker
	var invoker = &requestInvoker{
		client:      client,
		interceptor: interceptor,
	}
	invoker.RequestInvoker = invoker
	return invoker
}

func buildInterceptor() RequestInterceptor {

	// timestamp
	timestampInterceptor := &timestampInterceptor{}
	timeChain := requestInterceptorChain{}
	timeChain.chain = nil
	timeChain.RequestInterceptor = &timeChain
	timeChain.doRequestInterceptor = timestampInterceptor
	timestampInterceptor.requestInterceptorChain = timeChain

	return timestampInterceptor
}

type timestampInterceptor struct {
	requestInterceptorChain
}

func (interceptor *timestampInterceptor) doHandler(request *Request) (*Response, bool) {
	// check timestamp
	requestTime := request.Params[TimestampKey]
	if requestTime == "" {
		return ReturnFail(Code[InvalidTimestamp], Code[InvalidTimestamp].Msg), false
	}
	_, err := strconv.ParseInt(requestTime, 10, 64)
	if err != nil {
		return ReturnFail(Code[InvalidTimestamp], err.Error()), false
	}
	//if getCurrentTimeInMillis()-t > int64(MaxInvalidTime) {
	//	return ReturnFail(Code[Timeout], Code[Timeout].Msg), false
	//}
	return nil, true
}

func (interceptor *timestampInterceptor) doInvoker(request *Request) (*Response, bool) {
	// add timestamp
	currTime := getCurrentTimeInMillis()
	request.AddParam(TimestampKey, strconv.FormatInt(currTime, 10))
	return nil, true
}
