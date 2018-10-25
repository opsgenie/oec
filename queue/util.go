package queue


var convertStringListToMapMethod = convertStringListToMap
var convertListToMapMethod = convertListToMap

func convertStringListToMap(list []string ) map[string]struct{} {
	stringMap := make(map[string]struct{})
	for i := 0; i < len(list); i ++ {
		stringMap[list[i]] = struct{}{}
	}
	return stringMap
}

func convertListToMap(list []interface{}) map[interface{}]struct{} {
	interfaceMap := make(map[interface{}]struct{})
	for i := 0; i < len(list); i ++ {
		interfaceMap[list[i]] = struct{}{}
	}
	return interfaceMap
}

func Min(x, y int64) int64 {
	if x > y {
		return y
	}
	return x
}