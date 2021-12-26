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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

var Verbose bool
var Recover bool

type context struct {

	/**
	Parent context if exist
	*/
	parent *context

	/**
		All instances scanned during creation of context.
	    No modifications on runtime allowed.
	*/
	core map[reflect.Type]*beanlist

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
	return createContext(nil, scan)
}

func (t *context) Extend(scan ...interface{}) (Context, error) {
	return createContext(t, scan)
}

func (t *context) Parent() (Context, bool) {
	if t.parent != nil {
		return t.parent, true
	} else {
		return nil, false
	}
}

func createContext(parent *context, scan []interface{}) (Context, error) {

	core := make(map[reflect.Type]*beanlist)
	pointers := make(map[reflect.Type][]*injection)
	interfaces := make(map[reflect.Type][]*injection)

	ctx := &context{
		parent: parent,
		core:   core,
		registry: registry{
			beansByName: make(map[string][]*bean),
			beansByType: make(map[reflect.Type][]*bean),
		},
	}

	ctxBean := &bean{
		obj:      ctx,
		valuePtr: reflect.ValueOf(ctx),
		beanDef: &beanDef{
			classPtr: reflect.TypeOf(ctx),
		},
		lifecycle: BeanInitialized,
	}
	core[ctxBean.beanDef.classPtr] = oneBean(ctxBean)

	// scan
	err := forEach("", scan, func(pos string, obj interface{}) error {

		classPtr := reflect.TypeOf(obj)

		if Recover {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Recover from object '%s' scan error %v\n", classPtr.String(), r)
				}
			}()
		}

		var elemClassPtr reflect.Type
		factoryBean, isFactoryBean := obj.(FactoryBean)
		if isFactoryBean {
			elemClassPtr = factoryBean.ObjectType()
		}

		if Verbose {
			if isFactoryBean {
				var info string
				if factoryBean.Singleton() {
					info = "singleton"
				} else {
					info = "non-singleton"
				}
				fmt.Printf("FactoryBean %v produce %s %v\n", classPtr, info, elemClassPtr)
			} else {
				fmt.Printf("Bean %v\n", classPtr)
			}
		}

		if isFactoryBean {
			elemClassKind := elemClassPtr.Kind()
			if elemClassKind != reflect.Ptr && elemClassKind != reflect.Interface {
				return errors.Errorf("factory bean '%v' on position '%s' can produce ptr or interface, but object type is '%v'", classPtr, pos, elemClassPtr)
			}
		}

		if classPtr.Kind() != reflect.Ptr {
			return errors.Errorf("non-pointer instance is not allowed on position '%s' of type '%v'", pos, classPtr)
		}

		/**
		Create bean from object
		*/
		objBean, err := investigate(obj, classPtr)
		if err != nil {
			return err
		}
		if len(objBean.beanDef.fields) > 0 {
			value := objBean.valuePtr.Elem()
			for _, injectDef := range objBean.beanDef.fields {
				if Verbose {
					fmt.Printf("	Field %v\n", injectDef.fieldType)
				}
				switch injectDef.fieldType.Kind() {
				case reflect.Ptr:
					pointers[injectDef.fieldType] = append(pointers[injectDef.fieldType], &injection{objBean, value, injectDef})
				case reflect.Interface:
					interfaces[injectDef.fieldType] = append(interfaces[injectDef.fieldType], &injection{objBean, value, injectDef})
				default:
					return errors.Errorf("injecting not a pointer or interface on field type '%v' at position '%s' in %v", injectDef.fieldType, pos, classPtr)
				}
			}
		}

		/*
			Register factory if needed
		*/
		if isFactoryBean {
			f := &factory{
				bean:            objBean,
				factoryObj:      obj,
				factoryClassPtr: classPtr,
				factoryBean:     factoryBean,
			}
			elemBean := &bean{
				name:        elemClassPtr.String(),
				beenFactory: f,
				beanDef: &beanDef{
					classPtr: elemClassPtr,
				},
				lifecycle: BeanAllocated,
			}
			f.instances = oneBean(elemBean)
			// we can have singleton or multiple beans in context produced by this factory, let's allocate reference for injections even if those beans are still not exist
			registerBean(core, elemClassPtr, elemBean)
		}

		/*
			Register bean itself
		*/
		registerBean(core, classPtr, objBean)
		return nil
	})

	if err != nil {
		return nil, err
	}

	// direct match
	for requiredType, injects := range pointers {
		if direct, ok := core[requiredType]; ok {

			ctx.registry.addBeanList(requiredType, direct)

			if Verbose {
				fmt.Printf("Inject '%v' by pointer '%+v' in to %+v\n", requiredType, direct, injects)
			}

			for _, inject := range injects {
				if err := inject.inject(direct); err != nil {
					return nil, errors.Errorf("required type '%s' injection error, %v", requiredType, err)
				}
			}

		} else {

			if Verbose {
				fmt.Printf("Bean '%v' not found in context\n", requiredType)
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
				return nil, errors.Errorf("can not find candidates for '%v' reference bean required by '%+v'", requiredType, required)
			}

		}
	}

	// interface match
	for ifaceType, injects := range interfaces {

		candidates := searchCandidates(ifaceType, core)
		if len(candidates) == 0 {

			if Verbose {
				fmt.Printf("No found bean candidates for interface '%v' in context\n", ifaceType)
			}

			var required []*injection
			for _, inject := range injects {
				if inject.injectionDef.optional {
					if Verbose {
						fmt.Printf("Skip optional inject of interface '%v' in to '%v'\n", ifaceType, inject)
					}
				} else {
					required = append(required, inject)
				}
			}

			if len(required) > 0 {
				return nil, errors.Errorf("can not find candidates for '%v' interface required by '%+v'", ifaceType, required)
			}

			continue
		}

		for _, candidate := range candidates {
			ctx.registry.addBeanList(ifaceType, candidate)
		}

		for _, inject := range injects {

			candidate, err := selectCandidate(ifaceType, candidates, inject)
			if err != nil {
				return nil, err
			}

			if Verbose {
				fmt.Printf("Inject '%v' by implementation '%+v' in to %+v\n", ifaceType, candidate, inject)
			}

			if err := inject.inject(candidate); err != nil {
				return nil, errors.Errorf("interface '%s' injection error, %v", ifaceType, err)
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

func registerBean(registry map[reflect.Type]*beanlist, classPtr reflect.Type, bean *bean) {
	if list, ok := registry[classPtr]; ok {
		list.append(bean)
	} else {
		registry[classPtr] = oneBean(bean)
	}
}

func forEach(initialPos string, scan []interface{}, cb func(i string, obj interface{}) error) error {
	for j, item := range scan {
		var pos string
		if len(initialPos) > 0 {
			pos = fmt.Sprintf("%s.%d", initialPos, j)
		} else {
			pos = strconv.Itoa(j)
		}
		if item == nil {
			return errors.Errorf("null object is not allowed on position '%s'", pos)
		}
		switch obj := item.(type) {
		case []interface{}:
			return forEach(pos, obj, cb)
		case interface{}:
			if err := cb(pos, obj); err != nil {
				return errors.Errorf("object '%v' error, %v", reflect.ValueOf(item).Type(), err)
			}
		default:
			return errors.Errorf("unknown object type '%v' on position '%s'", reflect.ValueOf(item).Type(), pos)
		}
	}
	return nil
}

func (t *context) Core() []reflect.Type {
	var list []reflect.Type
	for typ := range t.core {
		list = append(list, typ)
	}
	return list
}

func (t *context) Bean(typ reflect.Type) []interface{} {
	var obj []interface{}
	if list, ok := t.getBean(typ); ok {
		for _, b := range list {
			obj = append(obj, b.obj)
		}
	}
	return obj
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
func (t *context) getBean(ifaceType reflect.Type) ([]*bean, bool) {
	if b, ok := t.registry.findByType(ifaceType); ok {
		return b, true
	} else if b, ok := t.core[ifaceType]; ok {
		// pointer match with core
		t.registry.addBeanList(ifaceType, b)
		return b.list(), true
	} else {
		b, err := searchByInterface(ifaceType, t.core)
		if err != nil {
			return nil, false
		}
		t.registry.addBeanList(ifaceType, b)
		return b.list(), true
	}
}

// multi-threading safe
func (t *context) cache(obj interface{}, classPtr reflect.Type) (*beanDef, error) {
	if bd, ok := t.runtimeCache.Load(classPtr); ok {
		return bd.(*beanDef), nil
	} else {
		b, err := investigate(obj, classPtr)
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

func (t *context) constructBeanList(list *beanlist, stack []*bean) error {
	for bean := list.head; bean != nil; bean = bean.next {
		if err := t.constructBean(bean, stack); err != nil {
			return err
		}
		if bean == list.tail {
			break
		}
	}
	return nil
}

func (t *context) constructBean(bean *bean, stack []*bean) error {

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

	if bean.beenFactory != nil && bean.obj == nil {
		if err := t.constructBean(bean.beenFactory.bean, append(stack, bean)); err != nil {
			return err
		}
		_, _, err := bean.beenFactory.ctor()
		if err != nil {
			return errors.Errorf("factory ctor '%v' failed, %v", bean.beenFactory.factoryClassPtr, err)
		}
		if bean.obj == nil {
			return errors.Errorf("bean '%v' was not created by factory ctor '%v'", bean, bean.beenFactory.factoryClassPtr)
		}
		return nil
	}

	for _, factoryDep := range bean.factoryDependencies {
		if err := t.constructBean(factoryDep.factory.bean, append(stack, bean)); err != nil {
			return err
		}
		bean, created, err := factoryDep.factory.ctor()
		if err != nil {
			return errors.Errorf("factory ctor '%v' failed, %v", factoryDep.factory.factoryClassPtr, err)
		}
		if created {
			t.registry.addBeanByName(bean)
		}
		err = factoryDep.injection(bean)
		if err != nil {
			return errors.Errorf("factory injection '%v' failed, %v", factoryDep.factory.factoryClassPtr, err)
		}
	}

	for _, dep := range bean.dependencies {
		if err := t.constructBeanList(dep, append(stack, bean)); err != nil {
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
	for _, list := range t.core {
		if err := t.constructBeanList(list, nil); err != nil {
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

func searchCandidates(ifaceType reflect.Type, core map[reflect.Type]*beanlist) []*beanlist {
	var candidates []*beanlist
	for _, list := range core {
		if list.head != nil && list.head.beanDef.implements(ifaceType) {
			candidates = append(candidates, list)
		}
	}
	return candidates
}

func selectCandidate(ifaceType reflect.Type, candidates []*beanlist, inject *injection) (*beanlist, error) {
	if inject.injectionDef.specificBean != "" {
		name := inject.injectionDef.specificBean
		for _, candidate := range candidates {
			if candidate.hasName(name) {
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
			return nil, errors.Errorf("found two or more bean lists that implements interface '%v', candidates=%v required by injection '%v'", ifaceType, candidates, inject)
		}
	}
}

func searchByInterface(ifaceType reflect.Type, core map[reflect.Type]*beanlist) (*beanlist, error) {
	var candidates []*beanlist
	for _, list := range core {
		if list.head.beanDef.implements(ifaceType) {
			candidates = append(candidates, list)
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
