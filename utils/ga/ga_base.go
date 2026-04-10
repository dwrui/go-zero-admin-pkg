package ga

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gstr"
	"github.com/dwrui/go-zero-admin-pkg/utils/tools/gvar"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// 判断元素是否存在数组中
func IsContain(items []interface{}, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

// 判断元素是否存在数组中-字符串类型
func IsContainStr(items []string, item string) bool {
	for _, eachItem := range items {
		if eachItem == item {
			return true
		}
	}
	return false
}

// 获取ip函数
func GetIp(r *http.Request) string {
	// 1. 优先检查 X-Forwarded-For
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		// 取第一个IP
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = strings.TrimSpace(ip[:idx])
		}
		// 处理本地地址
		if ip == "::1" || ip == "" {
			return "127.0.0.1"
		}
		return ip
	}

	// 2. 检查 X-Real-IP（Nginx常用）
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		if ip == "::1" {
			return "127.0.0.1"
		}
		return ip
	}

	// 3. 使用 RemoteAddr
	ip := r.RemoteAddr
	if ip != "" {
		// 去除端口号
		if idx := strings.LastIndex(ip, ":"); idx != -1 {
			ip = ip[:idx]
		}
		// 处理本地地址
		if ip == "::1" || ip == "" {
			return "127.0.0.1"
		}
		return ip
	}

	// 默认返回本地IP
	return "127.0.0.1"
}

// 获取本地ip
func LocalIP() string {
	ip := ""
	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && !ipnet.IP.IsMulticast() && !ipnet.IP.IsLinkLocalUnicast() && !ipnet.IP.IsLinkLocalMulticast() && ipnet.IP.To4() != nil {
				ip = ipnet.IP.String()
			}
		}
	}
	return ip
}

// FindAllChildrenIDs 根据 targetID 切片查找所有下级 ID
func FindAllChildrenIDs(data []map[string]uint64, targetIDs []uint64) []uint64 {
	// 用于存储所有下级 ID，避免重复
	idSet := make(map[uint64]bool)

	// 遍历 targetID 切片中的每个 ID
	for _, targetID := range targetIDs {
		findChildrenRecursive(data, targetID, idSet)
	}

	// 将 map 转换为切片返回
	result := make([]uint64, 0, len(idSet))
	for id := range idSet {
		result = append(result, id)
	}
	return result
}

// findChildrenRecursive 递归查找子级 ID
func findChildrenRecursive(data []map[string]uint64, targetID uint64, idSet map[uint64]bool) {
	for _, item := range data {
		// 如果当前项的 pid 匹配 targetID，则记录其 id
		if pid, ok := item["pid"]; ok && pid == targetID {
			id := item["id"]
			if !idSet[id] {
				idSet[id] = true
				// 递归查找该 id 的子级
				findChildrenRecursive(data, id, idSet)
			}
		}
	}
}

// 合并数组-两个数组合并为一个数组
func MergeArrUint64(a []interface{}, b []interface{}) []uint64 {
	var arr []uint64
	for _, i := range a {
		arr = append(arr, Uint64(i))
	}
	for _, j := range b {
		arr = append(arr, Uint64(j))
	}
	return arr
}

// 多维数组合并-权限
func ArrayMerge(data []*gvar.Var) []interface{} {
	var rule_ids_arr []interface{}
	for _, mainv := range data {
		ids_arr := strings.Split(mainv.String(), `,`)
		for _, intv := range ids_arr {
			rule_ids_arr = append(rule_ids_arr, intv)
		}
	}
	return rule_ids_arr
}

// 把字符串打散为数组
func Axplode(data string) []interface{} {
	var rule_ids_arr []interface{}
	ids_arr := strings.Split(data, `,`)
	for _, intv := range ids_arr {
		rule_ids_arr = append(rule_ids_arr, intv)
	}
	return rule_ids_arr
}

// Int类型是否存在Var数组中
func IntInVarArray(target int, arr []*gvar.Var) bool {
	for _, element := range arr {
		if target == Int(element) {
			return true
		}
	}
	return false
}

// Int类型是否存在interface数组中
func IntInInterfaceArray(target int, arr []interface{}) bool {
	for _, element := range arr {
		if target == Int(element) {
			return true
		}
	}
	return false
}

// 转JSON编码为字符串
func JSONToString(data interface{}) string {
	if str, err := json.Marshal(data); err != nil {
		return ""
	} else {
		return string(str)
	}
}

// 字符串转JSON编码
func StringToJSON(val interface{}) interface{} {
	str := val.(string)
	if strings.HasPrefix(str, "{") && strings.HasSuffix(str, "}") {
		var parameter interface{}
		_ = json.Unmarshal([]byte(str), &parameter)
		return parameter
	} else {
		var parameter []interface{}
		_ = json.Unmarshal([]byte(str), &parameter)
		return parameter
	}
}

