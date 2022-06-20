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
	"strings"
	"sync"
)

type beanDef struct {
	/**
	Class of the pointer to the struct or interface
	*/
	classPtr reflect.Type

	/**
	Anonymous fields expose their interfaces though the bean itself.
	This is confusing on injection, because this bean is an encapsulator, not an implementation.

	Skip those fields.
	*/
	anonymousFields []reflect.Type

	/**
	Fields that are going to be injected
	*/
	fields []*injectionDef
}

type bean struct {
	/**
	Name of the bean
	*/
	name string

	/**
	Qualifier of the bean
	 */
	qualifier string

	/**
	Order of the bean
	*/
	ordered bool
	order   int

	/**
	Factory of the bean if exist
	*/
	beenFactory *factory

	/**
	Instance to the bean, could be empty if beenFactory exist
	*/
	obj interface{}

	/**
	Reflect instance to the pointer or interface of the bean, could be empty if beenFactory exist
	*/
	valuePtr reflect.Value

	/**
	Bean description
	*/
	beanDef *beanDef

	/**
	Bean lifecycle
	*/
	lifecycle BeanLifecycle

	/**
	List of beans that should initialize before current bean
	*/
	dependencies []*beanlist

	/**
	List of factory beans that should initialize before current bean
	*/
	factoryDependencies []*factoryDependency

	/**
	Next bean in the list
	*/
	next *bean

	/**
	Constructor mutex for the bean
	*/
	ctorMu sync.Mutex
}

func (t *bean) String() string {
	if t.beenFactory != nil {
		objectName := t.beenFactory.factoryBean.ObjectName()
		if objectName != "" {
			return fmt.Sprintf("<FactoryBean %s->%s(%s)>", t.beenFactory.factoryClassPtr, t.beanDef.classPtr, objectName)
		} else {
			return fmt.Sprintf("<FactoryBean %s->%s>", t.beenFactory.factoryClassPtr, t.beanDef.classPtr)
		}
	} else if t.qualifier != "" {
		return fmt.Sprintf("<Bean %s(%s)>", t.beanDef.classPtr, t.qualifier)
	} else {
		return fmt.Sprintf("<Bean %s>", t.beanDef.classPtr)
	}
}

func (t *bean) Name() string {
	return t.name
}

func (t *bean) Class() reflect.Type {
	return t.beanDef.classPtr
}

func (t *bean) Implements(ifaceType reflect.Type) bool {
	return t.beanDef.implements(ifaceType)
}

func (t *bean) Object() interface{} {
	return t.obj
}

func (t *bean) FactoryBean() (Bean, bool) {
	if t.beenFactory != nil {
		return t.beenFactory.bean, true
	} else {
		return nil, false
	}
}

func (t *bean) Reload() error {
	t.ctorMu.Lock()
	defer t.ctorMu.Unlock()

	t.lifecycle = BeanDestroying
	if dis, ok := t.obj.(DisposableBean); ok {
		if err := dis.Destroy(); err != nil {
			return err
		}
	}
	t.lifecycle = BeanConstructing
	if t.beenFactory != nil {
		return errors.Errorf("bean '%s' was created by factory bean '%v and can not be reloaded", t.name, t.beenFactory.factoryClassPtr)
	} else {
		if init, ok := t.obj.(InitializingBean); ok {
			if err := init.PostConstruct(); err != nil {
				return err
			}
		}
	}
	t.lifecycle = BeanInitialized
	return nil
}

func (t *bean) Lifecycle() BeanLifecycle {
	return t.lifecycle
}

/**
Check if bean definition can implement interface type
*/
func (t *beanDef) implements(ifaceType reflect.Type) bool {
	if isSomeoneImplements(ifaceType, t.anonymousFields) {
		return false
	}
	return t.classPtr.Implements(ifaceType)
}

type factory struct {
	/**
	Bean associated with Factory in context
	*/
	bean *bean

	/**
	Instance to the factory bean
	*/
	factoryObj interface{}

	/**
	Factory bean type
	*/
	factoryClassPtr reflect.Type

	/**
	Factory bean interface
	*/
	factoryBean FactoryBean

	/**
	Created bean instances by this factory
	*/
	instances *beanlist
}

type beanlist struct {
	head *bean
	tail *bean
}

func oneBean(bean *bean) *beanlist {
	return &beanlist{
		head: bean,
		tail: bean,
	}
}

func (t *beanlist) append(bean *bean) {
	if t.tail == nil {
		t.head, t.tail = bean, bean
	} else {
		t.tail.next = bean
		t.tail = bean
	}
}

func (t *beanlist) single() bool {
	return t.head == t.tail
}

func (t *beanlist) list() []*bean {
	var list []*bean
	for b := t.head; b != nil; b = b.next {
		list = append(list, b)
		if b == t.tail {
			break
		}
	}
	return list
}

func (t *beanlist) forEach(cb func(*bean)) {
	for b := t.head; b != nil; b = b.next {
		cb(b)
		if b == t.tail {
			break
		}
	}
}

func (t *beanlist) String() string {
	if t.head != nil {
		return t.head.String()
	}
	return ""
}

func (t *factory) String() string {
	return t.factoryClassPtr.String()
}

