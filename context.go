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

package beans

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
)

var Verbose bool

type context struct {

	/**
		All instances scanned during creation of context.
	    No modifications on runtime.
	*/
	core map[reflect.Type]*bean

	/**
	List of beans in initialization order that should depose on close
	*/
	disposables []*bean

	/**
	Fast search of beans by faceType and name
	*/
	registry registry

	/**
	Cache bean descriptions for Inject calls in runtime
	*/
	runtimeCache sync.Map // key is reflect.Type (classPtr), value is *beanDef

	/**
	Guarantees that context would be closed once
	*/
	destroyOnce sync.Once
}

func Create(scan ...interface{}) (Context, error) {

	beansByName := make(map[string][]*bean)
	beansByType := make(map[reflect.Type]*bean)

	core := make(map[reflect.Type]*bean)
	pointers := make(map[reflect.Type][]*injection)
	interfaces := make(map[reflect.Type][]*injection)
	factories := make(map[reflect.Type]*factory)

	ctx := &context{
		core: core,
		registry: registry{
			beansByName: beansByName,
			beansByType: beansByType,
		},
	}

	ctxBean := &bean{
		obj:      ctx,
		valuePtr: reflect.ValueOf(ctx),
		beanDef: &beanDef{
			classPtr: reflect.TypeOf(ctx),
		},
	}
	core[ctxBean.beanDef.classPtr] = ctxBean

	// scan
	for i, obj := range scan {
		if obj == nil {
			return nil, errors.Errorf("null object is not allowed on position %d", i)
		}
		classPtr := reflect.TypeOf(obj)
		var objClassPtr reflect.Type
		factoryBean, isFactoryBean := obj.(FactoryBean)
		if isFactoryBean {
			objClassPtr = factoryBean.ObjectType()
		}
		if Verbose {
			if isFactoryBean {
				var info string
				if factoryBean.Singleton() {
					info = "singleton"
				} else {
					info = "non-singleton"
				}
				fmt.Printf("FactoryBean %v produce %s %v\n", classPtr, info, objClassPtr)
			} else {
				fmt.Printf("Bean %v\n", classPtr)
			}
		}
		if classPtr.Kind() != reflect.Ptr {
			return nil, errors.Errorf("non-pointer instance is not allowed on position %d of type '%v'", i, classPtr)
		}
		if already, ok := core[classPtr]; ok {
			return nil, errors.Errorf("instance '%v' already registered, detected repeated instance on position %d of type '%v'", classPtr, i, already.beanDef.classPtr)
		}
		if isFactoryBean {
			objClassKind := objClassPtr.Kind()
			if objClassKind != reflect.Ptr && objClassKind != reflect.Interface {
				return nil, errors.Errorf("factory bean '%v' on position %d can produce ptr or interface, but object type is '%v'", classPtr, i, objClassPtr)
			}
			if already, ok := factories[objClassPtr]; ok {
				return nil, errors.Errorf("factory '%v' already registered for instance '%v', detected repeated factory on position %d of type '%v'", already.factoryClassPtr, objClassPtr, i, classPtr)
			}
		}
		/**
		Create bean
		*/
		bean, err := investigate(obj, classPtr)
		if err != nil {
			return nil, err
		}
		if len(bean.beanDef.fields) > 0 {
			value := bean.valuePtr.Elem()
			for _, injectDef := range bean.beanDef.fields {
				if Verbose {
					fmt.Printf("	Field %v\n", injectDef.fieldType)
				}
				switch injectDef.fieldType.Kind() {
				case reflect.Ptr:
					pointers[injectDef.fieldType] = append(pointers[injectDef.fieldType], &injection{bean, value, injectDef})
				case reflect.Interface:
					interfaces[injectDef.fieldType] = append(interfaces[injectDef.fieldType], &injection{bean, value, injectDef})
				default:
					return nil, errors.Errorf("injecting not a pointer or interface on field type '%v' at position %d in %v", injectDef.fieldType, i, classPtr)
				}
			}
		}
		/*
			Register bean
		*/
		core[classPtr] = bean
		if isFactoryBean {
			/*
				Register factory bean
			*/
			factories[objClassPtr] = &factory{
				bean:            bean,
				factoryObj:      obj,
				factoryClassPtr: classPtr,
				factoryBean:     factoryBean,
			}
		}
	}

	// direct match
	var found []reflect.Type
	for requiredType, injects := range pointers {
		if direct, ok := core[requiredType]; ok {

			beansByType[requiredType] = direct
			name := requiredType.String()
			beansByName[name] = append(beansByName[name], direct)

			if Verbose {
				fmt.Printf("Inject '%v' by pointer '%v' in to %+v\n", requiredType, direct.beanDef.classPtr, injects)
			}

			for _, inject := range injects {
				if err := inject.inject(direct); err != nil {
					return nil, err
				}
				// register dependency that 'inject.bean' is using 'direct' if it not lazy
				if !inject.injectionDef.lazy {
					inject.bean.dependencies = append(inject.bean.dependencies, direct)
				}
			}
			found = append(found, requiredType)

		} else if factory, ok := factories[requiredType]; ok {

			if Verbose {
				fmt.Printf("FactoryInject '%v' by pointer '%v' through factory '%v' in to %+v\n", requiredType, factory.factoryBean.ObjectType(), factory.factoryClassPtr, injects)
			}

			for _, inject := range injects {
				if inject.injectionDef.lazy {
					return nil, errors.Errorf("lazy injection is not supported for type '%v' by pointer '%v' through factory '%v' in to '%v'", requiredType, factory.factoryBean.ObjectType(), factory.factoryClassPtr, inject)
				}
				// register factory dependency for 'inject.bean' that is using 'factory'
				inject.bean.factoryDependencies = append(inject.bean.factoryDependencies,
					&factoryDependency{
						factory: factory,
						injection: func(direct *bean) error {

							beansByType[requiredType] = direct
							name := requiredType.String()
							beansByName[name] = append(beansByName[name], direct)

							return inject.inject(direct)
						},
					})
			}

			found = append(found, requiredType)

		} else {

			if Verbose {
				fmt.Printf("No found bean '%v' in context\n", requiredType)
			}

			var required []*injection
			for _, inject := range injects {
				if inject.injectionDef.optional {
					if Verbose {
						fmt.Printf("Skip optional inject '%v' in to '%v'\n", requiredType, inject)
					}
				} else {
					required = append(required, inject)
				}
			}

			if len(required) > 0 {
				return nil, errorNoCandidates(requiredType, required)
			}

		}
	}

	// interface match
	for ifaceType, injects := range interfaces {

		candidates, err := searchCandidates(ifaceType, core, factories)
		if err != nil {
			return nil, errors.Errorf("can not find candidates for '%v' interface, %v", ifaceType, err)
		}

		if len(candidates) == 0 {
			return nil, errors.Errorf("can not find implementations for '%v' interface", ifaceType)
		}

		for _, candidate := range candidates {
			if candidate.bean != nil {
				service := candidate.bean
				beansByType[ifaceType] = service
				name := ifaceType.String()
				beansByName[name] = append(beansByName[name], service)
			}
		}

		for _, inject := range injects {

			candidate, err := selectCandidate(ifaceType, candidates, inject)
			if err != nil {
				return nil, err
			}

			if candidate.bean != nil {

				service := candidate.bean
				if Verbose {
					fmt.Printf("Inject '%v' by implementation '%v' in to %+v\n", ifaceType, service.beanDef.classPtr, injects)
				}

				if err := inject.inject(service); err != nil {
					return nil, err
				}
				// register dependency that 'inject.bean' is using 'service' if not lazy
				if !inject.injectionDef.lazy {
					inject.bean.dependencies = append(inject.bean.dependencies, service)
				}

			} else if candidate.factory != nil {

				factory := candidate.factory
				if Verbose {
					fmt.Printf("FactoryInject '%v' by implementation '%v' through factory '%v' in to %+v\n", ifaceType, factory.factoryBean.ObjectType(), factory.factoryClassPtr, injects)
				}

				if inject.injectionDef.lazy {
					return nil, errors.Errorf("lazy injection is not supported for type '%v' by implementation '%v' through factory '%v' in to '%v'", ifaceType, factory.factoryBean.ObjectType(), factory.factoryClassPtr, inject)
				}
				// register factory dependency for 'inject.bean' that is using 'factory'
				inject.bean.factoryDependencies = append(inject.bean.factoryDependencies,
					&factoryDependency{
						factory: factory,
						injection: func(service *bean) error {

							beansByType[ifaceType] = service
							name := ifaceType.String()
							beansByName[name] = append(beansByName[name], service)

							return inject.inject(service)
						},
					})
			}

		}

	}

	if err := ctx.postConstruct(); err != nil {
		ctx.Close()
		return nil, err
	} else {
		return ctx, nil
	}

}

