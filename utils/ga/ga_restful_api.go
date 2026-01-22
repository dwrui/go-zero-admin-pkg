package ga

import (
	"time"
)

const defaultMultipartMemory = 32 << 20 // 32 MB
// 组装Restful API 接口返回数据
// 返回信息主体
type (
	R struct {
		Code    int
		Message string
		Data    interface{}
		Exdata  interface{}
		Token   string
		Time    int64
	}
)

// 错误码
var (
	succCode = 1 // 成功
	errCode  = 0 // 失败
)

// 设置返回内容
func (r *R) SetData(data interface{}) *R {
	r.Data = data
	return r
}

// 设置返回扩展内容内容
func (r *R) SetExdata(exdata interface{}) *R {
	r.Exdata = exdata
	return r
}

// 设置编码
func (r *R) SetCode(code int) *R {
	r.Code = code
	return r
}

// 设置返回提示信息
func (r *R) SetMsg(msg string) *R {
	r.Message = msg
	return r
}

// 设置返回Token
func (r *R) SetToken(token string) *R {
	r.Token = token
	return r
}

// 返回成功内容
func Success() *R {
	r := &R{}
	r.Message = "Success"
	r.Code = succCode
	r.Time = time.Now().UnixMilli()
	return r
}

//// 接口返回成功内容
//func (r *R) Regin(ctx *GinCtx) {
//	if r.Token == "" {
//		//如果已经登录则刷新token
//		getuser, exists := ctx.Get("user") //当前用户
//		if exists {
//			userinfo := getuser.(*routeuse.UserClaims)
//			tokenouttime := gconv.Int64(appConf_arr["tokenouttime"])
//			if userinfo.ExpiresAt-gtime.Now().Unix() < tokenouttime*60/2 { //小设置的时间超时时间一半就刷新/单位秒
//				token := ctx.Request.Header.Get("Authorization")
//				tockenarr, err := routeuse.Refresh(token)
//				if err == nil {
//					r.Token = gconv.String(tockenarr)
//				}
//			}
//		}
//	}
//	ctx.JSON(http.StatusOK, GinObj{
//		"code":    r.Code,
//		"message": r.Message,
//		"data":    r.Data,
//		"exdata":  r.Exdata,
//		"token":   r.Token,
//		"time":    r.Time,
//	})
//}

// 返回失败内容
func Failed() *R {
	r := &R{}
	r.Message = "Fail"
	r.Code = errCode
	r.Data = false
	r.Time = time.Now().UnixMilli()
	return r
}
