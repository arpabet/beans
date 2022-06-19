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

var reloadableBeanClass = reflect.TypeOf((*reloadableBean)(nil))

type reloadableBean struct {
	constructed int
	destroyed   int
}

func (t *reloadableBean) PostConstruct() error {
	t.constructed++
	return nil
}

func (t *reloadableBean) Destroy() error {
	t.destroyed++
	return nil
}

type topBean struct {
	ReloadableBean *reloadableBean `inject`
}

func TestBeanReload(t *testing.T) {

	beans.Verbose = true

	reBean := &reloadableBean{}
	tBean := &topBean{}

	// initialization order
	ctx, err := beans.Create(
		reBean,
		tBean,
	)
	require.NoError(t, err)

	require.Equal(t, 1, reBean.constructed)
	require.Equal(t, 0, reBean.destroyed)
	require.True(t, tBean.ReloadableBean == reBean)

	list := ctx.Bean(reloadableBeanClass)
	require.Equal(t, 1, len(list))
	require.Equal(t, reBean, list[0].Object())

	err = list[0].Reload()
	require.NoError(t, err)

	require.Equal(t, 2, reBean.constructed)
	require.Equal(t, 1, reBean.destroyed)

	ctx.Close()

	require.Equal(t, 2, reBean.constructed)
	require.Equal(t, 2, reBean.destroyed)
	require.True(t, tBean.ReloadableBean == reBean)

}
