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

package beans

import (
	"fmt"
	"github.com/pkg/errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync"
)

/**
Extended logs printed in console if enabled

Only for testing purposes.
 */

var Verbose bool

type context struct {

	/**
	Parent context if exist
	*/
	parent *context

	/**
		All instances scanned during creation of context.
	    No modifications on runtime allowed.
	*/
	core map[reflect.Type][]*bean

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

	core := make(map[reflect.Type][]*bean)
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
	err := forEach("", scan, func(pos string, obj interface{}) (err error) {

		classPtr := reflect.TypeOf(obj)

		defer func() {
			if r := recover(); r != nil {
				err = errors.Errorf("recover from object scan '%s' on error %v\n", classPtr.String(), r)
			}
		}()

		switch classPtr.Kind() {
		case reflect.Ptr:
			/**
			Create bean from object
			*/
			objBean, err := investigate(obj, classPtr)
			if err != nil {
				return err
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
					objectName := factoryBean.ObjectName()
					if objectName != "" {
						fmt.Printf("FactoryBean %v produce %s %v with name '%s'\n", classPtr, info, elemClassPtr, objectName)
					} else {
						fmt.Printf("FactoryBean %v produce %s %v\n", classPtr, info, elemClassPtr)
					}
				} else {
					if objBean.qualifier != "" {
						fmt.Printf("Bean %v with name '%s'\n", classPtr, objBean.qualifier)
					} else {
						fmt.Printf("Bean %v\n", classPtr)
					}
				}
			}

			if isFactoryBean {
				elemClassKind := elemClassPtr.Kind()
				if elemClassKind != reflect.Ptr && elemClassKind != reflect.Interface {
					return errors.Errorf("factory bean '%v' on position '%s' can produce ptr or interface, but object type is '%v'", classPtr, pos, elemClassPtr)
				}
			}

			if len(objBean.beanDef.fields) > 0 {
				value := objBean.valuePtr.Elem()
				for _, injectDef := range objBean.beanDef.fields {
					if Verbose {
						var attr []string
						if injectDef.lazy {
							attr = append(attr,  "lazy")
						}
						if injectDef.optional {
							attr = append(attr,  "optional")
						}
						if injectDef.qualifier != "" {
							attr = append(attr,  "bean=" + injectDef.qualifier)
						}
						var attrs string
						if len(attr) > 0 {
							attrs = fmt.Sprintf("[%s]", strings.Join(attr, ","))
						}
						var prefix string
						if injectDef.slice {
							prefix = "[]"
						}
						if injectDef.table {
							prefix = "map[string]"
						}
						fmt.Printf("	Field %s%v %s\n", prefix, injectDef.fieldType, attrs)
					}
					switch injectDef.fieldType.Kind() {
					case reflect.Ptr:
						pointers[injectDef.fieldType] = append(pointers[injectDef.fieldType], &injection{objBean, value, injectDef})
					case reflect.Interface:
						interfaces[injectDef.fieldType] = append(interfaces[injectDef.fieldType], &injection{objBean, value, injectDef})
					case reflect.Func:
						pointers[injectDef.fieldType] = append(pointers[injectDef.fieldType], &injection{objBean, value, injectDef})
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
				objectName := factoryBean.ObjectName()
				if objectName == "" {
					objectName = elemClassPtr.String()
				}
				elemBean := &bean{
					name:        objectName,
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
		case reflect.Func:

			if Verbose {
				fmt.Printf("Function %v\n", classPtr)
			}

			/*
				Register function in context
			*/
			registerBean(core, classPtr, &bean{
				name:     classPtr.String(),
				obj:      obj,
				valuePtr: reflect.ValueOf(obj),
				beanDef: &beanDef{
					classPtr: classPtr,
				},
				lifecycle: BeanCreated,
			})
		default:
			return errors.Errorf("instance could be a pointer or function, but was '%s' on position '%s' of type '%v'", classPtr.Kind().String(), pos, classPtr)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// direct match
	for requiredType, injects := range pointers {

		direct := ctx.findDirectRecursive(requiredType)
		if len(direct) > 0 {

			ctx.registry.addBeanList(requiredType, direct)

			if Verbose {
				fmt.Printf("Inject '%v' by pointer '%+v' in to %+v\n", requiredType, direct, injects)
			}

			list := orderBeans(direct)
			for _, inject := range injects {
				if err := inject.inject(list); err != nil {
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

		candidates := ctx.searchCandidatesRecursive(ifaceType)
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

		ctx.registry.addBeanList(ifaceType, candidates)

		list := orderBeans(candidates)
		for _, inject := range injects {

			if Verbose {
				fmt.Printf("Inject '%v' by implementation '%+v' in to %+v\n", ifaceType, list, inject)
			}

			if err := inject.inject(list); err != nil {
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


func (t *context) findDirectRecursive(requiredType reflect.Type) []*bean {
	var candidates []*bean
	for ctx := t; ctx != nil; ctx = ctx.parent {
		if direct, ok := ctx.core[requiredType]; ok {
			candidates = append(candidates, direct...)
		}
	}
	return candidates
}

func registerBean(registry map[reflect.Type][]*bean, classPtr reflect.Type, bean *bean) {
	if list, ok := registry[classPtr]; ok {
		registry[classPtr] = append(list, bean)
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
			continue
		}
		switch obj := item.(type) {
		case Scanner:
			if err := forEach(pos, obj.Beans(), cb); err != nil {
				return err
			}
		case []interface{}:
			if err := forEach(pos, obj, cb); err != nil {
				return err
			}
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

func (t *context) Bean(typ reflect.Type) []Bean {
	var beanList []Bean
	for _, b := range t.getBean(typ) {
		beanList = append(beanList, b)
	}
	return beanList
}

func (t *context) Lookup(iface string) []Bean {
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
			impl := t.getBean(inject.fieldType)
			if len(impl) > 0 {
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
func (t *context) getBean(ifaceType reflect.Type) []*bean {

	// search in registry containing all parents
	if b, ok := t.registry.findByType(ifaceType); ok {
		return b
	}

	// unknown entity request, le't search and cache it
	switch ifaceType.Kind() {
	case reflect.Ptr, reflect.Func:
		direct := t.findDirectRecursive(ifaceType)
		if len(direct) > 0 {
			t.registry.addBeanList(ifaceType, direct)
		}
		return direct
	case reflect.Interface:
		candidates := t.searchCandidatesRecursive(ifaceType)
		if len(candidates) > 0 {
			t.registry.addBeanList(ifaceType, candidates)
		}
		return candidates
	default:
		return nil
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

func (t *context) constructBeanList(list []*bean, stack []*bean) error {
	for _, bean := range list {
		if err := t.constructBean(bean, stack); err != nil {
			return err
		}
	}
	return nil
}

func indent(n int) string {
	if n == 0 {
		return ""
	}
	var out []byte
	for i := 0; i < n; i++ {
		out = append(out, ' ', ' ')
	}
	return string(out)
}

func (t *context) constructBean(bean *bean, stack []*bean) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("construct bean '%s' with type '%v' recovered with error %v", bean.name, bean.beanDef.classPtr, r)
		}
	}()

	if bean.lifecycle == BeanInitialized {
		return nil
	}

	_, isFactoryBean := bean.obj.(FactoryBean)
	initializer, hasConstructor := bean.obj.(InitializingBean)
	if Verbose {
		fmt.Printf("%sConstruct Bean '%s' with type '%v', isFactoryBean=%v, hasFactory=%v, hasObject=%v, hasConstructor=%v\n", indent(len(stack)), bean.name, bean.beanDef.classPtr, isFactoryBean, bean.beenFactory != nil, bean.obj != nil, hasConstructor)
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
	bean.ctorMu.Lock()
	defer func() {
		bean.ctorMu.Unlock()
	}()

	for _, factoryDep := range bean.factoryDependencies {
		if err := t.constructBean(factoryDep.factory.bean, append(stack, bean)); err != nil {
			return err
		}
		if Verbose {
			fmt.Printf("%sFactoryDep (%v).Object()\n", indent(len(stack)+1), factoryDep.factory.factoryClassPtr)
		}
		bean, created, err := factoryDep.factory.ctor()
		if err != nil {
			return errors.Errorf("factory ctor '%v' failed, %v", factoryDep.factory.factoryClassPtr, err)
		}
		if created {
			if Verbose {
				fmt.Printf("%sDep Created Bean %s with type '%v'\n", indent(len(stack)+1), bean.name, bean.beanDef.classPtr)
			}
			t.registry.addBean(factoryDep.factory.factoryBean.ObjectType(), bean)
		}
		err = factoryDep.injection(bean)
		if err != nil {
			return errors.Errorf("factory injection '%v' failed, %v", factoryDep.factory.factoryClassPtr, err)
		}
	}

	if isFactoryBean || hasConstructor {
		for _, dep := range bean.dependencies {
			if err := t.constructBeanList(dep, append(stack, bean)); err != nil {
				return err
			}
		}
	}

	// check if it is empty element bean
	if bean.beenFactory != nil && bean.obj == nil {
		if err := t.constructBean(bean.beenFactory.bean, append(stack, bean)); err != nil {
			return err
		}
		if Verbose {
			fmt.Printf("%s(%v).Object()\n", indent(len(stack)), bean.beenFactory.factoryClassPtr)
		}
		_, _, err := bean.beenFactory.ctor() // always new
		if err != nil {
			return errors.Errorf("factory ctor '%v' failed, %v", bean.beenFactory.factoryClassPtr, err)
		}
		if bean.obj == nil {
			return errors.Errorf("bean '%v' was not created by factory ctor '%v'", bean, bean.beenFactory.factoryClassPtr)
		}
		return nil
	}

	if hasConstructor {
		if Verbose {
			fmt.Printf("%sPostConstruct Bean '%s' with type '%v'\n", indent(len(stack)), bean.name, bean.beanDef.classPtr)
		}
		if err := initializer.PostConstruct(); err != nil {
			return errors.Errorf("post construct failed %s, %v", getStackInfo(reverseStack(append(stack, bean)), " required by "), err)
		}
	}

	t.addDisposable(bean)
	bean.lifecycle = BeanInitialized
	return nil
}

func (t *context) addDisposable(bean *bean) {
	if _, ok := bean.obj.(DisposableBean); ok {
		t.disposables = append(t.disposables, bean)
	}
}

func (t *context) postConstruct() (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("post construct recover on error, %v\n", r)
		}
	}()

	for _, list := range t.core {
		if err = t.constructBeanList(list, nil); err != nil {
			return err
		}
	}
	return nil
}

// destroy in reverse initialization order
func (t *context) Close() (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("context close recover error: %v", r)
		}
	}()

	var listErr []error
	t.destroyOnce.Do(func() {
		n := len(t.disposables)
		for j := n - 1; j >= 0; j-- {
			t.destroyBean(t.disposables[j])
		}
	})
	return multipleErr(listErr)
}

func (t *context) destroyBean(b *bean) (err error) {

	defer func() {
		if r := recover(); r != nil {
			err = errors.Errorf("destroy bean '%s' with type '%v' recovered with error: %v", b.name, b.beanDef.classPtr, r)
		}
	}()

	if b.lifecycle != BeanInitialized {
		return nil
	}

	b.lifecycle = BeanDestroying
	if Verbose {
		fmt.Printf("Destroy bean '%s' with type '%v'\n", b.name, b.beanDef.classPtr)
	}
	if dis, ok := b.obj.(DisposableBean); ok {
		if e := dis.Destroy(); e != nil {
			err = e
		} else {
			b.lifecycle = BeanDestroyed
		}
	}
	return
}

func multipleErr(err []error) error {
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

func (t *context) searchCandidatesRecursive(ifaceType reflect.Type) []*bean {
	var candidates []*bean
	for ctx := t; ctx != nil; ctx = ctx.parent {
		list := ctx.searchCandidates(ifaceType)
		if len(list) > 0 {
			candidates = append(candidates, list...)
		}
	}
	return candidates
}

func (t *context) searchCandidates(ifaceType reflect.Type) []*bean {
	var candidates []*bean
	for _, list := range t.core {
		if len(list) > 0 && list[0].beanDef.implements(ifaceType) {
			candidates = append(candidates, list...)
		}
	}
	return candidates
}

func orderBeans(candidates []*bean) []*bean {
	var ordered []*bean
	for _, candidate := range candidates {
		if candidate.ordered {
			ordered = append(ordered, candidate)
		}
	}
	n := len(ordered)
	if n > 0 {
		sort.Slice(ordered, func(i, j int) bool {
			return ordered[i].order < ordered[j].order
		})
		if n != len(candidates) {
			var unordered []*bean
			for _, candidate := range candidates {
				if !candidate.ordered {
					unordered = append(unordered, candidate)
				}
			}
			return append(ordered, unordered...)
		}
		return ordered
	} else {
		return candidates
	}
}

func searchByInterface(ifaceType reflect.Type, core map[reflect.Type][]*bean) ([]*bean, error) {
	var candidates [][]*bean
	for _, list := range core {
		if len(list) > 0 && list[0].beanDef.implements(ifaceType) {
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

func (t *context) String() string {
	return fmt.Sprintf("Context [hasParent=%v, types=%d, destructors=%d]", t.parent != nil, len(t.core), len(t.disposables))
}
