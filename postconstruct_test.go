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
	"errors"
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"reflect"
	"strings"
	"testing"
)

var ServerServiceClass = reflect.TypeOf((*ServerService)(nil)).Elem()

type ServerService interface {
	beans.InitializingBean
	beans.DisposableBean
	IsInitialized() bool
	IsDestroyed() bool
	Serve(app string) error
}

type beanServer struct {
	initialized bool
	destroyed   bool
	throwError  bool
}

func (t *beanServer) Serve(app string) error {
	println("ServerService.Serve: ", app)
	return nil
}

func (t *beanServer) PostConstruct() error {
	if t.throwError {
		println("ServerService.PostConstruct Error")
		return errors.New("server construct error")
	}
	println("ServerService.PostConstruct")
	t.initialized = true
	return nil
}

func (t *beanServer) IsInitialized() bool {
	return t.initialized
}

func (t *beanServer) Destroy() error {
	println("ServerService.Destroy")
	t.destroyed = true
	return nil
}

func (t *beanServer) IsDestroyed() bool {
	return t.destroyed
}

var ClientServiceClass = reflect.TypeOf((*ClientService)(nil)).Elem()

type ClientService interface {
	beans.InitializingBean
	beans.DisposableBean
	Run(app string) error
}

type beanClient struct {
	testing       *testing.T
	ServerService ServerService `inject`
}

func (t *beanClient) PostConstruct() error {
	println("ClientService.PostConstruct")
	require.NotNil(t.testing, t.ServerService)
	require.True(t.testing, t.ServerService.IsInitialized())
	return nil
}

func (t *beanClient) Run(app string) error {
	println("ClientService.Run: ", app)
	return t.ServerService.Serve(app)
}

func (t *beanClient) Destroy() error {
	println("ClientService.Destroy")
	require.NotNil(t.testing, t.ServerService)
	require.False(t.testing, t.ServerService.IsDestroyed())
	return nil
}

func TestPostConstruct(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&beanClient{testing: t},
		&beanServer{},
		/**
		  enum all interfaces in context, to make sure that all of them are initialized
		*/
		&struct {
			ClientService `inject`
			ServerService `inject`
		}{},
	)
	require.NoError(t, err)

	defer ctx.Close()

	client := ctx.Bean(ClientServiceClass)
	require.Equal(t, 1, len(client))

	client[0].(ClientService).Run("t3st")

}

func TestPostConstructWithError(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&beanClient{testing: t},
		&beanServer{throwError: true},
		/**
		  enum all interfaces in context, to make sure that all of them are initialized
		*/
		&struct {
			ClientService `inject`
			ServerService `inject`
		}{},
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "server construct error"))
	println(err.Error())

}

type aService struct {
	beans.InitializingBean
	testing  *testing.T
	BService *bService `inject`
}

func (t *aService) PostConstruct() error {
	println("a.PostConstruct")
	require.NotNil(t.testing, t.BService)
	return nil
}

type bService struct {
	beans.InitializingBean
	testing  *testing.T
	CService *cService `inject`
}

func (t *bService) PostConstruct() error {
	println("b.PostConstruct")
	require.NotNil(t.testing, t.CService)
	return nil
}

type cService struct {
	beans.InitializingBean
	testing  *testing.T
	AService *aService `inject`
	//LazyAService func() *aService `inject`
}

func (t *cService) PostConstruct() error {
	println("c.PostConstruct")
	require.NotNil(t.testing, t.AService)
	return nil
}

func TestPostConstructCycle(t *testing.T) {

	beans.Verbose = true

	ctx, err := beans.Create(
		&aService{testing: t},
		&bService{testing: t},
		&cService{testing: t},
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "cycle"))
	println(err.Error())
}
