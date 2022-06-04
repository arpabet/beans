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

package beans

import "reflect"

var ContextClass = reflect.TypeOf((*Context)(nil)).Elem()

type Context interface {
	/**
	Gets parent context if exist
	*/
	Parent() (Context, bool)

	/**
	Create new context with additional beans based on current one
	*/
	Extend(scan ...interface{}) (Context, error)

	/**
	Destroy all beans that implement interface DisposableBean.
	*/
	Close() error

	/**
	Get list of all registered instances on creation of context with scope 'core'
	*/
	Core() []reflect.Type

	/**
	Gets obj by type, that is a pointer to the structure or interface.

	Example:
		package app
		type UserService interface {
		}

		list := ctx.Bean(reflect.TypeOf((*app.UserService)(nil)).Elem())
	*/
	Bean(typ reflect.Type) []interface{}

	/**
	Lookup registered beans in context by name.
	The name is the local package plus name of the interface, for example 'app.UserService'
	Or if bean implements NamedBean interface the name of it.

	Example:
		beans := ctx.Bean("app.UserService")
		beans := ctx.Bean("userService")
	*/
	Lookup(name string) []interface{}

	/**
	Inject fields in to the obj on runtime that is not part of core context.
	Does not add a new bean in to the core context, so this method is only for one-time use with scope 'runtime'.
	Does not initialize bean and does not destroy it.

	Example:
		type requestProcessor struct {
			app.UserService  `inject`
		}

		rp := new(requestProcessor)
		ctx.Inject(rp)
		required.NotNil(t, rp.UserService)
	*/

	Inject(interface{}) error
}

/**
This interface used to provide pre-scanned instances in beans.Create method
*/
var ScannerClass = reflect.TypeOf((*Scanner)(nil)).Elem()

type Scanner interface {

	/**
	Returns pre-scanned instances
	*/
	Beans() []interface{}
}

/**
The bean object would be created after Object() function call.

ObjectType can be pointer to structure or interface.

All objects for now created are singletons, that means single instance with ObjectType in context.
*/

var FactoryBeanClass = reflect.TypeOf((*FactoryBean)(nil)).Elem()

type FactoryBean interface {

	/**
	returns an object produced by the factory, and this is the object that will be used in context, but not going to be a bean
	*/
	Object() (interface{}, error)

	/**
	returns the type of object that this FactoryBean produces
	*/
	ObjectType() reflect.Type

	/**
	denotes if the object produced by this FactoryBean is a singleton
	*/
	Singleton() bool
}

/**
Initializing bean context is using to run required method on post-construct injection stage
*/

var InitializingBeanClass = reflect.TypeOf((*InitializingBean)(nil)).Elem()

type InitializingBean interface {

	/**
	Runs this method automatically after initializing and injecting context
	*/

	PostConstruct() error
}

/**
This interface uses to select objects that could free resources after closing context
*/
var DisposableBeanClass = reflect.TypeOf((*DisposableBean)(nil)).Elem()

type DisposableBean interface {

	/**
	During close context would be called for each bean in the core.
	*/

	Destroy() error
}

/**
This interface used to collect all beans with similar type in map, where the name is the key
*/
var NamedBeanClass = reflect.TypeOf((*NamedBean)(nil)).Elem()

type NamedBean interface {

	/**
	Returns bean name
	*/
	BeanName() string
}

/**
This interface used to collect beans in list with specific order
*/
var OrderedBeanClass = reflect.TypeOf((*OrderedBean)(nil)).Elem()

type OrderedBean interface {

	/**
	Returns bean order
	*/
	BeanOrder() int
}
