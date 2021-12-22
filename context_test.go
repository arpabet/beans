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
	"fmt"
	"github.com/stretchr/testify/require"
	"go.arpabet.com/beans"
	"log"
	"os"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"testing"
)

func TestCreateNil(t *testing.T) {

	ctx, err := beans.Create(nil)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "null object"))
}

func TestCreateEmpty(t *testing.T) {

	ctx, err := beans.Create()
	require.Nil(t, err)
	require.NotNil(t, ctx)
	require.Equal(t, 1, len(ctx.Core()))

	c, ok := ctx.Bean(beans.ContextClass)
	require.True(t, ok)
	require.NotNil(t, c)
	require.Equal(t, ctx, c)

}

var StorageClass = reflect.TypeOf((*Storage)(nil)).Elem()

type Storage interface {
	Load(key string) string
	Store(key, value string)
}

var ConfigServiceClass = reflect.TypeOf((*ConfigService)(nil)).Elem()

type ConfigService interface {
	GetConfig(key string) string
	SetConfig(key, value string)
}

var UserServiceClass = reflect.TypeOf((*UserService)(nil)).Elem()

type UserService interface {
	GetUser(user string) string
	SaveUser(user, details string)
}

var AppServiceClass = reflect.TypeOf((*AppService)(nil)).Elem()

type AppService interface {
	GetContext() beans.Context
}

type storageImpl struct {
	Logger   *log.Logger `inject`
	internal sync.Map
}

func (t *storageImpl) Load(key string) string {
	t.Logger.Printf("Load %s\n", key)
	if val, ok := t.internal.Load(key); ok {
		return val.(string)
	} else {
		return ""
	}
}

func (t *storageImpl) Store(key, value string) {
	t.Logger.Printf("Store %s, %s\n", key, value)
	t.internal.Store(key, value)
}

type configServiceImpl struct {
	Storage `inject`
}

func (t *configServiceImpl) GetConfig(key string) string {
	return t.Load("config:" + key)
}

func (t *configServiceImpl) SetConfig(key, value string) {
	t.Store("config:"+key, value)
}

type userServiceImpl struct {
	Storage       `inject`
	ConfigService `inject`
}

func (t *userServiceImpl) GetUser(user string) string {
	return t.Load("user:" + user)
}

func (t *userServiceImpl) SaveUser(user, details string) {
	if t.allowWrites() {
		t.Store("user:"+user, details)
	}
}

func (t *userServiceImpl) allowWrites() bool {
	b, err := strconv.ParseBool(t.GetConfig("allowWrites"))
	if err != nil {
		return false
	}
	return b
}

func (t *userServiceImpl) PostConstruct() error {
	t.SetConfig("allowWrites", "true")
	return nil
}

type appServiceImpl struct {
	beans.Context `inject`
}

func (t *appServiceImpl) GetContext() beans.Context {
	return t.Context
}

func TestCreateEmptyObject(t *testing.T) {

	ctx, err := beans.Create(
		&storageImpl{}, // requires log, but we forgot to add it to this context
		/**
		  needed to define usage of interfaces
		*/
		&struct {
			Storage `inject`
		}{},
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "*log.Logger"))
}

func TestCreateDoubleObjects(t *testing.T) {

	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	ctx, err := beans.Create(
		logger,
		&storageImpl{}, // first is ok
		&storageImpl{}, // second singleton is too much
		/**
		  needed to define usage of interfaces
		*/
		&struct {
			Storage `inject`
		}{},
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "repeated"))
	require.True(t, strings.Contains(err.Error(), "*beans_test.storageImpl"))
}

func TestCreate(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	var ctx, err = beans.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		&appServiceImpl{},
		/**
		  needed to define usage of UserService in context in order to register bean name with this interface name
		*/
		&struct {
			UserService `inject`
			AppService  `inject`
		}{},
	)

	require.Nil(t, err)
	require.NotNil(t, ctx)
	require.Equal(t, 7, len(ctx.Core()))

	beans := ctx.Lookup("beans_test.Storage")
	require.Equal(t, 1, len(beans))
	storageInstance := beans[0].(*storageImpl)
	require.NotNil(t, storageInstance)
	require.Equal(t, storageInstance.Logger, logger)
	require.Equal(t, storageInstance, ctx.MustBean(StorageClass))

	beans = ctx.Lookup("beans_test.ConfigService")
	require.Equal(t, 1, len(beans))
	configServiceInstance := beans[0].(*configServiceImpl)
	require.NotNil(t, configServiceInstance)
	require.Equal(t, configServiceInstance.Storage, storageInstance)
	require.Equal(t, configServiceInstance, ctx.MustBean(ConfigServiceClass))

	beans = ctx.Lookup("beans_test.UserService")
	require.Equal(t, 1, len(beans))
	userServiceInstance := beans[0].(*userServiceImpl)
	require.NotNil(t, userServiceInstance)
	require.Equal(t, userServiceInstance.Storage, storageInstance)
	require.Equal(t, userServiceInstance.ConfigService, configServiceInstance)
	require.Equal(t, userServiceInstance, ctx.MustBean(UserServiceClass))

	beans = ctx.Lookup("beans_test.AppService")
	require.Equal(t, 1, len(beans))
	appServiceInstance := beans[0].(*appServiceImpl)
	require.NotNil(t, appServiceInstance)
	require.Equal(t, ctx, appServiceInstance.GetContext())
	require.Equal(t, appServiceInstance, ctx.MustBean(AppServiceClass))

}

type requestScope struct {
	requestParams string   // scope `runtime`
	UserService   `inject` // with `inject` tag it guarantees non-null instance
}

func (t *requestScope) routeAddUser(user string) {
	t.UserService.SaveUser(user, t.requestParams)
}

func TestRequest(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	var ctx, err = beans.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		&struct {
			UserService `inject`
		}{}, // could be used by runtime injects
	)
	require.Nil(t, err)

	controller := &requestScope{
		requestParams: "username=Bob",
	}

	err = ctx.Inject(controller)
	require.Nil(t, err)

	controller.routeAddUser("bob")

}

func TestMissingPointer(t *testing.T) {

	beans.Verbose = true

	_, err := beans.Create(
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		&struct {
			UserService `inject`
		}{}, // could be used by runtime injects
	)
	require.NotNil(t, err)
	fmt.Printf("TestMissingPointer: %v\n", err)

}

func TestMissingInterface(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	_, err := beans.Create(
		logger,
		&storageImpl{},
		&userServiceImpl{},
	)
	require.NotNil(t, err)
	fmt.Printf("TestMissingInterface: %v\n", err)

}

func TestMissingInterfaceBean(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	var ctx, err = beans.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
	)
	require.Nil(t, err)

	beans := ctx.Lookup("beans_test.UserService")

	/**
	No one is requested context_test.UserService in scan list, therefore no bean defined under this interface

	To define bean interface use this construction in scan list:
		&struct{ UserService `inject` }{}
	*/
	require.Equal(t, 0, len(beans))

	_, ok := ctx.Bean(UserServiceClass)
	require.True(t, ok)

}

func TestRequestMultithreading(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	var ctx, err = beans.Create(
		logger,
		&storageImpl{},
		&configServiceImpl{},
		&userServiceImpl{},
		&struct {
			UserService `inject`
		}{}, // could be used by runtime injects
	)
	require.Nil(t, err)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			controller := &requestScope{
				requestParams: fmt.Sprintf("firstName=Bob%d", i),
			}
			err = ctx.Inject(controller)
			require.Nil(t, err)
			username := fmt.Sprintf("user%d", i)
			controller.routeAddUser(username)
		}(i)
	}

	wg.Wait()

}
