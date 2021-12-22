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
	FirstBean *firstBean `inject`
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

	second, ok := ctx.Bean(SecondBeanClass)
	require.True(t, ok)

	second.(*secondBean).Run()

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

	firstService, ok := ctx.Bean(FirstServiceClass)
	require.True(t, ok)

	firstService.(FirstService).First()

	secondService, ok := ctx.Bean(SecondServiceClass)
	require.True(t, ok)

	secondService.(SecondService).Second()

}