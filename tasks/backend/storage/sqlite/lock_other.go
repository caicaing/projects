//go:build !linux && !darwin && !freebsd && !openbsd && !netbsd && !dragonfly

package sqlite

import "fmt"

func lockFile(string) (func(), error) {
	return nil, fmt.Errorf("当前支持 Linux、macOS 和 BSD 的文件锁")
}
