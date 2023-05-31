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

type Set struct {
	m map[interface{}]bool
}

func NewSet() *Set {
	return &Set{
		m: make(map[interface{}]bool),
	}
}

func (set *Set) Add(item interface{}) {
	set.m[item] = true
}

func (set *Set) Remove(item interface{}) {
	delete(set.m, item)
}

func (set *Set) Contains(item interface{}) bool {
	ok := set.m[item]
	return ok
}

func (set *Set) Length() int {
	return len(set.m)
}

func (set *Set) Clear() {
	set.m = map[interface{}]bool{}
}

func (set *Set) Keys() []interface{} {
	keys := make([]interface{}, 0, len(set.m))
	for key := range set.m {
		keys = append(keys, key)
	}
	return keys
}

func (set *Set) StringKeys() []string {
	keys := make([]string, 0, len(set.m))
	for key := range set.m {
		keys = append(keys, key.(string))
	}
	return keys
}
