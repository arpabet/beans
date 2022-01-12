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

	firstService := ctx.Bean(FirstServiceClass)
	require.Equal(t, 2, len(firstService))

	firstService[0].(FirstService).First()

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