// tool-获取树状数组
func GetTreeArray(list []map[string]interface{}, pid int64, itemprefix string) List {
	childs := ToolFar(list, pid) //获取pid下的所有数据
	var chridnum List
	if childs != nil {
		var number int = 1
		var total int = len(childs)
		for _, v := range childs {
			j := ""
			k := ""
			if number == total {
				j += "└"
				k = ""
				if itemprefix != "" {
					k = "&nbsp;"
				}

			} else {
				j += "├"
				k = ""
				if itemprefix != "" {
					k = "│"
				}
			}
			spacer := ""
			if itemprefix != "" {
				spacer = itemprefix + j
			}
			v["spacer"] = spacer
			children := GetTreeArray(list, Int64(v["id"]), itemprefix+k+"&nbsp;")
			if children != nil {
				v["children"] = children
			} else {
				v["children"] = Slice{}
			}
			chridnum = append(chridnum, v)
			number++
		}
	}
	return chridnum
}

// 将getTreeArray的结果返回为二维数组
func GetTreeToList(list []Map, field string) []Map {
	var midleArr []Map
	for _, v := range list {
		var children []Map
		if childrendata, ok := v["children"]; ok && childrendata != nil {
			switch childrendata := childrendata.(type) {
			case []interface{}:
				for _, cv := range childrendata {
					children = append(children, cv.(Map))
				}
			case []Map:
				children = childrendata
			}
		} else {
			children = nil
		}
		delete(v, "children")
		v[field+"_txt"] = fmt.Sprintf("%v %v", v["spacer"], v[field+""])
		if _, ok := v["id"]; ok {
			midleArr = append(midleArr, v)
		}
		if len(children) > 0 {
			newarr := GetTreeToList(children, field)
			midleArr = ArrayMerge_x(midleArr, newarr)
		}
	}
	return midleArr
}

// 数组拼接
func ArrayMerge_x(ss ...[]Map) []Map {
	n := 0
	for _, v := range ss {
		n += len(v)
	}
	s := make([]Map, 0, n)
	for _, v := range ss {
		s = append(s, v...)
	}
	return s
}

// base_tool-获取pid下所有数组
func ToolFar(data List, pid int64) List {
	var mapString List
	for _, v := range data {
		if Int64(v["pid"]) == pid {
			mapString = append(mapString, v)
		}
	}
	return mapString
}

// 去重
func UniqueArr(datas []interface{}) []interface{} {
	d := make([]interface{}, 0)
	tempMap := make(map[int]bool, len(datas))
	for _, v := range datas { // 以值作为键名
		keyv := Int(v)
		if tempMap[keyv] == false {
			tempMap[keyv] = true
			d = append(d, v)
		}
	}
	return d
}

// 合并数组-interface
func MergeArr_interface(a, b []interface{}) []interface{} {
	var arr []interface{}
	for _, i := range a {
		arr = append(arr, i)
	}
	for _, j := range b {
		arr = append(arr, j)
	}
	return arr
}

// 将带有逗号的数组中字符串差分合并为数组
func ArraymoreMerge(data []*gvar.Var) []interface{} {
	var rule_ids_arr []interface{}
	for _, mainv := range data {
		ids_arr := strings.Split(mainv.String(), `,`)
		for _, intv := range ids_arr {
			rule_ids_arr = append(rule_ids_arr, intv)
		}
	}
	return rule_ids_arr
}

// 获取后台菜单子树结构
func GetMenuChildrenArray(pdata List, parent_id int64, pid_file string) List {
	var returnList List
	for _, v := range pdata {
		if Int64(v[pid_file]) == parent_id {
			children := GetMenuChildrenArray(pdata, Int64(v["id"]), pid_file)
			if children != nil {
				v["children"] = gvar.New(children)
			}
			returnList = append(returnList, v)
		}
	}
	return returnList
}

// 日期时间转时间戳
// timetype时间格式类型  datetime=日期时间 datesecond=日期时间秒date=日期
func StringTimestamp(timeLayout, timetype string) int64 {
	timetpl := "2006-01-02 15:04:05"
	if timetype == "date" {
		timetpl = "2006-01-02"
	} else if timetype == "datetime" {
		timetpl = "2006-01-02 15:04"
	}
	times, _ := time.ParseInLocation(timetpl, timeLayout, time.Local)
	timeUnix := times.Unix()
	return timeUnix
}

// 时间戳格式化为日期字符串
// timetype时间格式类型 date=日期 datetime=日期时间 datesecond=日期时间秒
func TimestampString(timedata interface{}, timetype string) string {
	timetpl := "2006-01-02 15:04:05"
	if timetype == "date" {
		timetpl = "2006-01-02"
	} else if timetype == "datetime" {
		timetpl = "2006-01-02 15:04"
	}
	return time.Unix(timedata.(int64), 0).Format(timetpl)
}

// 判断字符串是否包含
func StrContains(str, filed string) bool {
	return strings.Contains(str, filed)
}

// 把字符串打散为数组
func SplitAndStr(str, step string) []string {
	return strings.Split(str, step)
}

// 把数组转字符串,号分隔
func ArrayToStr(data interface{}, step string) string {
	if data != nil && data != "" {
		data_arr := data.([]interface{})
		var str_arr = make([]string, len(data_arr))
		for k, v := range data_arr {
			str_arr[k] = fmt.Sprintf("%v", v)
		}
		return strings.Join(str_arr, step)
	} else {
		return ""
	}
}