func (t *context) Extend(scan ...interface{}) (Context, error) {
	return t, nil
}

func errorNoCandidates(requiredType reflect.Type, injects []*injection) error {
	var out strings.Builder
	out.WriteString("can not find candidates for '")
	out.WriteString(requiredType.String())
	out.WriteString("' required by [")
	for i, inject := range injects {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(inject.String())
	}
	out.WriteString("]")
	return errors.New(out.String())
}

func (t *context) Core() []reflect.Type {
	var list []reflect.Type
	for typ := range t.core {
		list = append(list, typ)
	}
	return list
}

func (t *context) Bean(typ reflect.Type) (interface{}, bool) {
	if b, ok := t.getBean(typ); ok {
		return b.obj, true
	} else {
		return nil, false
	}
}

func (t *context) MustBean(typ reflect.Type) interface{} {
	if bean, ok := t.Bean(typ); ok {
		return bean
	} else {
		panic(fmt.Sprintf("bean not found %v", typ))
	}
}

func (t *context) Lookup(iface string) []interface{} {
	return t.registry.findByName(iface)
}

func (t *context) Inject(obj interface{}) error {
	if obj == nil {
		return errors.New("null obj is are not allowed")
	}
	classPtr := reflect.TypeOf(obj)
	if classPtr.Kind() != reflect.Ptr {
		return errors.Errorf("non-pointer instances are not allowed, type %v", classPtr)
	}
	valuePtr := reflect.ValueOf(obj)
	value := valuePtr.Elem()
	if bd, err := t.cache(obj, classPtr); err != nil {
		return err
	} else {
		for _, inject := range bd.fields {
			if impl, ok := t.getBean(inject.fieldType); ok {
				if err := inject.inject(&value, impl); err != nil {
					return err
				}
			} else {
				return errors.Errorf("implementation not found for field '%s' with type '%v'", inject.fieldName, inject.fieldType)
			}
		}
	}
	return nil
}

