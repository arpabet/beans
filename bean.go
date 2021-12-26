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
)

type beanDef struct {
	/**
	Class of the pointer to the struct or interface
	*/
	classPtr reflect.Type

	/**
	Anonymous fields expose their interfaces though bean itself.
	This is confusing on injection, because this bean is an encapsulator, not an implementation.

	Skip those fields.
	*/
	notImplements []reflect.Type

	/**
	Fields that are going to be injected
	*/
	fields []*injectionDef
}

const (
	BeanAllocated int32 = iota
	BeanCreated
	BeanConstructing
	BeanInitialized
	BeanDestroyed
)

type bean struct {
	/**
	Name of the bean
	*/
	name string

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
	lifecycle int32

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
}

func (t *bean) String() string {
	if t.beenFactory != nil {
		return fmt.Sprintf("<FactoryBean %s->%s>", t.beenFactory.factoryClassPtr, t.beanDef.classPtr)
	} else {
		return fmt.Sprintf("<Bean %s>", t.beanDef.classPtr)
	}
}

/**
Check if bean definition can implement interface type
*/
func (t *beanDef) implements(ifaceType reflect.Type) bool {
	for _, ni := range t.notImplements {
		if ni == ifaceType {
			return false
		}
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

func (t *beanlist) hasName(name string) bool {
	for b := t.head; b != nil; b = b.next {
		if b.name == name {
			return true
		}
		if b == t.tail {
			break
		}
	}
	return false
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
	name := classPtr.String()
	if namedBean, ok := obj.(NamedBean); ok {
		name = namedBean.BeanName()
	}
	var fields []*injectionDef
	var notImplements []reflect.Type
	valuePtr := reflect.ValueOf(obj)
	class := classPtr.Elem()
	for j := 0; j < class.NumField(); j++ {
		field := class.Field(j)
		if field.Anonymous {
			notImplements = append(notImplements, field.Type)
		}
		injectTag, hasInjectTag := field.Tag.Lookup("inject")
		if field.Tag == "inject" || hasInjectTag {
			var specificBean string
			var optionalBean bool
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
						optionalBean = true
					}
				}
			}
			kind := field.Type.Kind()
			fieldType := field.Type
			var fieldLazy, fieldSlice bool
			if kind == reflect.Slice {
				fmt.Printf("field.Name = %s, field.Kind = %v, key=%v\n", field.Name, field.Type.Kind(), field.Type.Elem())
				fieldSlice = true
				fieldType = field.Type.Elem()
				kind = fieldType.Kind()
			}
			if kind == reflect.Func && field.Type.NumIn() == 0 && field.Type.NumOut() == 1 {
				fieldType = field.Type.Out(0)
				fieldLazy = true
				kind = fieldType.Kind()
			}
			if kind != reflect.Ptr && kind != reflect.Interface {
				return nil, errors.Errorf("not a pointer or interface field type '%v' on position %d in %v", field.Type, j, classPtr)
			}
			injectDef := &injectionDef{
				class:        class,
				fieldNum:     j,
				fieldName:    field.Name,
				fieldType:    fieldType,
				lazy:         fieldLazy,
				slice:        fieldSlice,
				optional:     optionalBean,
				specificBean: specificBean,
			}
			fields = append(fields, injectDef)
		}
	}
	return &bean{
		name:     name,
		obj:      obj,
		valuePtr: valuePtr,
		beanDef: &beanDef{
			classPtr:      classPtr,
			notImplements: notImplements,
			fields:        fields,
		},
	}, nil
}
