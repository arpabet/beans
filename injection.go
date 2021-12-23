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
*/
func (t *injection) inject(impl *bean) error {
	return t.injectionDef.inject(&t.value, impl)
}

func (t *injectionDef) inject(value *reflect.Value, impl *bean) error {
	field := value.Field(t.fieldNum)
	if field.CanSet() {
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
	} else {
		return errors.Errorf("field '%s' in class '%v' is not public", t.fieldName, t.class)
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
