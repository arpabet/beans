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
	"sort"
	"testing"
)

type Component interface {
	Information() string
}

type implComponent struct {
	value string
}

func (t *implComponent) Information() string {
	return t.value
}

var coreBeanClass = reflect.TypeOf((*coreBean)(nil)) // *serviceBean
type coreBean struct {
	count int
	Components    []Component   `inject:"optional"`
}

func (t *coreBean) Inc() int {
	t.count++
	return t.count
}

var serviceBeanClass = reflect.TypeOf((*serviceBean)(nil)) // *serviceBean
type serviceBean struct {
	Core    *coreBean `inject`
	Components    []Component   `inject:"optional"`
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

func TestParentCollection(t *testing.T) {

	coreBean := &coreBean{}
	parent, err := beans.Create(
		coreBean,
		&implComponent{value:"fromParent"},
	)
	require.NoError(t, err)
	defer parent.Close()

	require.Equal(t, 1, len(coreBean.Components))
	require.Equal(t, "fromParent", coreBean.Components[0].Information())

	serviceBean := &serviceBean{testing: t}
	service, err := parent.Extend(
		serviceBean,
		&implComponent{value:"fromChild"},
	)
	require.NoError(t, err)
	defer service.Close()

	require.Equal(t, 1, len(serviceBean.Components))

	sort.Slice(serviceBean.Components, func(i, j int) bool {
		return serviceBean.Components[i].Information() < serviceBean.Components[j].Information()
	})

	require.Equal(t, "fromChild", serviceBean.Components[0].Information())
	//require.Equal(t, "fromParent", serviceBean.Components[1].Information())
}

