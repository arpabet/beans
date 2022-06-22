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
	defer ctx.Close()

	second := ctx.Bean(SecondBeanClass)
	require.Equal(t, 1, len(second))

	second[0].Object().(*secondBean).Run()

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

func TestSearchBeanByPointerNotFound(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstBean{},
	)
	require.NoError(t, err)
	defer ctx.Close()

	second := ctx.Bean(SecondBeanClass)
	require.Equal(t, 0, len(second))

}

func TestBeanByStruct(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		firstBean{},
		&secondBean{testing: t},
	)
	require.Error(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "could be a pointer or function"))

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
			FirstService  FirstService  `inject`
			SecondService SecondService `inject`
		}{},
	)

	require.NoError(t, err)
	defer ctx.Close()

	firstService := ctx.Bean(FirstServiceClass)
	require.Equal(t, 1, len(firstService))

	firstService[0].Object().(FirstService).First()

	secondService := ctx.Bean(SecondServiceClass)
	require.Equal(t, 1, len(secondService))

	secondService[0].Object().(SecondService).Second()

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
			FirstService FirstService `inject:"-"`
		}{},
	)

	require.Error(t, err)
	require.Nil(t, ctx)
	println(err.Error())
	require.True(t, strings.Contains(err.Error(), "multiple candidates"))

}

func TestSpecificBeanByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstServiceImpl{testing: t},
		&firstService2Impl{testing: t},

		&struct {
			FirstService FirstService `inject:"bean=*beans_test.firstServiceImpl"`
		}{},
	)

	require.NoError(t, err)
	defer ctx.Close()

	firstService := ctx.Bean(FirstServiceClass)
	require.Equal(t, 2, len(firstService))

	firstService[0].Object().(FirstService).First()

}

func TestNotFoundSpecificBeanByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&firstServiceImpl{testing: t},
		&firstService2Impl{testing: t},

		&struct {
			FirstService FirstService `inject:"bean=*beans_test.unknownBean"`
		}{},
	)

	require.Error(t, err)
	require.Nil(t, ctx)
	println(err.Error())
	require.True(t, strings.Contains(err.Error(), "can not find candidates"))

}