// multi-threading safe
func (t *context) getBean(ifaceType reflect.Type) (*bean, bool) {
	if b, ok := t.registry.findByType(ifaceType); ok {
		return b, true
	} else if b, ok := t.core[ifaceType]; ok {
		// pointer match with core
		t.registry.addBean(ifaceType, b)
		return b, true
	} else {
		b, err := searchByInterface(ifaceType, t.core)
		if err != nil {
			return nil, false
		}
		t.registry.addBean(ifaceType, b)
		return b, true
	}
}

// multi-threading safe
func (t *context) cache(instance interface{}, classPtr reflect.Type) (*beanDef, error) {
	if bd, ok := t.runtimeCache.Load(classPtr); ok {
		return bd.(*beanDef), nil
	} else {
		b, err := investigate(instance, classPtr)
		if err != nil {
			return nil, err
		}
		t.runtimeCache.Store(classPtr, b.beanDef)
		return b.beanDef, nil
	}
}

func getStackInfo(stack []*bean, delim string) string {
	var out strings.Builder
	n := len(stack)
	for i := 0; i < n; i++ {
		if i > 0 {
			out.WriteString(delim)
		}
		out.WriteString(stack[i].beanDef.classPtr.String())
	}
	return out.String()
}

func reverseStack(stack []*bean) []*bean {
	var out []*bean
	n := len(stack)
	for j := n - 1; j >= 0; j-- {
		out = append(out, stack[j])
	}
	return out
}

func (t *context) initalizeBean(bean *bean, stack []*bean) error {
	if bean.lifecycle == BeanInitialized {
		return nil
	}
	if bean.lifecycle == BeanConstructing {
		for i, b := range stack {
			if b == bean {
				// cycle dependency detected
				return errors.Errorf("detected cycle dependency %s", getStackInfo(append(stack[i:], bean), "->"))
			}
		}
	}
	bean.lifecycle = BeanConstructing
	defer func() {
		bean.lifecycle = BeanInitialized
	}()
	for _, factoryDep := range bean.factoryDependencies {
		if err := t.initalizeBean(factoryDep.factory.bean, append(stack, bean)); err != nil {
			return err
		}
		bean, err := factoryDep.factory.ctor()
		if err != nil {
			return errors.Errorf("factory ctor '%v' failed, %v", factoryDep.factory.factoryClassPtr, err)
		}
		err = factoryDep.injection(bean)
		if err != nil {
			return errors.Errorf("factory injection '%v' failed, %v", factoryDep.factory.factoryClassPtr, err)
		}
	}

	for _, dep := range bean.dependencies {
		if err := t.initalizeBean(dep, append(stack, bean)); err != nil {
			return err
		}
	}

	if initializer, ok := bean.obj.(InitializingBean); ok {
		if err := initializer.PostConstruct(); err != nil {
			return errors.Errorf("post construct failed %s, %v", getStackInfo(reverseStack(append(stack, bean)), " required by "), err)
		}
	}

	t.addDisposable(bean)
	return nil
}

