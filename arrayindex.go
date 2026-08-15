package jq

import "strconv"

const maxArrayIndex int64 = (1 << 32) - 2

func parseArrayIndex(component string) (index int, ok bool) {
	if component == "0" {
		ok = true
		return
	}
	if component != "" {
		if component[0] >= '1' && component[0] <= '9' {
			var err error
			if index, err = strconv.Atoi(component); err == nil {
				if int64(index) <= maxArrayIndex {
					ok = true
					return
				}
			}
		}
	}
	index = 0
	return
}
