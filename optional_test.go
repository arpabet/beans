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

package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
	"testing"
)

type beanA struct {
}

var BeanBClass = reflect.TypeOf((*beanB)(nil)) // *beanB
type beanB struct {
	BeanA   *beanA `inject:"optional"`
	testing *testing.T
}

func TestOptionalBeanByPointer(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&beanB{testing: t},
	)
	require.NoError(t, err)

	b := ctx.Bean(BeanBClass)
	require.Equal(t, 1, len(b))

	require.Nil(t, b[0].(*beanB).BeanA)
}

var BeanAServiceClass = reflect.TypeOf((*BeanAService)(nil)).Elem()

type BeanAService interface {
	A()
}

var BeanBServiceClass = reflect.TypeOf((*BeanBService)(nil)).Elem()

type BeanBService interface {
	B()
}

type beanBServiceImpl struct {
	BeanAService BeanAService `inject:"optional"`
	testing      *testing.T
}

func (t *beanBServiceImpl) B() {
}

func TestOptionalBeanByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&beanBServiceImpl{testing: t},
		&struct {
			BeanBService BeanBService `inject`
		}{},
	)
	require.NoError(t, err)

	b := ctx.Bean(BeanBServiceClass)
	require.Equal(t, 1, len(b))

	require.Nil(t, b[0].(*beanBServiceImpl).BeanAService)
}
