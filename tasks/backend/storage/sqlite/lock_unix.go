//go:build linux || darwin || freebsd || openbsd || netbsd || dragonfly

package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
)

func lockFile(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path+".lock",
		os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return nil, err
	}
	err = syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		f.Close()
		return nil, fmt.Errorf("数据文件已被其他进程占用: %w", err)
	}
	return func() { _ = f.Close() }, nil
}