func (t *factory) ctor() (*bean, bool, error) {
	var b *bean
	if t.factoryBean.Singleton() {
		if t.instances.head.obj == nil {
			b = t.instances.head
		} else {
			return t.instances.head, false, nil
		}
	} else {
		if t.instances.head.obj == nil {
			b = t.instances.head
		} else {
			b = &bean{
				name:        t.instances.head.beanDef.classPtr.String(),
				beenFactory: t.instances.head.beenFactory,
				beanDef:     t.instances.head.beanDef,
			}
			t.instances.tail.next = b
			t.instances.tail = b
		}
	}

	obj, err := t.factoryBean.Object()
	if err != nil {
		return nil, false, errors.Errorf("factory bean '%v' failed to create bean '%v', %v", t.factoryClassPtr, t.factoryBean.ObjectType(), err)
	}

	b.obj = obj
	b.lifecycle = BeanInitialized
	if namedBean, ok := obj.(NamedBean); ok {
		b.name = namedBean.BeanName()
	}
	b.valuePtr = reflect.ValueOf(obj)

	return b, true, nil
}

type factoryDependency struct {

	/*
		Reference on factory bean used to produce instance
	*/

	factory *factory

	/*
		Injection function where we need to inject produced instance
	*/
	injection func(instance *bean) error
}

/**
Investigate bean by using reflection
*/
func investigate(obj interface{}, classPtr reflect.Type) (*bean, error) {
	var fields []*injectionDef
	var anonymousFields []reflect.Type
	valuePtr := reflect.ValueOf(obj)
	value := valuePtr.Elem()
	class := classPtr.Elem()
	for j := 0; j < class.NumField(); j++ {
		field := class.Field(j)
		if field.Anonymous {
			anonymousFields = append(anonymousFields, field.Type)
			switch field.Type {
			case NamedBeanClass:
				stub := &namedBeanStub{name: classPtr.String()}
				stubValuePtr := reflect.ValueOf(stub)
				value.Field(j).Set(stubValuePtr)
			case OrderedBeanClass:
				stub := &orderedBeanStub{}
				stubValuePtr := reflect.ValueOf(stub)
				value.Field(j).Set(stubValuePtr)
			case InitializingBeanClass:
				stub := &initializingBeanStub{name: classPtr.String()}
				stubValuePtr := reflect.ValueOf(stub)
				value.Field(j).Set(stubValuePtr)
			case DisposableBeanClass:
				stub := &disposableBeanStub{name: classPtr.String()}
				stubValuePtr := reflect.ValueOf(stub)
				value.Field(j).Set(stubValuePtr)
			case FactoryBeanClass:
				stub := &factoryBeanStub{name: classPtr.String(), elemType: classPtr}
				stubValuePtr := reflect.ValueOf(stub)
				value.Field(j).Set(stubValuePtr)
			case ContextClass:
				return nil, errors.Errorf("exposing by anonymous field '%s' in '%v' interface beans.Context is not allowed", field.Name, classPtr)
			}
		}
		injectTag, hasInjectTag := field.Tag.Lookup("inject")
		if field.Tag == "inject" || hasInjectTag {
			if field.Anonymous {
				return nil, errors.Errorf("injection to anonymous field '%s' in '%v' is not allowed", field.Name, classPtr)
			}
			var specificBean string
			var fieldOptional bool
			var fieldLazy bool
			if hasInjectTag {
				pairs := strings.Split(injectTag, ",")
				for _, pair := range pairs {
					p := strings.TrimSpace(pair)
					kv := strings.Split(p, "=")
					switch strings.TrimSpace(kv[0]) {
					case "bean":
						if len(kv) > 1 {
							specificBean = strings.TrimSpace(kv[1])
						}
					case "optional":
						fieldOptional = true
					case "lazy":
						fieldLazy = true
					}
				}
			}
			kind := field.Type.Kind()
			fieldType := field.Type
			var fieldSlice, fieldMap bool
			switch kind {
			case reflect.Slice:
				fieldSlice = true
				fieldType = field.Type.Elem()
				kind = fieldType.Kind()
			case reflect.Map:
				fieldMap = true
				if field.Type.Key().Kind() != reflect.String {
					return nil, errors.Errorf("map must have string key to be injected for field type '%v' on position %d in %v with 'inject' tag", field.Type, j, classPtr)
				}
				fieldType = field.Type.Elem()
				kind = fieldType.Kind()
			}
			if kind != reflect.Ptr && kind != reflect.Interface && kind != reflect.Func {
				return nil, errors.Errorf("not a pointer, interface or function field type '%v' on position %d in %v with 'inject' tag", field.Type, j, classPtr)
			}
			injectDef := &injectionDef{
				class:     class,
				fieldNum:  j,
				fieldName: field.Name,
				fieldType: fieldType,
				lazy:      fieldLazy,
				slice:     fieldSlice,
				table:     fieldMap,
				optional:  fieldOptional,
				qualifier: specificBean,
			}
			fields = append(fields, injectDef)
		}
	}
	name := classPtr.String()
	var qualifier string
	if namedBean, ok := obj.(NamedBean); ok {
		name = namedBean.BeanName()
		qualifier = name
	}
	ordered := false
	var order int
	if orderedBean, ok := obj.(OrderedBean); ok {
		ordered = true
		order = orderedBean.BeanOrder()
	}
	return &bean{
		name:     name,
		qualifier: qualifier,
		ordered:  ordered,
		order:    order,
		obj:      obj,
		valuePtr: valuePtr,
		beanDef: &beanDef{
			classPtr:        classPtr,
			anonymousFields: anonymousFields,
			fields:          fields,
		},
		lifecycle: BeanCreated,
	}, nil
}

func isSomeoneImplements(iface reflect.Type, list []reflect.Type) bool {
	for _, el := range list {
		if el.Implements(iface) {
			return true
		}
	}
	return false
}
