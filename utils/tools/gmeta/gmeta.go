// Package gmeta provides embedded meta data feature for struct.
package gmeta

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gstructs"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
	"reflect"
)

// Meta is used as an embedded attribute for struct to enabled metadata feature.
type Meta struct{}

// metaAttributeName is the attribute name of metadata in struct.
const metaAttributeName = "Meta"

// metaType holds the reflection. Type of Meta, used for efficient type comparison.
var metaType = reflect.TypeOf(Meta{})

// Data retrieves and returns all metadata from `object`.
func Data(object any) map[string]string {
	reflectType, err := gstructs.StructType(object)
	if err != nil {
		return nil
	}
	if field, ok := reflectType.FieldByName(metaAttributeName); ok {
		if field.Type == metaType {
			return gstructs.ParseTag(string(field.Tag))
		}
	}
	return map[string]string{}
}

// Get retrieves and returns specified metadata by `key` from `object`.
func Get(object any, key string) *gvar.Var {
	v, ok := Data(object)[key]
	if !ok {
		return nil
	}
	return gvar.New(v)
}
