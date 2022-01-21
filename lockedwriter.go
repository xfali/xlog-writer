// Copyright (C) 2019-2020, Xiongfa Li.
// @author xiongfa.li
// @version V1.0
// Description:

package writer

import (
	"io"
	"sync"
)

// Logging不会自动为输出的Writer加锁，如果需要加锁请使用这个封装工具：
// logging.SetOutPut(&writer.LockedWriter{w})
type LockedWriter struct {
	lock sync.Mutex
	W    io.Writer
}

func (lw *LockedWriter) Write(d []byte) (int, error) {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	return lw.W.Write(d)
}

type LockedWriteCloser struct {
	lock sync.Mutex
	W    io.WriteCloser
}

func (lw *LockedWriteCloser) Write(d []byte) (int, error) {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	return lw.W.Write(d)
}

func (lw *LockedWriteCloser) Close() error {
	lw.lock.Lock()
	defer lw.lock.Unlock()

	return lw.W.Close()
}
