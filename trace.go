package gdht

import (
	"runtime"
	"fmt"
)

func GetDefaultTrace()string{
	return GetTrace(3,5)
}

func GetTrace(skip, total_line int) string {
	var trace string
	for i := skip; i < skip+total_line; i++ { // Skip the expected number of frames
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		trace = trace + fmt.Sprintf("%s:%d\n", file, line)
	}
	return trace
}
