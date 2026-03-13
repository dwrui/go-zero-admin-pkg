package ga

//import (
//	"context"
//	"github.com/dwrui/go-zero-admin/pkg/utils/plugin"
//	"net/http"
//)
//
///**
//* 后台RBAC的权限管理
// */
//
//// 检查权限
//// path是当前请求路径
//func CheckAuth(c *GinCtx, modelname string) bool {
//	role_id, acerr := Model(modelname+"_auth_role_access").Where("uid", c.GetInt64("userID")).Array("role_id")
//	if acerr != nil || role_id == nil {
//		return false
//	}
//	//1.判断是否有超级角色
//	super_role, rerr := Model(modelname+"_auth_role").WhereIn("id", role_id).Where("rules", "*").Count()
//	if rerr != nil {
//		return false
//	}
//	if super_role != 0 { //超级角色
//		if Bool(appConf_arr["superRoleAuth"]) {
//			//1.需要查找是否已经把权限接口添加到数据库
//			hasepath, ruerr := Model(modelname+"_auth_rule").Where("status", 0).Where("type", 2).Where("path", c.FullPath()).Count()
//			if ruerr == nil && hasepath != 0 {
//				return true
//			}
//		} else {
//			//2.不需添加权限数据直接返回
//			return true
//		}
//	} else { //普通角色
//		menu_ids, rerr := Model(modelname+"_auth_role").WhereIn("id", role_id).Array("rules")
//		if rerr != nil {
//			return false
//		}
//		hasepath, ruerr := Model(modelname+"_auth_rule").Where("status", 0).Where("type", 2).WhereIn("id", ArrayMerge(menu_ids)).Where("path", c.FullPath()).Count()
//		if ruerr == nil && hasepath != 0 {
//			return true
//		}
//	}
//	return false
//}
