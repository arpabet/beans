package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
	"testing"
)

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
	cbs := list[0].(ClientBeans)

	require.Equal(t, reflect.ValueOf(clientBeans).Pointer(), reflect.ValueOf(cbs).Pointer())

	cb := cbs()
	require.Equal(t, 1, len(cb))

	require.Equal(t, clientBeanInstance, cb[0])

	list = ctx.Bean(ServerBeansClass)
	require.Equal(t, 1, len(list))
	sbs := list[0].(ServerBeans)

	require.Equal(t, reflect.ValueOf(serverBeans).Pointer(), reflect.ValueOf(sbs).Pointer())
	require.Nil(t, sbs())
}
