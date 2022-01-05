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
)

type injectionDef struct {

	/**
	Class of that struct
	*/
	class reflect.Type
	/**
	Field number of that struct
	*/
	fieldNum int
	/**
	Field name where injection is going to be happen
	*/
	fieldName string
	/**
	Type of the field that is going to be injected
	*/
	fieldType reflect.Type
	/**
	Field is Slice of elements
	*/
	slice bool
	/**
	Lazy injection represented by function
	*/
	lazy bool
	/**
	Optional injection
	*/
	optional bool
	/*
		Injection expects specific bean to be injected
	*/
	specificBean string
}

type injection struct {

	/*
		Bean where injection is going to be happen
	*/
	bean *bean

	/**
	Reflection value of the bean where injection is going to be happen
	*/
	value reflect.Value

	/**
	Injection information
	*/
	injectionDef *injectionDef
}

/**
Inject value in to the field by using reflection
Returns beanlist of used beans in this injection
*/
func (t *injection) inject(list *beanlist) error {
	if list.head == nil {
		return errors.Errorf("field '%s' in class '%v' can not be injected with nil bean", t.injectionDef.fieldName, t.injectionDef.class)
	}

	field := t.value.Field(t.injectionDef.fieldNum)
	if !field.CanSet() {
		return errors.Errorf("field '%s' in class '%v' is not public", t.injectionDef.fieldName, t.injectionDef.class)
	}

	if t.injectionDef.slice {
		newSlice := field
		var factorylist []*bean
		list.forEach(func(instance *bean) {
			if instance.beenFactory != nil {
				factorylist = append(factorylist, instance)
			} else {
				newSlice = reflect.Append(newSlice, instance.valuePtr)
			}
		})
		field.Set(newSlice)

		for _, instance := range factorylist {
			// register factory dependency for 'inject.bean' that is using 'factory'
			t.bean.factoryDependencies = append(t.bean.factoryDependencies,
				&factoryDependency{
					factory: instance.beenFactory,
					injection: func(service *bean) error {
						field.Set(reflect.Append(field, instance.valuePtr))
						return nil
					},
				})
		}

		return nil
	}

	impl, err := t.injectionDef.selectBean(list)
	if err != nil {
		return err
	}

	if impl.beenFactory != nil {
		if t.injectionDef.lazy {
			return errors.Errorf("lazy injection is not supported of type '%v' through factory '%v' in to '%v'", impl.beenFactory.factoryBean.ObjectType(), impl.beenFactory.factoryClassPtr, t.String())
		}

		// register factory dependency for 'inject.bean' that is using 'factory'
		t.bean.factoryDependencies = append(t.bean.factoryDependencies,
			&factoryDependency{
				factory: impl.beenFactory,
				injection: func(service *bean) error {
					field.Set(service.valuePtr)
					return nil
				},
			})

		return nil
	}

	field.Set(impl.valuePtr)

	// register dependency that 'inject.bean' is using if it is not lazy
	if !t.injectionDef.lazy {
		t.bean.dependencies = append(t.bean.dependencies, oneBean(impl))
	}

	return nil
}

func (t *injectionDef) inject(value *reflect.Value, list []*bean) error {
	field := value.Field(t.fieldNum)

	if !field.CanSet() {
		return errors.Errorf("field '%s' in class '%v' is not public", t.fieldName, t.class)
	}

	if t.slice {
		newSlice := field
		for _, bean := range list {
			if !bean.valuePtr.IsValid() {
				newSlice = reflect.Append(newSlice, reflect.Zero(t.fieldType))
			} else {
				newSlice = reflect.Append(newSlice, bean.valuePtr)
			}
		}
		field.Set(newSlice)
		return nil
	}

	switch len(list) {

	case 0:
		return errors.Errorf("can not find candidates to inject the field '%s' in class '%v'", t.fieldName, t.class)

	case 1:
		impl := list[0]
		if t.lazy {
			fn := reflect.MakeFunc(field.Type(), func(args []reflect.Value) (results []reflect.Value) {
				if impl.lifecycle != BeanInitialized {
					return []reflect.Value{reflect.Zero(t.fieldType)}
				} else {
					return []reflect.Value{impl.valuePtr}
				}
			})
			field.Set(fn)
		} else {
			field.Set(impl.valuePtr)
		}
		return nil

	default:
		return errors.Errorf("can not inject to field '%s' in class '%v' non single bean '%+v'", t.fieldName, t.class, list)

	}

}

func (t *injectionDef) selectBean(list *beanlist) (*bean, error) {
	if list.single() {
		return list.head, nil
	}
	if t.specificBean == "" {
		return nil, errors.Errorf("field '%s' in class '%v' can not be injected with multiple candidates %+v", t.fieldName, t.class, list.list())
	}
	var candidates []*bean
	list.forEach(func(b *bean) {
		if t.specificBean == b.name {
			candidates = append(candidates, b)
		}
	})
	switch len(candidates) {
	case 0:
		return nil, errors.Errorf("field '%s' in class '%v' with specific bean '%s' can not find candidates", t.fieldName, t.class, t.specificBean)
	case 1:
		return candidates[0], nil
	default:
		return nil, errors.Errorf("field '%s' in class '%v' with specific bean '%s' can not be injected with multiple candidates %+v", t.fieldName, t.class, t.specificBean, candidates)
	}
}

/**
User friendly information about class and field
*/

func (t *injection) String() string {
	return t.injectionDef.String()
}

func (t *injectionDef) String() string {
	return fmt.Sprintf(" %v->%s ", t.class, t.fieldName)
}
