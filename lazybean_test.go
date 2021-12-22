package beans_test

import (
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
	"testing"
)

var UnoServiceClass = reflect.TypeOf((*UnoService)(nil)).Elem()

type UnoService interface {
	beans.InitializingBean
	Uno()
}

var DosServiceClass = reflect.TypeOf((*DosService)(nil)).Elem()

type DosService interface {
	beans.InitializingBean
	Dos()
}

type unoServiceImpl struct {
	DosService func() DosService `inject`
	testing    *testing.T
}

func (t *unoServiceImpl) PostConstruct() error {
	// not yet initialized, lazy field must be nil, because DosService depends on UnoService
	require.Nil(t.testing, t.DosService())
	println("UnoPostConstruct: ", t.DosService())
	return nil
}

func (t *unoServiceImpl) Uno() {
	require.NotNil(t.testing, t.DosService())
	println("Uno")
	t.DosService().Dos()
}

type dosServiceImpl struct {
	UnoService UnoService `inject`
	testing    *testing.T
}

func (t *dosServiceImpl) PostConstruct() error {
	println("DosPostConstruct: ", t.UnoService)
	return nil
}

func (t *dosServiceImpl) Dos() {
	require.NotNil(t.testing, t.UnoService)
	println("Dos")
}

func TestLazyBeanInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&unoServiceImpl{testing: t},
		&dosServiceImpl{testing: t},

		&struct {
			UnoService `inject`
			DosService `inject`
		}{},
	)

	require.NoError(t, err)

	unoService, ok := ctx.Bean(UnoServiceClass)
	require.True(t, ok)

	unoService.(UnoService).Uno()

	dosService, ok := ctx.Bean(DosServiceClass)
	require.True(t, ok)

	dosService.(DosService).Dos()

}

var ZeroServiceClass = reflect.TypeOf((*zeroService)(nil))

type zeroService struct {
	beans.InitializingBean
	UnService func() *unService `inject`
	testing   *testing.T
}

func (t *zeroService) PostConstruct() error {
	// not yet initialized, lazy field must be nil, because UnService depends on ZeroService
	require.Nil(t.testing, t.UnService())
	println("ZeroPostConstruct: ", t.UnService())
	return nil
}

func (t *zeroService) Zero() {
	require.NotNil(t.testing, t.UnService())
	println("Zero")
	t.UnService().Un()
}

var UnServiceClass = reflect.TypeOf((*unService)(nil))

type unService struct {
	beans.InitializingBean
	ZeroService *zeroService `inject`
	testing     *testing.T
}

func (t *unService) PostConstruct() error {
	println("UnPostConstruct: ", t.ZeroService)
	return nil
}

func (t *unService) Un() {
	require.NotNil(t.testing, t.ZeroService)
	println("Un")
}

func TestLazyBeanPointers(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&zeroService{testing: t},
		&unService{testing: t},
	)

	require.NoError(t, err)

	zero, ok := ctx.Bean(ZeroServiceClass)
	require.True(t, ok)

	zero.(*zeroService).Zero()

	un, ok := ctx.Bean(UnServiceClass)
	require.True(t, ok)

	un.(*unService).Un()

}
