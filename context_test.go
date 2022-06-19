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

	// skip all nil beans
	ctx, err := beans.Create(nil)

	require.NoError(t, err)
	require.NotNil(t, ctx)
	defer ctx.Close()
}

func TestCreateNilArray(t *testing.T) {

	// skip all nil beans
	ctx, err := beans.Create([]interface{}{nil, nil})

	require.NoError(t, err)
	require.NotNil(t, ctx)
	defer ctx.Close()
}

func TestCreateEmpty(t *testing.T) {

	ctx, err := beans.Create()
	require.NoError(t, err)
	require.NotNil(t, ctx)
	defer ctx.Close()

	require.Equal(t, 1, len(ctx.Core()))

	c := ctx.Bean(beans.ContextClass)
	require.Equal(t, 1, len(c))
	require.Equal(t, ctx, c[0].Object())

}

var StorageClass = reflect.TypeOf((*Storage)(nil)).Elem()

type Storage interface {
	beans.NamedBean
	Load(key string) string
	Store(key, value string)
}

var ConfigServiceClass = reflect.TypeOf((*ConfigService)(nil)).Elem()

type ConfigService interface {
	beans.NamedBean
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

func (t *storageImpl) BeanName() string {
	return "storage"
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
	Storage Storage `inject`
}

func (t *configServiceImpl) BeanName() string {
	return "configService"
}

func (t *configServiceImpl) GetConfig(key string) string {
	return t.Storage.Load("config:" + key)
}

func (t *configServiceImpl) SetConfig(key, value string) {
	t.Storage.Store("config:"+key, value)
}

type userServiceImpl struct {
	Storage       Storage       `inject`
	ConfigService ConfigService `inject`
}

func (t *userServiceImpl) GetUser(user string) string {
	return t.Storage.Load("user:" + user)
}

func (t *userServiceImpl) SaveUser(user, details string) {
	if t.allowWrites() {
		t.Storage.Store("user:"+user, details)
	}
}

func (t *userServiceImpl) allowWrites() bool {
	b, err := strconv.ParseBool(t.ConfigService.GetConfig("allowWrites"))
	if err != nil {
		return false
	}
	return b
}

func (t *userServiceImpl) PostConstruct() error {
	t.ConfigService.SetConfig("allowWrites", "true")
	return nil
}

type appServiceImpl struct {
	Context beans.Context `inject`
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
			Storage Storage `inject`
		}{},
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	println(err.Error())
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
			Storage Storage `inject`
		}{},
	)

	require.NotNil(t, err)
	require.Nil(t, ctx)
	require.True(t, strings.Contains(err.Error(), "multiple candidates"))
	require.True(t, strings.Contains(err.Error(), "beans_test.Storage"))
	println(err.Error())
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
			UserService UserService `inject`
			AppService  AppService  `inject`
		}{},
	)

	require.NoError(t, err)
	require.NotNil(t, ctx)
	defer ctx.Close()

	require.Equal(t, 7, len(ctx.Core()))

	beans := ctx.Lookup("storage")
	require.Equal(t, 1, len(beans))
	storageInstance := beans[0].Object().(*storageImpl)
	require.NotNil(t, storageInstance)
	require.Equal(t, storageInstance.Logger, logger)
	require.Equal(t, storageInstance, ctx.Bean(StorageClass)[0].Object())

	beans = ctx.Lookup("configService")
	require.Equal(t, 1, len(beans))
	configServiceInstance := beans[0].Object().(*configServiceImpl)
	require.NotNil(t, configServiceInstance)
	require.Equal(t, configServiceInstance.Storage, storageInstance)
	require.Equal(t, configServiceInstance, ctx.Bean(ConfigServiceClass)[0].Object())

	beans = ctx.Lookup("*beans_test.userServiceImpl")
	require.Equal(t, 1, len(beans))
	userServiceInstance := beans[0].Object().(*userServiceImpl)
	require.NotNil(t, userServiceInstance)
	require.Equal(t, userServiceInstance.Storage, storageInstance)
	require.Equal(t, userServiceInstance.ConfigService, configServiceInstance)
	require.Equal(t, userServiceInstance, ctx.Bean(UserServiceClass)[0].Object())

	beans = ctx.Lookup("*beans_test.appServiceImpl")
	require.Equal(t, 1, len(beans))
	appServiceInstance := beans[0].Object().(*appServiceImpl)
	require.NotNil(t, appServiceInstance)
	require.Equal(t, ctx, appServiceInstance.GetContext())
	require.Equal(t, appServiceInstance, ctx.Bean(AppServiceClass)[0].Object())

}

func TestCreateArray(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	var b []interface{}
	b = append(b, logger, &storageImpl{}, &configServiceImpl{})

	ctx, err := beans.Create(
		b,
		&userServiceImpl{},
		&appServiceImpl{},
	)

	require.NoError(t, err)
	require.NotNil(t, ctx)
	defer ctx.Close()

	require.Equal(t, 6, len(ctx.Core()))

}

type scannerImpl struct {
	arr []interface{}
}

// implements beans.Scanner
func (t scannerImpl) Beans() []interface{} {
	return t.arr
}

func TestCreateScanner(t *testing.T) {

	beans.Verbose = true
	logger := log.New(os.Stderr, "beans: ", log.LstdFlags)

	scanner := scannerImpl{
		arr: []interface{}{logger, &storageImpl{}, &configServiceImpl{}},
	}

	ctx, err := beans.Create(
		scanner,
		&userServiceImpl{},
		&appServiceImpl{},
	)

	require.NoError(t, err)
	require.NotNil(t, ctx)
	defer ctx.Close()

	require.Equal(t, 6, len(ctx.Core()))

}

type requestScope struct {
	requestParams string      // scope `runtime`
	UserService   UserService `inject` // with `inject` tag it guarantees non-null instance
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
			UserService UserService `inject`
		}{}, // could be used by runtime injects
	)
	require.NoError(t, err)
	defer ctx.Close()

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
			UserService UserService `inject`
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
	require.NoError(t, err)
	defer ctx.Close()

	beans := ctx.Lookup("beans_test.UserService")

	/**
	No one is requested context_test.UserService in scan list, therefore no bean defined under this interface

	To define bean interface use this construction in scan list:
		&struct{ UserService `inject` }{}
	*/
	require.Equal(t, 0, len(beans))

	b := ctx.Bean(UserServiceClass)
	require.Equal(t, 1, len(b))

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
			UserService UserService `inject`
		}{}, // could be used by runtime injects
	)
	require.NoError(t, err)
	defer ctx.Close()

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
