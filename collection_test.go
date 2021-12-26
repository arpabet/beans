package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
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
