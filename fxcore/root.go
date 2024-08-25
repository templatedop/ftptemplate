package fxcore

import (
	"path/filepath"
	"runtime"
)


func RootDir(skip int) string {
	_, file, _, _ := runtime.Caller(skip)

	return filepath.Join(filepath.Dir(file), "..")
}
