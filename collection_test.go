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

type elementX struct {
	beans.NamedBean
	name string
}

func (t *elementX) BeanName() string {
	return t.name
}

var holderXClass = reflect.TypeOf((*holderX)(nil)) // *holderX
type holderX struct {
	Array []*elementX `inject`
	//Map    map[string]*elementX   `inject`
	testing *testing.T
}

func TestArrayRequiredByPointer(t *testing.T) {

	_, err := beans.Create(
		&holderX{testing: t},
	)
	require.NotNil(t, err)
	println(err.Error())
	require.True(t, strings.Contains(err.Error(), "can not find candidates"))

}

func TestArrayByPointer(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&elementX{name: "a"},
		&elementX{name: "b"},
		&elementX{name: "c"},
		&holderX{testing: t},
	)
	require.NoError(t, err)

	b := ctx.Bean(holderXClass)
	require.Equal(t, 1, len(b))

	holder := b[0].(*holderX)
	require.NotNil(t, holder.Array)
	require.Equal(t, 3, len(holder.Array))

	require.Equal(t, "a", holder.Array[0].name)
	require.Equal(t, "b", holder.Array[1].name)
	require.Equal(t, "c", holder.Array[2].name)

	el := ctx.Lookup("a")
	require.Equal(t, 1, len(el))
	require.Equal(t, "a", el[0].(*elementX).BeanName())

	el = ctx.Lookup("b")
	require.Equal(t, 1, len(el))
	require.Equal(t, "b", el[0].(*elementX).BeanName())

	el = ctx.Lookup("c")
	require.Equal(t, 1, len(el))
	require.Equal(t, "c", el[0].(*elementX).BeanName())

}

var ElementClass = reflect.TypeOf((*Element)(nil)).Elem()

type Element interface {
	beans.NamedBean
}

var HolderClass = reflect.TypeOf((*Holder)(nil)).Elem()

type Holder interface {
	Elements() []Element
}

type elementImpl struct {
	name string
}

func (t *elementImpl) BeanName() string {
	return t.name
}

type holderImpl struct {
	Array   []Element `inject`
	testing *testing.T
}

func (t *holderImpl) Elements() []Element {
	return t.Array
}

func TestArrayRequiredByInterface(t *testing.T) {

	_, err := beans.Create(
		&holderImpl{testing: t},
	)
	require.NotNil(t, err)
	println(err.Error())
	require.True(t, strings.Contains(err.Error(), "can not find candidates"))

}

func TestArrayByInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&elementImpl{name: "a"},
		&elementImpl{name: "b"},
		&elementImpl{name: "c"},
		&holderImpl{testing: t},
	)
	require.NoError(t, err)

	b := ctx.Bean(HolderClass)
	require.Equal(t, 1, len(b))
	holder := b[0].(Holder)

	require.Equal(t, 3, len(holder.Elements()))

	require.Equal(t, "a", holder.Elements()[0].BeanName())
	require.Equal(t, "b", holder.Elements()[1].BeanName())
	require.Equal(t, "c", holder.Elements()[2].BeanName())

	el := ctx.Lookup("a")
	require.Equal(t, 1, len(el))
	require.Equal(t, "a", el[0].(Element).BeanName())

	el = ctx.Lookup("b")
	require.Equal(t, 1, len(el))
	require.Equal(t, "b", el[0].(Element).BeanName())

	el = ctx.Lookup("c")
	require.Equal(t, 1, len(el))
	require.Equal(t, "c", el[0].(Element).BeanName())

}

type specificHolderImpl struct {
	Array   []Element `inject:"bean=a"`
	testing *testing.T
}

func (t *specificHolderImpl) Elements() []Element {
	return t.Array
}

func TestArraySpecificByInterface(t *testing.T) {

	ctx, err := beans.Create(
		&elementImpl{name: "a"},
		&elementImpl{name: "a"},
		&elementImpl{name: "b"},
		&specificHolderImpl{testing: t},
	)
	require.NoError(t, err)

	b := ctx.Bean(HolderClass)
	require.Equal(t, 1, len(b))
	holder := b[0].(Holder)

	require.Equal(t, 2, len(holder.Elements()))

}
