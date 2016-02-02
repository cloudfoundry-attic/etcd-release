package destiny

import (
	"fmt"
	"strings"
)

type IPRange string

func (r IPRange) IP(index int) string {
	parts := strings.Split(string(r), ".")
	return fmt.Sprintf("%s.%s.%s.%d", parts[0], parts[1], parts[2], index)
}

func (r IPRange) Range(start, end int) string {
	return fmt.Sprintf("%s-%s", r.IP(start), r.IP(end))
}
