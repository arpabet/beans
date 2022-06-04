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
	Field is Slice of beans
	*/
	slice bool
	/**
	Field is Map of beans
	*/
	table bool
	/**
	Lazy injection represented by function
	*/
	lazy bool
	/**
	Optional injection
	*/
	optional bool
	/*
		Injection expects the specific bean to be injected
	*/
	qualifier string
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
func (t *injection) inject(list []*bean) error {

	field := t.value.Field(t.injectionDef.fieldNum)
	if !field.CanSet() {
		return errors.Errorf("field '%s' in class '%v' is not public", t.injectionDef.fieldName, t.injectionDef.class)
	}

	list = t.injectionDef.filterBeans(list)

	if len(list) == 0 {
		if !t.injectionDef.optional {
			return errors.Errorf("can not find candidates to inject the required field '%s' in class '%v'", t.injectionDef.fieldName, t.injectionDef.class)
		}
		return nil
	}

	if t.injectionDef.slice {

		newSlice := field
		var factoryList []*bean
		for _, instance := range list {
			if instance.beenFactory != nil {
				factoryList = append(factoryList, instance)
			} else {
				newSlice = reflect.Append(newSlice, instance.valuePtr)
			}
		}
		field.Set(newSlice)

		for _, instance := range factoryList {
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

	if t.injectionDef.table {

		field.Set(reflect.MakeMap(field.Type()))

		visited := make(map[string]bool)
		for _, instance := range list {
			if instance.beenFactory != nil {
				// register factory dependency for 'inject.bean' that is using 'factory'
				t.bean.factoryDependencies = append(t.bean.factoryDependencies,
					&factoryDependency{
						factory: instance.beenFactory,
						injection: func(service *bean) error {
							if visited[instance.name] {
								return errors.Errorf("can not inject duplicates '%s' to the map field '%s' in class '%v'", instance.name, t.injectionDef.fieldName, t.injectionDef.class)
							}
							visited[instance.name] = true
							field.SetMapIndex(reflect.ValueOf(instance.name), instance.valuePtr)
							return nil
						},
					})
			} else {
				if visited[instance.name] {
					return errors.Errorf("can not inject duplicates '%s' to the map field '%s' in class '%v'", instance.name, t.injectionDef.fieldName, t.injectionDef.class)
				}
				visited[instance.name] = true
				field.SetMapIndex(reflect.ValueOf(instance.name), instance.valuePtr)
			}
		}

		return nil
	}

	if len(list) > 1 {
		return errors.Errorf("field '%s' in class '%v' can not be injected with multiple candidates %+v", t.injectionDef.fieldName, t.injectionDef.class, list)
	}

	impl := list[0]

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

// runtime injection
func (t *injectionDef) inject(value *reflect.Value, list []*bean) error {
	field := value.Field(t.fieldNum)

	if !field.CanSet() {
		return errors.Errorf("field '%s' in class '%v' is not public", t.fieldName, t.class)
	}

	list = t.filterBeans(list)

	if len(list) == 0 {
		if !t.optional {
			return errors.Errorf("can not find candidates to inject the required field '%s' in class '%v'", t.fieldName, t.class)
		}
		return nil
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

	if t.table {

		field.Set(reflect.MakeMap(field.Type()))

		visited := make(map[string]bool)
		for _, instance := range list {
			if !instance.valuePtr.IsValid() {
				if visited[instance.name] {
					return errors.Errorf("can not inject duplicates '%s' to the map field '%s' in class '%v'", instance.name, t.fieldName, t.class)
				}
				visited[instance.name] = true
				field.SetMapIndex(reflect.ValueOf(instance.name), instance.valuePtr)
			}
		}

		return nil
	}

	if len(list) > 1 {
		return errors.Errorf("field '%s' in class '%v' can not be injected with multiple candidates %+v", t.fieldName, t.class, list)
	}

	impl := list[0]

	if impl.lifecycle != BeanInitialized {
		return errors.Errorf("field '%s' in class '%v' can not be injected with non-initialized bean %+v", t.fieldName, t.class, impl)
	}

	if impl.beenFactory != nil {

		service, _, err := impl.beenFactory.ctor()
		if err != nil {
			return errors.Errorf("field '%s' in class '%v' can not be injected because of factory bean %+v error, %v", t.fieldName, t.class, impl, err)
		}

		impl = service
	}

	field.Set(impl.valuePtr)

	return nil
}

func (t *injectionDef) filterBeans(list []*bean) []*bean {
	if t.qualifier != "" {
		var candidates []*bean
		for _, b := range list {
			if t.qualifier == b.name {
				candidates = append(candidates, b)
			}
		}
		return candidates
	} else {
		return list
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
