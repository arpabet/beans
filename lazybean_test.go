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
	DosService func() DosService `inject:"lazy"`
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
			UnoService UnoService `inject`
			DosService DosService `inject`
		}{},
	)

	require.NoError(t, err)

	unoService := ctx.Bean(UnoServiceClass)
	require.Equal(t, 1, len(unoService))

	unoService[0].(UnoService).Uno()

	dosService := ctx.Bean(DosServiceClass)
	require.Equal(t, 1, len(dosService))

	dosService[0].(DosService).Dos()

}

var ZeroServiceClass = reflect.TypeOf((*zeroService)(nil))

type zeroService struct {
	beans.InitializingBean
	UnService func() *unService `inject:"lazy"`
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

	zero := ctx.Bean(ZeroServiceClass)
	require.Equal(t, 1, len(zero))

	zero[0].(*zeroService).Zero()

	un := ctx.Bean(UnServiceClass)
	require.Equal(t, 1, len(un))

	un[0].(*unService).Un()

}
