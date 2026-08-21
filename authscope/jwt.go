package authscope

// JwtSettings JWT 签发/校验参数。
type JwtSettings struct {
	AccessSecret string
	AccessExpire int64
}

// JwtForScope 按 scope 选择 JWT 配置；总后台可独立密钥。
func JwtForScope(scope string, admin, superadmin JwtSettings) JwtSettings {
	if scope == Superadmin && superadmin.AccessSecret != "" {
		if superadmin.AccessExpire > 0 {
			return superadmin
		}
		return JwtSettings{
			AccessSecret: superadmin.AccessSecret,
			AccessExpire: admin.AccessExpire,
		}
	}
	return admin
}
