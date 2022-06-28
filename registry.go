/**
    Copyright (c) 2020-2022 Arpabet, Inc.

	Permission is hereby granted, free of charge, to any person obtaining a copy
	of this software and associated documentation files (the "Software"), to deal
	in the Software without restriction, including without limitation the rights
	to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
	copies of the Software, and to permit persons to whom the Software is
	furnished to do so, subject to the following conditions:

	The above copyright notice and this permission notice shall be included in
	all copies or substantial portions of the Software.

	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
	IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
	FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
	AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
	LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
	OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
	THE SOFTWARE.
*/

package beans

import (
	"reflect"
	"sync"
)

/**
	Holds runtime information about all beans visible from current context including all parents.
 */

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

func (t *registry) findByName(name string) ([]*bean, bool) {
	t.RLock()
	defer t.RUnlock()
	list, ok := t.beansByName[name]
	return list, ok
}

func (t *registry) addBeanList(ifaceType reflect.Type, list []*bean) {
	t.Lock()
	defer t.Unlock()
	for _, b := range list {
		t.beansByType[ifaceType] = append(t.beansByType[ifaceType], b)
		t.beansByName[b.name] = append(t.beansByName[b.name], b)
	}
}

func (t *registry) addBean(ifaceType reflect.Type, b *bean) {
	t.Lock()
	defer t.Unlock()
	t.beansByType[ifaceType] = append(t.beansByType[ifaceType], b)
	t.beansByName[b.name] = append(t.beansByName[b.name], b)
}

