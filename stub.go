/*
 *
 * Copyright 2020-present Arpabet LLC.
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

func (t *factoryBeanStub) Singleton() bool {
	return true
}
