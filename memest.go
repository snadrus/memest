package memest

import (
	"reflect"
)

type sizeInfo struct {
	size       uintptr
	isVariable bool
}

func DeepSize(v interface{}) uintptr {
	seen := make(map[uintptr]bool)
	structSizes := map[string]uintptr{} //may be useful as a global syncMap

	var deepSizeValue func(val reflect.Value) sizeInfo
	deepSizeValue = func(val reflect.Value) sizeInfo {
		if !val.IsValid() {
			return sizeInfo{}
		}

		kind := val.Kind()
		var info sizeInfo

		// Dereference pointer and interface
		for kind == reflect.Ptr || kind == reflect.Interface {
			if val.IsNil() {
				return sizeInfo{}
			}
			ptr := val.Pointer()
			if seen[ptr] {
				return sizeInfo{}
			}
			seen[ptr] = true
			val = val.Elem()
			kind = val.Kind()
		}

		typ := val.Type()
		info.size = typ.Size()

		switch kind {
		case reflect.Struct:
			tn := typ.Name()
			// Check if we've already calculated size for this non-variable struct type
			if cachedSize := structSizes[tn]; cachedSize != 0 {
				info.size = cachedSize
				return info
			}

			for i := 0; i < val.NumField(); i++ {
				fieldInfo := deepSizeValue(val.Field(i))
				info.size += fieldInfo.size
				info.isVariable = info.isVariable || fieldInfo.isVariable
			}

			// Cache the size if struct is not variable
			if !info.isVariable {
				structSizes[tn] = info.size
			}

		case reflect.Slice:
			info.isVariable = true
			if val.Len() > 0 {
				// Only need to calculate size of one element for non-variable types
				elemInfo := deepSizeValue(val.Index(0))
				if !elemInfo.isVariable {
					info.size += elemInfo.size * uintptr(val.Len())
				} else {
					// For variable types, we need to check all elements
					for i := 0; i < val.Len(); i++ {
						elemInfo := deepSizeValue(val.Index(i))
						info.size += elemInfo.size
					}
				}
			}

		case reflect.Map:
			info.isVariable = true
			if val.Len() > 0 {
				iter := val.MapRange()
				iter.Next() // Get first key-value pair
				keyInfo := deepSizeValue(iter.Key())
				valueInfo := deepSizeValue(iter.Value())

				if !keyInfo.isVariable {
					info.size += keyInfo.size * uintptr(val.Len())
				}
				if !valueInfo.isVariable {
					info.size += valueInfo.size * uintptr(val.Len())
				}
				if keyInfo.isVariable || valueInfo.isVariable {
					// For variable types, we need to check all elements
					info.size += keyInfo.size + valueInfo.size
					for iter.Next() {
						if keyInfo.isVariable {
							keyInfo := deepSizeValue(iter.Key())
							info.size += keyInfo.size
						}
						if valueInfo.isVariable {
							valueInfo := deepSizeValue(iter.Value())
							info.size += valueInfo.size
						}
					}
				}
			}

		case reflect.String:
			info.size += uintptr(val.Len())
			info.isVariable = true

		case reflect.Array:
			if val.Len() > 0 {
				elemInfo := deepSizeValue(val.Index(0))
				if !elemInfo.isVariable {
					// For non-variable types, we can calculate total size directly
					info.size += elemInfo.size * uintptr(val.Len())
				} else {
					// For variable types, we need to check all elements
					for i := 0; i < val.Len(); i++ {
						elemInfo := deepSizeValue(val.Index(i))
						info.size += elemInfo.size
					}
				}
			}
		}

		return info
	}

	info := deepSizeValue(reflect.ValueOf(v))
	return info.size
}
