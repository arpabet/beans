/*
 *
 * Copyright 2020-present Arpabet LLC.
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
	Initialized() bool
}

var DosServiceClass = reflect.TypeOf((*DosService)(nil)).Elem()

type DosService interface {
	beans.InitializingBean
	Dos()
	Initialized() bool
}

type unoServiceImpl struct {
	DosService DosService `inject:"lazy"`
	testing    *testing.T
	initialized bool
}

func newUnoService(t *testing.T) UnoService {
	return &unoServiceImpl{testing: t}
}

func (t *unoServiceImpl) Initialized() bool {
	return t.initialized
}

// when this method called, Context is in initialization stage, so lazy bean can be not initialized
func (t *unoServiceImpl) PostConstruct() error {
	// not yet initialized, lazy field can not be nil (this is not optional field that can be nil),
	// but DosService not initialized, because DosService depends on UnoService
	require.NotNil(t.testing, t.DosService)
	println("UnoPostConstruct: DosService initialized=", t.DosService.Initialized())
	require.False(t.testing, t.DosService.Initialized())
	t.initialized = true
	return nil
}

// when this method called, Context already created, all beans must be initialized
func (t *unoServiceImpl) Uno() {
	require.NotNil(t.testing, t.DosService)
	println("Uno: DosService initialized=", t.DosService.Initialized())
	require.True(t.testing, t.DosService.Initialized())
	t.DosService.Dos()
}

type dosServiceImpl struct {
	UnoService UnoService `inject`
	testing    *testing.T
	initialized bool
}

func newDosService(t *testing.T) DosService {
	return &dosServiceImpl{testing: t}
}

func (t *dosServiceImpl) PostConstruct() error {
	require.NotNil(t.testing, t.UnoService)
	println("DosPostConstruct: UnoService initialized=", t.UnoService.Initialized())
	// this class does not have lazy dependency, therefore UnoService must be initialized before PostConstruct call
	require.True(t.testing, t.UnoService.Initialized())
	t.initialized = true
	return nil
}

func (t *dosServiceImpl) Initialized() bool {
	return t.initialized
}

// when this method called, Context already created, all beans must be initialized
func (t *dosServiceImpl) Dos() {
	require.NotNil(t.testing, t.UnoService)
	println("Dos: UnoService initialized=", t.UnoService.Initialized())
	require.True(t.testing, t.UnoService.Initialized())
}

func TestLazyBeanInterface(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		newUnoService(t),
		newDosService(t),

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
	UnService  *unService `inject:"lazy"`
	testing   *testing.T
	Initialized bool
}

// when this method called, Context is in initialization stage, so lazy bean can be not initialized
func (t *zeroService) PostConstruct() error {
	// not yet initialized, lazy field can not be nil (this is not optional field that can be nil),
	// but unService is not initialized, because zeroService depends on unService
	require.NotNil(t.testing, t.UnService)
	println("ZeroPostConstruct: unService initialized=", t.UnService.Initialized)
	t.Initialized = true
	return nil
}

// when this method called, Context already created, all beans must be initialized
func (t *zeroService) Zero() {
	require.NotNil(t.testing, t.UnService)
	println("Zero")
	require.True(t.testing, t.UnService.Initialized)
	t.UnService.Un()
}

var UnServiceClass = reflect.TypeOf((*unService)(nil))

type unService struct {
	beans.InitializingBean
	ZeroService *zeroService `inject`
	testing     *testing.T
	Initialized bool
}

func (t *unService) PostConstruct() error {
	require.NotNil(t.testing, t.ZeroService)
	println("UnPostConstruct: zeroService initialized=", t.ZeroService.Initialized)
	t.Initialized = true
	return nil
}

// when this method called, Context already created, all beans must be initialized
func (t *unService) Un() {
	require.NotNil(t.testing, t.ZeroService)
	println("Un")
	require.True(t.testing, t.ZeroService.Initialized)
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
