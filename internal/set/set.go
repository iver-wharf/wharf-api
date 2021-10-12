package set

import (
	"fmt"
	"strings"
)

type emptyStruct struct{}

func goString(set interface{}, values []interface{}) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%T{", set)
	for i, value := range values {
		if i > 0 {
			sb.WriteString(", ")
		}
		fmt.Fprintf(&sb, "%#v", value)
	}
	sb.WriteByte('}')
	return sb.String()
}