func (t *context) addDisposable(bean *bean) {
	if _, ok := bean.obj.(DisposableBean); ok {
		t.disposables = append(t.disposables, bean)
	}
}

func (t *context) postConstruct() error {
	for _, bean := range t.core {
		if err := t.initalizeBean(bean, nil); err != nil {
			return err
		}
	}
	return nil
}

// destroy in reverse initialization order
func (t *context) Close() error {
	var err []error
	t.destroyOnce.Do(func() {
		n := len(t.disposables)
		for j := n - 1; j >= 0; j-- {
			atomic.StoreInt32(&t.disposables[j].lifecycle, BeanDestroyed)
			if dis, ok := t.disposables[j].obj.(DisposableBean); ok {
				if e := dis.Destroy(); e != nil {
					err = append(err, e)
				}
			}
		}
	})
	return multiple(err)
}

func multiple(err []error) error {
	switch len(err) {
	case 0:
		return nil
	case 1:
		return err[0]
	default:
		return errors.Errorf("multiple errors, %v", err)
	}
}

var errNotFoundInterface = errors.New("not found")

type candidate struct {
	name    string
	bean    *bean
	factory *factory
}

func (t *candidate) String() string {
	return t.name
}

func searchCandidates(ifaceType reflect.Type, core map[reflect.Type]*bean, factories map[reflect.Type]*factory) ([]*candidate, error) {
	var candidates []*candidate
	for _, bean := range core {
		if bean.beanDef.implements(ifaceType) {
			candidates = append(candidates, &candidate{name: bean.beanDef.classPtr.String(), bean: bean})
		}
	}
	for objTyp, factory := range factories {
		if objTyp.Implements(ifaceType) {
			candidates = append(candidates, &candidate{name: factory.factoryClassPtr.String(), factory: factory})
		}
	}
	return candidates, nil
}

func selectCandidate(ifaceType reflect.Type, candidates []*candidate, inject *injection) (*candidate, error) {
	if inject.injectionDef.specificBean != "" {
		name := inject.injectionDef.specificBean
		for _, candidate := range candidates {
			if candidate.name == name {
				return candidate, nil
			}
		}
		return nil, errors.Errorf("the specific implementation '%s' of interface '%v' required by injection '%v' is not found from candidates '%v'", name, ifaceType, inject, candidates)
	} else {
		switch len(candidates) {
		case 0:
			return nil, errors.Errorf("can not find implementation for '%v' interface required by injection '%v", ifaceType, inject)
		case 1:
			return candidates[0], nil
		default:
			return nil, errors.Errorf("found two or more implementation of interface '%v', candidates=%v required by injection '%v'", ifaceType, candidates, inject)
		}
	}
}

func searchByInterface(ifaceType reflect.Type, core map[reflect.Type]*bean) (*bean, error) {
	var candidates []*bean
	for _, bean := range core {
		if bean.beanDef.implements(ifaceType) {
			candidates = append(candidates, bean)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, errNotFoundInterface
	case 1:
		return candidates[0], nil
	default:
		return nil, errors.Errorf("found two or more implementation of interface '%v', candidates=%v", ifaceType, candidates)
	}
}

func searchFactoryByInterface(ifaceType reflect.Type, factories map[reflect.Type]*factory) (*factory, error) {
	var candidates []*factory
	for objTyp, factory := range factories {
		if objTyp.Implements(ifaceType) {
			candidates = append(candidates, factory)
		}
	}
	switch len(candidates) {
	case 0:
		return nil, errNotFoundInterface
	case 1:
		return candidates[0], nil
	default:
		return nil, errors.Errorf("found two or more factory beans produce interface '%v', candidates=%v", ifaceType, candidates)
	}
}
