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
	"testing"
)

var ComponentClass = reflect.TypeOf((*Component)(nil)).Elem()
type Component interface {
	beans.OrderedBean
	Information() string
}

type implComponent struct {
	value string
	order int
}

func (t *implComponent) Information() string {
	return t.value
}

func (t *implComponent) BeanOrder() int {
	return t.order
}

var implElementClass = reflect.TypeOf((*implElement)(nil)) // *firstBean
type implElement struct {
	value string
	order int
}

func (t *implElement) BeanOrder() int {
	return t.order
}

var coreBeanClass = reflect.TypeOf((*coreBean)(nil)) // *serviceBean
type coreBean struct {
	count int
	Components    []Component   `inject:"optional"`
	Elements      []*implElement   `inject:"optional"`
}

func (t *coreBean) Inc() int {
	t.count++
	return t.count
}

var serviceBeanClass = reflect.TypeOf((*serviceBean)(nil)) // *serviceBean
type serviceBean struct {
	Core    *coreBean `inject`
	Components    []Component   `inject:"optional,level=1"`   // default level is 1, only current context
	Elements      []*implElement   `inject:"optional,level=1"`
	Components2   []Component   `inject:"optional,level=2"` // level 2 is current context plus parent context
	Elements2     []*implElement   `inject:"optional,level=2"`
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

	b := service.Bean(serviceBeanClass, beans.DefaultLevel)
	require.Equal(t, 1, len(b))

	b[0].Object().(*serviceBean).Run()

	b = service.Bean(coreBeanClass, beans.DefaultLevel)
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
		&implComponent{value:"fromParent", order: 1},
		&implElement{value: "parent", order: 2},
	)
	require.NoError(t, err)
	defer parent.Close()

	require.Equal(t, 1, len(coreBean.Components))
	require.Equal(t, "fromParent", coreBean.Components[0].Information())

	serviceBean := &serviceBean{testing: t}
	child, err := parent.Extend(
		serviceBean,
		&implComponent{value:"fromChild", order: 2},
		&implElement{value: "child", order: 1},
	)
	require.NoError(t, err)
	defer child.Close()

	require.Equal(t, 2, len(serviceBean.Elements2))
	require.Equal(t, "child", serviceBean.Elements2[0].value)
	require.Equal(t, "parent", serviceBean.Elements2[1].value)

	require.Equal(t, 2, len(serviceBean.Components2))

	require.Equal(t, "fromParent", serviceBean.Components2[0].Information())
	require.Equal(t, "fromChild", serviceBean.Components2[1].Information())

	/*
	Check runtime bean access
	 */

	list := parent.Bean(ComponentClass, -1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "fromParent", list[0].Object().(Component).Information())

	list = parent.Lookup("*beans_test.implComponent", -1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "fromParent", list[0].Object().(Component).Information())

	list = parent.Bean(implElementClass, -1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "parent", list[0].Object().(*implElement).value)

	list = parent.Lookup("*beans_test.implElement", -1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "parent", list[0].Object().(*implElement).value)

	/*
	Test interface injected child context
	 */

	list = child.Bean(ComponentClass, 0)
	require.Equal(t, 1, len(list))
	require.Equal(t, "fromChild", list[0].Object().(Component).Information())

	list = child.Lookup("*beans_test.implComponent", 0)
	require.Equal(t, 1, len(list))
	require.Equal(t, "fromChild", list[0].Object().(Component).Information())

	list = child.Bean(ComponentClass, 1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "fromChild", list[0].Object().(Component).Information())

	list = child.Lookup("*beans_test.implComponent", 1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "fromChild", list[0].Object().(Component).Information())

	list = child.Bean(ComponentClass, 2)  // include parent context
	require.Equal(t, 2, len(list))

	list = child.Lookup("*beans_test.implComponent", 2)  // include parent context
	require.Equal(t, 2, len(list))

	list = child.Bean(ComponentClass, 3)  // include parent context
	require.Equal(t, 2, len(list))

	list = child.Bean(ComponentClass, -1)  // include parent context
	require.Equal(t, 2, len(list))

	require.Equal(t, "fromParent", list[0].Object().(Component).Information())
	require.Equal(t, "fromChild", list[1].Object().(Component).Information())

	/*
		Test pointer injected child context
	*/

	list = child.Bean(implElementClass, 0)
	require.Equal(t, 1, len(list))
	require.Equal(t, "child", list[0].Object().(*implElement).value)

	list = child.Lookup("*beans_test.implElement", 0)
	require.Equal(t, 1, len(list))
	require.Equal(t, "child", list[0].Object().(*implElement).value)

	list = child.Bean(implElementClass, 1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "child", list[0].Object().(*implElement).value)

	list = child.Lookup("*beans_test.implElement", 1)
	require.Equal(t, 1, len(list))
	require.Equal(t, "child", list[0].Object().(*implElement).value)

	list = child.Bean(implElementClass, 2)
	require.Equal(t, 2, len(list))

	list = child.Lookup("*beans_test.implElement", 2)
	require.Equal(t, 2, len(list))

	list = child.Bean(implElementClass, 3)
	require.Equal(t, 2, len(list))

	list = child.Bean(implElementClass, -1)
	require.Equal(t, 2, len(list))

	require.Equal(t, "child", list[0].Object().(*implElement).value)
	require.Equal(t, "parent", list[1].Object().(*implElement).value)

}

