package ga

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvalid"
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gvar"
)

// 把任何类型转var
func VarNew(val interface{}) *gvar.Var {
	return gvar.New(val)
}

func Validator() *gvalid.MessageValidator {
	return gvalid.NewMessageValidator()
}
