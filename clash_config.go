package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"regexp"
)

func ClashHandler(config string, filterPattern string) string {
	resultMap := make(map[string]interface{})
	err := yaml.Unmarshal([]byte(config), &resultMap)
	if err != nil {
		fmt.Println(err.Error())
		return ""
	}
	v := resultMap["proxy-groups"]
	switch value := v.(type) {
	case nil:
		fmt.Println("is nil", "null")
	case string:
		fmt.Println("is string", value)
	case int:
		fmt.Println("is int", value)
	case float64:
		fmt.Println("is float64", value)
	case []interface{}:
		fmt.Println("is an array:")
		for _, u := range value {
			switch proxy := u.(type) {
			case map[interface{}]interface{}:
				for k1, v1 := range proxy {
					if k1 == "proxies" {
						switch proxies := v1.(type) {
						case []interface{}:
							deleteIndex := []int{}
							for i, proxyValue := range proxies {
								var s = proxyValue.(string)
								match, _ := regexp.MatchString(filterPattern, s)
								if match {
									// 指定删除位置
									fmt.Println("delete:", i, proxies[i])
									deleteIndex = append(deleteIndex, i)
								}
							}
							newValues := []interface{}{}
							for i, proxyValue := range proxies {
								if !contains(deleteIndex, i) {
									newValues = append(newValues, proxyValue)
								}
							}

							v1 = newValues
						}
						proxy["proxies"] = v1
						fmt.Println("v1", len(proxy["proxies"].([]interface{})), v1)
					}
				}
			default:
				fmt.Println("is unknown type", fmt.Sprintf("%T", v))
			}

		}
	case map[string]interface{}:
		fmt.Println("is an map:")
	default:
		fmt.Println("is unknown type", fmt.Sprintf("%T", v))
	}

	bytes, _ := yaml.Marshal(&resultMap)
	myString := string(bytes[:])
	fmt.Println(myString)
	return myString
}

func contains(s []int, e int) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}
