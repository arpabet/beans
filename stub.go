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
	"github.com/pkg/errors"
	"reflect"
)

/**
Named Bean Stub is using to replace empty field in struct that has beans.NamedBean type
*/

type namedBeanStub struct {
	name string
}

func (t *namedBeanStub) BeanName() string {
	return t.name
}

/**
Ordered Bean Stub is using to replace empty field in struct that has beans.OrderedBean type
*/

type orderedBeanStub struct {
}

func (t *orderedBeanStub) BeanOrder() int {
	return 0
}

/**
Initializing Bean Stub is using to replace empty field in struct that has beans.InitializingBean type
*/

type initializingBeanStub struct {
	name string
}

func (t *initializingBeanStub) PostConstruct() error {
	return errors.Errorf("bean '%s' does not implement PostConstruct method, but has anonymous field InitializingBean", t.name)
}

/**
Disposable Bean Stub is using to replace empty field in struct that has beans.DisposableBean type
*/

type disposableBeanStub struct {
	name string
}

func (t *disposableBeanStub) Destroy() error {
	return errors.Errorf("bean '%s' does not implement Destroy method, but has anonymous field DisposableBean", t.name)
}

/**
Factory Bean Stub is using to replace empty field in struct that has beans.FactoryBean type
*/

type factoryBeanStub struct {
	name     string
	elemType reflect.Type
}

func (t *factoryBeanStub) Object() (interface{}, error) {
	return nil, errors.Errorf("bean '%s' does not implement Object method, but has anonymous field FactoryBean", t.name)
}

func (t *factoryBeanStub) ObjectType() reflect.Type {
	return t.elemType
}

func (t *factoryBeanStub) ObjectName() string {
	return ""
}

func (t *factoryBeanStub) Singleton() bool {
	return true
}
