package errors

import (
	"fmt"
	"runtime"
)

// Frame 堆栈帧
type Frame struct {
	File     string
	Line     int
	Function string
}

// String 返回堆栈帧的字符串表示
func (f Frame) String() string {
	return fmt.Sprintf("%s:%d %s", f.File, f.Line, f.Function)
}

// captureStack 捕获当前调用栈
func captureStack(skip int) []Frame {
	const maxDepth = 32
	var frames []Frame

	pcs := make([]uintptr, maxDepth)
	n := runtime.Callers(skip, pcs)

	if n == 0 {
		return frames
	}

	callersFrames := runtime.CallersFrames(pcs[:n])
	for {
		frame, more := callersFrames.Next()

		frames = append(frames, Frame{
			File:     frame.File,
			Line:     frame.Line,
			Function: frame.Function,
		})

		if !more {
			break
		}
	}

	return frames
}

// FormatStack 格式化堆栈信息
func FormatStack(frames []Frame) string {
	if len(frames) == 0 {
		return ""
	}

	result := "\nStack trace:\n"
	for i, frame := range frames {
		result += fmt.Sprintf("  %d. %s\n", i+1, frame.String())
	}
	return result
}