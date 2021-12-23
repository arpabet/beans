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
	BeanCreated int32 = iota
	BeanConstructing
	BeanInitialized
	BeanDestroyed
)

type bean struct {
	/**
	Instance to the bean
	*/
	obj interface{}

	/**
	Reflect instance to the pointer or interface of the bean
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
	dependencies []*bean

	/**
	List of factory beans that should initialize before current bean
	*/
	factoryDependencies []*factoryDependency
}

func (t *bean) String() string {
	return t.beanDef.classPtr.String()
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
	Factory bean
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
	Singleton object
	*/
	singletonObj interface{}

	/**
	Singleton bean
	*/
	singletonBean *bean
}

func (t *factory) String() string {
	return t.factoryClassPtr.String()
}

func (t *factory) ctor() (*bean, error) {
	if !t.factoryBean.Singleton() {
		t.singletonObj = nil
	}
	if t.singletonObj == nil {
		var err error
		t.singletonObj, err = t.factoryBean.Object()
		if err != nil {
			return nil, errors.Errorf("factory bean '%v' failed to create bean '%v', %v", t.factoryClassPtr, t.factoryBean.ObjectType(), err)
		}
		producedClassPtr := reflect.TypeOf(t.singletonObj)
		if producedClassPtr != t.factoryBean.ObjectType() && !producedClassPtr.Implements(t.factoryBean.ObjectType()) {
			return nil, errors.Errorf("factory bean '%v' produced '%v' object that does not implement or equal '%v'", t.factoryClassPtr, producedClassPtr, t.factoryBean.ObjectType())
		}
		if t.singletonBean != nil {
			t.singletonBean = &bean{
				obj:       t.singletonObj,
				valuePtr:  t.singletonBean.valuePtr,
				beanDef:   t.singletonBean.beanDef,
				lifecycle: BeanCreated,
			}
		} else {
			t.singletonBean, err = investigate(t.singletonObj, producedClassPtr)
			if err != nil {
				return nil, errors.Errorf("factory bean '%v' produced invalid bean '%v', %v", t.factoryClassPtr, producedClassPtr, err)
			}
			for _, injectDef := range t.singletonBean.beanDef.fields {
				return nil, errors.Errorf("factory bean '%v' produced bean '%v' with 'inject' annotated field '%v' on position %d that is should be injected by the factory itself", t.factoryClassPtr, producedClassPtr, injectDef.fieldName, injectDef.fieldNum)
			}
		}
	}
	return t.singletonBean, nil
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
			fieldLazy := false
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
				optional:     optionalBean,
				specificBean: specificBean,
			}
			fields = append(fields, injectDef)
		}
	}
	return &bean{
		obj:      obj,
		valuePtr: valuePtr,
		beanDef: &beanDef{
			classPtr:      classPtr,
			notImplements: notImplements,
			fields:        fields,
		},
	}, nil
}
