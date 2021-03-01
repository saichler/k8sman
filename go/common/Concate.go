package common

import (
	"bytes"
	utils "github.com/saichler/utils/golang"
	"reflect"
	"strconv"
)

func Con(args ...interface{}) string {
	if args == nil || len(args) == 0 {
		return ""
	}
	buff := bytes.Buffer{}
	for _, arg := range args {
		val := reflect.ValueOf(arg)
		if val.Kind() == reflect.String {
			buff.WriteString(val.String())
		} else if val.Kind() == reflect.Int {
			buff.WriteString(strconv.Itoa(int(val.Int())))
		} else {
			utils.Error("unsupported, yet, type:", val.Kind().String())
		}
	}
	return buff.String()
}
