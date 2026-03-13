package gtag

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/json"
)

// Type name => enums json.
var enumsMap = make(map[string]json.RawMessage)

// SetGlobalEnums sets the global enums into package.
// Note that this operation is not concurrent safety.
func SetGlobalEnums(enumsJson string) error {
	return json.Unmarshal([]byte(enumsJson), &enumsMap)
}

// GetGlobalEnums retrieves and returns the global enums.
func GetGlobalEnums() (string, error) {
	enumsBytes, err := json.Marshal(enumsMap)
	if err != nil {
		return "", err
	}
	return string(enumsBytes), nil
}

// GetEnumsByType retrieves and returns the stored enums json by type name.
// The type name is like: gofly/utils/tools/gjson.ContentType
func GetEnumsByType(typeName string) string {
	return string(enumsMap[typeName])
}
