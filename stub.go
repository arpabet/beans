/**
  Copyright (c) 2022 Arpabet, LLC. All rights reserved.
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