// 判断字符串是否在一个数组中
func StrInArray(target string, str_array []string) bool {
	for _, element := range str_array {
		if target == element {
			return true
		}
	}
	return false
}

// 隐藏手机号等敏感信息用*替换展示
func HideStrInfo(strtype, val string) string {
	if val == "" {
		return ""
	}
	switch strtype {
	case "email":
		var arr = strings.Split(val, "@")
		var star = ""
		if len(arr[0]) <= 3 {
			star = "*"
			arr[0] = gstr.SubStr(arr[0], 0, len(arr[0])) + star
		} else {
			star = "***"
			arr[0] = gstr.SubStr(arr[0], 0, 1) + star + gstr.SubStr(arr[0], len(arr[0])-1, 1)
		}
		return arr[0] + "@" + arr[1]
	case "mobile":
		if len(val) <= 10 {
			return val
		}
		return val[:3] + "****" + val[len(val)-4:]
	}
	return ""
}

func ResData(r *http.Request, data any) error {
	// 处理GET请求的查询参数
	if r.Method == http.MethodGet {
		return httpx.ParseForm(r, data)
	}

	// 检查是否是 multipart/form-data 请求（文件上传）
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			return errors.New("解析表单数据失败")
		}
		return parseMultipartForm(r, data)
	}

	//手动解析JSON
	body, err := io.ReadAll(r.Body)
	if err != nil {
		return errors.New("读取请求失败")
	}
	defer r.Body.Close()
	if string(body) == "" {
		return nil
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return errors.New("请求参数格式不合法,请核对参数格式")
	}
	// 设置结构体字段的默认值
	setDefaultValues(data)
	return nil
}

func parseMultipartForm(r *http.Request, data any) error {
	val := reflect.ValueOf(data)
	if val.Kind() != reflect.Ptr {
		return errors.New("data must be a pointer")
	}
	val = val.Elem()
	if val.Kind() != reflect.Struct {
		return errors.New("data must be a pointer to struct")
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		if !field.CanSet() {
			continue
		}

		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" {
			continue
		}
		tagName := strings.Split(jsonTag, ",")[0]
		if tagName == "" || tagName == "-" {
			continue
		}

		formValue := r.FormValue(tagName)
		if formValue == "" {
			continue
		}

		if err := setFieldValueFromString(field, formValue); err != nil {
			return fmt.Errorf("字段 %s 值 %s 格式错误: %v", tagName, formValue, err)
		}
	}

	setDefaultValues(data)
	return nil
}

func setFieldValueFromString(field reflect.Value, value string) error {
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, err := strconv.ParseInt(value, 10, 64); err != nil {
			return err
		} else {
			field.SetInt(val)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(value, 10, 64); err != nil {
			return err
		} else {
			field.SetUint(val)
		}
	case reflect.Float32, reflect.Float64:
		if val, err := strconv.ParseFloat(value, 64); err != nil {
			return err
		} else {
			field.SetFloat(val)
		}
	case reflect.Bool:
		if val, err := strconv.ParseBool(value); err != nil {
			return err
		} else {
			field.SetBool(val)
		}
	case reflect.Ptr:
		ptr := reflect.New(field.Type().Elem())
		if err := setFieldValueFromString(ptr.Elem(), value); err != nil {
			return err
		}
		field.Set(ptr)
	default:
		return fmt.Errorf("unsupported field type: %s", field.Kind())
	}
	return nil
}

// setDefaultValues 设置结构体字段的默认值
func setDefaultValues(data any) {
	val := reflect.ValueOf(data)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	if val.Kind() != reflect.Struct {
		return
	}

	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)

		// 检查字段是否可设置
		if !field.CanSet() {
			continue
		}

		// 获取JSON标签
		jsonTag := fieldType.Tag.Get("json")
		if jsonTag == "" {
			continue
		}

		// 解析JSON标签中的default参数
		tagParts := strings.Split(jsonTag, ",")
		for _, part := range tagParts {
			if strings.HasPrefix(part, "default=") {
				defaultValue := strings.TrimPrefix(part, "default=")
				setFieldDefaultValue(field, defaultValue)
				break
			}
		}
	}
}

// setFieldDefaultValue 根据字段类型设置默认值
func setFieldDefaultValue(field reflect.Value, defaultValue string) {
	// 如果字段已经有值（非零值），则不设置默认值
	if !field.IsZero() {
		return
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(defaultValue)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if val, err := strconv.ParseInt(defaultValue, 10, 64); err == nil {
			field.SetInt(val)
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if val, err := strconv.ParseUint(defaultValue, 10, 64); err == nil {
			field.SetUint(val)
		}
	case reflect.Float32, reflect.Float64:
		if val, err := strconv.ParseFloat(defaultValue, 64); err == nil {
			field.SetFloat(val)
		}
	case reflect.Bool:
		if val, err := strconv.ParseBool(defaultValue); err == nil {
			field.SetBool(val)
		}
	}
}
