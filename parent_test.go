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

	service, err := parent.Extend(
		&serviceBean{testing: t},
	)

	require.NoError(t, err)
	p, _ := service.Parent()
	require.Equal(t, parent, p)

	b := service.Bean(serviceBeanClass)
	require.Equal(t, 1, len(b))

	b[0].(*serviceBean).Run()

}
