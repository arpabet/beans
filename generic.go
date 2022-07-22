/**
  Copyright (c) 2022 Arpabet, LLC. All rights reserved.
*/

package beans

import (
	"reflect"
)

func GetBean[T any](context Context, typ reflect.Type) (ret T, ok bool) {
	list := context.Bean(typ, 0)
	if len(list) == 0 {
		ok = false
		return
	}
	ret, ok = list[0].Object().(T)
	return
}

