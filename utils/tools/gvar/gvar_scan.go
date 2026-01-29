package gvar

import (
	"github.com/dwrui/go-zero-admin/pkg/utils/tools/gconv"
)

// Scan automatically checks the type of `pointer` and converts value of Var to `pointer`.
//
// See gconv.Scan.
func (v *Var) Scan(pointer interface{}, mapping ...map[string]string) error {
	return gconv.Scan(v.Val(), pointer, mapping...)
}
