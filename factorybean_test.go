package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
	"strings"
	"testing"
)

type someService struct {
	beans.InitializingBean
	Initialized bool
	testing     *testing.T
}

func (t *someService) PostConstruct() error {
	println("*someService.PostConstruct")
	t.Initialized = true
	return nil
}

func (t *someService) GetProperty() string {
	require.True(t.testing, t.Initialized)
	println("*someService.GetProperty", t)
	return "someProperty"
}

var beanConstructedClass = reflect.TypeOf((*beanConstructed)(nil))

type beanConstructed struct {
	someService *someService
	testing     *testing.T
}

func (t *beanConstructed) Run() error {
	require.NotNil(t.testing, t.someService)
	require.True(t.testing, t.someService.Initialized)
	println("*beanConstructed.Run")
	return nil
}

type factoryBeanExample struct {
	beans.FactoryBean
	testing     *testing.T
	SomeService *someService `inject`
}

func (t *factoryBeanExample) Object() interface{} {
	require.NotNil(t.testing, t.SomeService)
	someProperty := t.SomeService.GetProperty()
	println("Construct beanConstructed after ", someProperty)
	return &beanConstructed{someService: t.SomeService, testing: t.testing}
}

func (t *factoryBeanExample) ObjectType() reflect.Type {
	return beanConstructedClass
}

func (t *factoryBeanExample) Singleton() bool {
	return true
}

type applicationContext struct {
	BeanConstructed *beanConstructed `inject`
}

type repeatedFactoryBeanExample struct {
	beans.FactoryBean
	testing *testing.T
}

func (t *repeatedFactoryBeanExample) Object() interface{} {
	return &beanConstructed{testing: t.testing}
}

func (t *repeatedFactoryBeanExample) ObjectType() reflect.Type {
	return beanConstructedClass
}

func (t *repeatedFactoryBeanExample) Singleton() bool {
	return true
}

func TestRepeatedFactoryBean(t *testing.T) {

	beans.Verbose = true

	app := &applicationContext{}
	ctx, err := beans.Create(
		&someService{testing: t},
		&factoryBeanExample{testing: t},
		&repeatedFactoryBeanExample{testing: t},
		app,
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "repeated"))
	println(err.Error())
}

func TestFactoryBean(t *testing.T) {

	beans.Verbose = true

	app := &applicationContext{}
	ctx, err := beans.Create(
		app,
		&factoryBeanExample{testing: t},
		&someService{testing: t},
	)
	require.NoError(t, err)

	defer ctx.Close()

	require.NotNil(t, app.BeanConstructed)
	err = app.BeanConstructed.Run()
	require.NoError(t, err)
}

var SomeServiceClass = reflect.TypeOf((*SomeService)(nil)).Elem()

type SomeService interface {
	beans.InitializingBean
	Initialized() bool
	GetProperty() string
}

type someServiceImpl struct {
	initialized bool
	testing     *testing.T
}

func (t *someServiceImpl) PostConstruct() error {
	println("*someServiceImpl.PostConstruct")
	t.initialized = true
	return nil
}

func (t *someServiceImpl) Initialized() bool {
	return t.initialized
}

func (t *someServiceImpl) GetProperty() string {
	require.True(t.testing, t.initialized)
	println("*someServiceImpl.GetProperty", t)
	return "someProperty"
}

var BeanConstructedClass = reflect.TypeOf((*BeanConstructed)(nil)).Elem()

type BeanConstructed interface {
	Run() error
}

type beanConstructedImpl struct {
	someService SomeService
	testing     *testing.T
}

func (t *beanConstructedImpl) Run() error {
	require.NotNil(t.testing, t.someService)
	require.True(t.testing, t.someService.Initialized())
	println("*beanConstructedImpl.Run")
	return nil
}

type factoryBeanImpl struct {
	beans.FactoryBean
	testing     *testing.T
	SomeService SomeService `inject`
}

func (t *factoryBeanImpl) Object() interface{} {
	require.NotNil(t.testing, t.SomeService)
	someProperty := t.SomeService.GetProperty()
	println("Construct beanConstructedImpl after ", someProperty)
	return &beanConstructedImpl{someService: t.SomeService, testing: t.testing}
}

func (t *factoryBeanImpl) ObjectType() reflect.Type {
	return BeanConstructedClass
}

func (t *factoryBeanImpl) Singleton() bool {
	return true
}

func TestFactoryInterfaceBean(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&factoryBeanImpl{testing: t},
		&someServiceImpl{testing: t},
		&struct {
			BeanConstructed `inject`
		}{},
	)
	require.NoError(t, err)

	defer ctx.Close()

	bc, ok := ctx.Bean(BeanConstructedClass)
	require.True(t, ok)

	err = bc.(BeanConstructed).Run()
	require.NoError(t, err)
}