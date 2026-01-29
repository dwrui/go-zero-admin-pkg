package gstr

// List2 将字符串 `str` 以 `delimiter` 分隔，并返回结果的前两个部分。
func List2(str, delimiter string) (part1, part2 string) {
	return doList2(delimiter, Split(str, delimiter))
}

// ListAndTrim2 将字符串 `str` 以 `delimiter` 分隔，并返回结果的前两个部分，同时去掉每个部分的首尾空格。
func ListAndTrim2(str, delimiter string) (part1, part2 string) {
	return doList2(delimiter, SplitAndTrim(str, delimiter))
}

func doList2(delimiter string, array []string) (part1, part2 string) {
	switch len(array) {
	case 0:
		return "", ""
	case 1:
		return array[0], ""
	case 2:
		return array[0], array[1]
	default:
		return array[0], Join(array[1:], delimiter)
	}
}

// List3 将字符串 `str` 以 `delimiter` 分隔，并返回结果的前三个部分。
func List3(str, delimiter string) (part1, part2, part3 string) {
	return doList3(delimiter, Split(str, delimiter))
}

// ListAndTrim3 将字符串 `str` 以 `delimiter` 分隔，并返回结果的前三个部分，同时去掉每个部分的首尾空格。
func ListAndTrim3(str, delimiter string) (part1, part2, part3 string) {
	return doList3(delimiter, SplitAndTrim(str, delimiter))
}

func doList3(delimiter string, array []string) (part1, part2, part3 string) {
	switch len(array) {
	case 0:
		return "", "", ""
	case 1:
		return array[0], "", ""
	case 2:
		return array[0], array[1], ""
	case 3:
		return array[0], array[1], array[2]
	default:
		return array[0], array[1], Join(array[2:], delimiter)
	}
}
