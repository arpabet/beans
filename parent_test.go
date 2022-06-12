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

var coreBeanClass = reflect.TypeOf((*coreBean)(nil)) // *serviceBean
type coreBean struct {
	count int
}

func (t *coreBean) Inc() int {
	t.count++
	return t.count
}

var serviceBeanClass = reflect.TypeOf((*serviceBean)(nil)) // *serviceBean
type serviceBean struct {
	Core    *coreBean `inject`
	testing *testing.T
}

func (t *serviceBean) Run() {
	require.NotNil(t.testing, t.Core)
	require.Equal(t.testing, 1, t.Core.Inc())
	require.Equal(t.testing, 2, t.Core.Inc())
	require.Equal(t.testing, 3, t.Core.Inc())
}

func TestParent(t *testing.T) {

	beans.Verbose = true

	parent, err := beans.Create(
		&coreBean{},
	)
	require.NoError(t, err)
	defer parent.Close()

	service, err := parent.Extend(
		&serviceBean{testing: t},
	)
	require.NoError(t, err)
	defer service.Close()

	p, _ := service.Parent()
	require.Equal(t, parent, p)

	b := service.Bean(serviceBeanClass)
	require.Equal(t, 1, len(b))

	b[0].Object().(*serviceBean).Run()

	b = service.Bean(coreBeanClass)
	require.Equal(t, 1, len(b))

	cnt := b[0].Object().(*coreBean).count
	require.Equal(t, 3, cnt)

}

type parentBean struct {
	testing *testing.T
}

func (t *parentBean) Destroy() error {
	// should never happened since we are not closing this context, only child one
	require.True(t.testing, false)
	return nil
}

func TestParentDestroy(t *testing.T) {

	parent, err := beans.Create(
		&parentBean{testing: t},
	)

	require.NoError(t, err)
	// defer parent.Close() for the purpose of test

	child, err := parent.Extend()
	require.NoError(t, err)
	child.Close()

}
