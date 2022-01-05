package beans

import (
	"github.com/pkg/errors"
	"reflect"
)

type namedBeanStub struct {
	name string
}

func (t *namedBeanStub) BeanName() string {
	return t.name
}

type initializingBeanStub struct {
	name string
}

func (t *initializingBeanStub) PostConstruct() error {
	return errors.Errorf("bean '%s' does not implement PostConstruct method, but has anonymous field InitializingBean", t.name)
}

type disposableBeanStub struct {
	name string
}

func (t *disposableBeanStub) Destroy() error {
	return errors.Errorf("bean '%s' does not implement Destroy method, but has anonymous field DisposableBean", t.name)
}

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
