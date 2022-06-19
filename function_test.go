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

type functionHolder struct {
	Int         func() int               `inject`
	StringArray func() []string          `inject`
	SomeMap     func() map[string]string `inject`
}

func TestPrimitiveFunctions(t *testing.T) {

	beans.Verbose = true

	holder := &functionHolder{}

	ctx, err := beans.Create(
		holder,
		func() int { return 123 },
		func() []string { return []string{"a", "b"} },
		func() map[string]string { return map[string]string{"a": "b"} },
	)
	require.NoError(t, err)
	defer ctx.Close()

	require.Equal(t, 123, holder.Int())

	arr := holder.StringArray()
	require.Equal(t, 2, len(arr))
	require.Equal(t, "a", arr[0])
	require.Equal(t, "b", arr[1])

	m := holder.SomeMap()
	require.Equal(t, 1, len(m))
	require.Equal(t, "b", m["a"])

}

type ClientBeans func() []interface{}

var ClientBeansClass = reflect.TypeOf((ClientBeans)(nil))

type ServerBeans func() []interface{}

var ServerBeansClass = reflect.TypeOf((ServerBeans)(nil))

type funcServiceImpl struct {
	ClientBeans ClientBeans `inject`
	ServerBeans ServerBeans `inject`
}

func TestFunctions(t *testing.T) {

	println(ClientBeansClass.String())
	println(ServerBeansClass.String())

	clientBeanInstance := &struct{}{}

	clientBeans := ClientBeans(func() []interface{} {
		println("clientBeans call")
		return []interface{}{clientBeanInstance}
	})

	serverBeans := ServerBeans(func() []interface{} {
		println("serverBeans call")
		return nil
	})

	beans.Verbose = true

	srv := &funcServiceImpl{}

	ctx, err := beans.Create(
		clientBeans,
		serverBeans,
		srv,
	)
	require.NoError(t, err)
	defer ctx.Close()

	require.NotNil(t, srv.ClientBeans)
	require.NotNil(t, srv.ServerBeans)

	list := ctx.Bean(ClientBeansClass)
	require.Equal(t, 1, len(list))
	cbs := list[0].Object().(ClientBeans)

	require.Equal(t, reflect.ValueOf(clientBeans).Pointer(), reflect.ValueOf(cbs).Pointer())

	cb := cbs()
	require.Equal(t, 1, len(cb))

	require.Equal(t, clientBeanInstance, cb[0])

	list = ctx.Bean(ServerBeansClass)
	require.Equal(t, 1, len(list))
	sbs := list[0].Object().(ServerBeans)

	require.Equal(t, reflect.ValueOf(serverBeans).Pointer(), reflect.ValueOf(sbs).Pointer())
	require.Nil(t, sbs())
}
