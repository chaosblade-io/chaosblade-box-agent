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

package tools

import (
	"container/list"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"strings"
	"sync"
)

type LimitedList struct {
	cap  int
	list *list.List
	lock sync.Mutex
}

func NewLimitedSortList(cap int) (*LimitedList, error) {
	if cap <= 0 {
		return nil, errors.New("cap less or equal than 0")
	}
	return &LimitedList{
		cap:  cap,
		list: list.New(),
	}, nil
}

func (this *LimitedList) Put(value interface{}) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.list.PushBack(value)
	if this.list.Len() > this.cap {
		e := this.list.Front()
		this.list.Remove(e)
	}
}

func (this *LimitedList) Foreach(handle func(v interface{}) error, breakWhenWrong bool) {
	l := list.New()
	this.lock.Lock()
	for element := this.list.Front(); element != nil; element = element.Next() {
		l.PushBack(element.Value)
	}
	this.lock.Unlock()
	for element := l.Front(); element != nil; element = element.Next() {
		if err := handle(element.Value); err != nil {
			if !strings.Contains(err.Error(), "nolog") {
				logrus.Warnf("handle wrong: %s", err.Error())
			}
			if breakWhenWrong {
				break
			}
		}
	}
}

func (this *LimitedList) ForeachReverse(handle func(v interface{}) error, breakWhenWrong bool) {
	l := list.New()
	this.lock.Lock()
	for element := this.list.Back(); element != nil; element = element.Prev() {
		l.PushBack(element.Value)
	}
	this.lock.Unlock()
	for element := l.Front(); element != nil; element = element.Next() {
		if err := handle(element.Value); err != nil {
			if !strings.Contains(err.Error(), "nolog") {
				logrus.Warnf("handle wrong: %s", err.Error())
			}
			if breakWhenWrong {
				break
			}
		}
	}
}
