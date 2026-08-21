// Package authscope 在 admin（B 端）与 superadmin（总后台）之间切换数据表与状态语义。
package authscope

import (
	"context"
	"strings"

	"google.golang.org/grpc/metadata"
)

const (
	// Admin B 端租户后台：ga_admin_*
	Admin = "admin"
	// Superadmin 总后台：ga_superadmin_*
	Superadmin = "superadmin"
	// Biz 总后台管理租户：读写 ga_admin_*，不做 business_id 隔离
	Biz = "biz"

	metadataKey = "auth-scope"
)

// TablePrefix 返回 ORM 表逻辑名前缀（不含 ga_）。
func TablePrefix(scope string) string {
	switch scope {
	case Superadmin:
		return Superadmin
	case Biz:
		return Admin
	default:
		return Admin
	}
}

// AuthTable 根据 scope 返回 auth 相关表前缀。
func AuthTable(scope string) string {
	return TablePrefix(scope)
}

// Model 拼接逻辑表名，如 auth_account。
func Model(scope, suffix string) string {
	return TablePrefix(scope) + "_" + strings.TrimPrefix(suffix, "_")
}

// EnabledStatus 菜单/角色等「启用」状态值：B 端 1，总后台 0。
func EnabledStatus(scope string) int64 {
	if scope == Superadmin {
		return 0
	}
	return 1
}

// IsAccountDisabled 账号是否禁用。
func IsAccountDisabled(scope string, status int64) bool {
	if scope == Superadmin {
		return status == 1
	}
	return status != 1
}

// SkipBusinessFilter 平台视角是否跳过 business_id 过滤。
func SkipBusinessFilter(scope string) bool {
	return scope == Superadmin || scope == Biz
}

// LogType 操作日志类型标识。
func LogType(scope string) string {
	if scope == Superadmin {
		return "adminpro"
	}
	return "admin"
}

// LoginSelectFields 登录查询字段（总后台账号表无 business_id）。
func LoginSelectFields(scope string) string {
	if scope == Superadmin {
		return "id,account_id,password,salt,name,status,login_attempts,lock_time,dept_id"
	}
	return "id,account_id,business_id,password,salt,name,status,login_attempts,lock_time,dept_id"
}

// TokenBusinessID JWT 中的 business_id（总后台固定 0）。
func TokenBusinessID(scope string, businessId int64) int64 {
	if scope == Superadmin {
		return 0
	}
	return businessId
}

// UserInfoSelectFields CheckToken 查询用户字段。
func UserInfoSelectFields(scope string) string {
	if scope == Superadmin {
		return "dept_id"
	}
	return "business_id,dept_id"
}

// WithOutgoing 在 gRPC 客户端调用前注入 scope。
func WithOutgoing(ctx context.Context, scope string) context.Context {
	if scope == "" {
		scope = Admin
	}
	return metadata.AppendToOutgoingContext(ctx, metadataKey, scope)
}

// RoleOwnerColumn 角色表「创建者」字段名。
func RoleOwnerColumn(scope string) string {
	if scope == Superadmin {
		return "uid"
	}
	return "account_id"
}

// AccountTimeColumns 账号表时间字段（superadmin 仍为 createtime/updatetime）。
func AccountTimeColumns(scope string) (createCol, updateCol string) {
	if scope == Superadmin {
		return "createtime", "updatetime"
	}
	return "create_time", "update_time"
}

// FromIncoming 从 gRPC 服务端上下文读取 scope，默认 admin。
func FromIncoming(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return Admin
	}
	vals := md.Get(metadataKey)
	if len(vals) == 0 || vals[0] == "" {
		return Admin
	}
	return vals[0]
}

// FromRequestPath 根据 HTTP 路径推断 scope（网关兜底）。
func FromRequestPath(path string) string {
	if strings.HasPrefix(path, "/admin/") || strings.HasPrefix(path, "/super-admin/") {
		return Superadmin
	}
	if strings.HasPrefix(path, "/api/admin") {
		return Admin
	}
	return Admin
}
