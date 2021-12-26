package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
	"strings"
	"testing"
)

var FirstBeanClass = reflect.TypeOf((*firstBean)(nil)) // *firstBean
type firstBean struct {
}

var SecondBeanClass = reflect.TypeOf((*secondBean)(nil)) // *secondBean
type secondBean struct {
	FirstBean *firstBean `inject:"-"`
	testing   *testing.T
}

func (t *secondBean) Run() {
	require.NotNil(t.testing, t.FirstBean)
}

func TestBeanByPointer(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstBean{},
		&secondBean{testing: t},
	)
	require.NoError(t, err)

	second := ctx.Bean(SecondBeanClass)
	require.Equal(t, 1, len(second))

	second[0].(*secondBean).Run()

}

func TestMultipleBeanByPointer(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstBean{},
		&firstBean{},
		&secondBean{testing: t},
	)

	require.Error(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "multiple candidates"))
	println(err.Error())

}

func TestBeanByStruct(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		firstBean{},
		&secondBean{testing: t},
	)
	require.Error(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "non-pointer"))

}

var FirstServiceClass = reflect.TypeOf((*FirstService)(nil)).Elem()

type FirstService interface {
	First()
}

var SecondServiceClass = reflect.TypeOf((*SecondService)(nil)).Elem()

type SecondService interface {
	Second()
}

type firstServiceImpl struct {
	testing *testing.T
}

func (t *firstServiceImpl) First() {
	require.True(t.testing, true)
}

type secondServiceImpl struct {
	FirstService FirstService `inject`
	testing      *testing.T
}

func (t *secondServiceImpl) Second() {
	require.NotNil(t.testing, t.FirstService)
}

func TestBeanByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstServiceImpl{testing: t},
		&secondServiceImpl{testing: t},

		&struct {
			FirstService  `inject`
			SecondService `inject`
		}{},
	)

	require.NoError(t, err)

	firstService := ctx.Bean(FirstServiceClass)
	require.Equal(t, 1, len(firstService))

	firstService[0].(FirstService).First()

	secondService := ctx.Bean(SecondServiceClass)
	require.Equal(t, 1, len(secondService))

	secondService[0].(SecondService).Second()

}

type firstService2Impl struct {
	testing *testing.T
}

func (t *firstService2Impl) First() {
	require.True(t.testing, true)
}

func TestMultipleBeansByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstServiceImpl{testing: t},
		&firstService2Impl{testing: t},

		&struct {
			FirstService `inject:"-"`
		}{},
	)

	require.Error(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "two or more"))

}

func TestSpecificBeanByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstServiceImpl{testing: t},
		&firstService2Impl{testing: t},

		&struct {
			FirstService `inject:"bean=*beans_test.firstServiceImpl"`
		}{},
	)

	require.NoError(t, err)

	firstService := ctx.Bean(FirstServiceClass)
	require.Equal(t, 1, len(firstService))

	firstService[0].(FirstService).First()

}

func TestNotFoundSpecificBeanByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstServiceImpl{testing: t},
		&firstService2Impl{testing: t},

		&struct {
			FirstService `inject:"bean=*beans_test.unknownBean"`
		}{},
	)

	require.Error(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "specific"))

}
