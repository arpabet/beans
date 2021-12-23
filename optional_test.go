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

	b, ok := ctx.Bean(BeanBClass)
	require.True(t, ok)

	require.Nil(t, b.(*beanB).BeanA)
}
