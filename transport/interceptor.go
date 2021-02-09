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
	"time"
)

const (
	TimestampKey = "ts"
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

//Handle interceptor. return nil,true if passed, otherwise return response of fail and false
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

//Invoke interceptor.
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

func getCurrentTimeInMillis() int64 {
	return time.Now().UnixNano() / 1000
}
