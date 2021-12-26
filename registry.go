/*
 *
 * Copyright 2020-present Arpabet, Inc.
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
 *
 */

package beans

import (
	"reflect"
	"sync"
)

type registry struct {
	sync.RWMutex
	beansByName map[string][]*bean
	beansByType map[reflect.Type][]*bean
}

func (t *registry) findByType(ifaceType reflect.Type) ([]*bean, bool) {
	t.RLock()
	defer t.RUnlock()
	list, ok := t.beansByType[ifaceType]
	return list, ok
}

func (t *registry) findByName(iface string) []interface{} {
	t.RLock()
	defer t.RUnlock()
	var res []interface{}
	for _, b := range t.beansByName[iface] {
		res = append(res, b.obj)
	}
	return res
}

func (t *registry) addBeanList(ifaceType reflect.Type, list *beanlist) {
	t.Lock()
	defer t.Unlock()
	list.forEach(func(b *bean) {
		t.beansByType[ifaceType] = append(t.beansByType[ifaceType], b)
		t.beansByName[b.name] = append(t.beansByName[b.name], b)
	})
}

func (t *registry) addBeanByType(ifaceType reflect.Type, b *bean) {
	t.Lock()
	defer t.Unlock()
	t.beansByType[ifaceType] = append(t.beansByType[ifaceType], b)
}

func (t *registry) addBeanByName(b *bean) {
	t.Lock()
	defer t.Unlock()
	t.beansByName[b.name] = append(t.beansByName[b.name], b)
}
