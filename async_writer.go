// Copyright (C) 2019, Xiongfa Li.
// All right reserved.
// @author xiongfa.li
// @version V1.0
// Description:

package writer

import (
	"errors"
	"io"
	"sync"
)

const (
	BufferSize = 10240
)

type Closer func() error

type AsyncLogWriter struct {
	stopChan chan struct{}
	logChan  chan []byte
	w        io.Writer
	block    bool
	wait     sync.WaitGroup
	once     sync.Once
}

// 异步写的Writer，本身Write、Close方法线程安全，参数WriteCloser可以非线程安全
// Param： w - 实际写入的Writer, bufSize - 接收的最大长度, block - 如果为true，则当超出bufSize大小时Write方法阻塞，否则返回error
func NewAsyncWriter(w io.Writer, closer Closer, bufSize int, block bool) *AsyncLogWriter {
	var logChan chan []byte
	// Channel without buffer
	if bufSize <= 0 {
		logChan = make(chan []byte)
	} else {
		logChan = make(chan []byte, bufSize)
	}
	l := AsyncLogWriter{
		stopChan: make(chan struct{}),
		logChan:  logChan,
		w:        w,
		block:    block,
	}
	l.wait.Add(1)

	go func() {
		defer l.wait.Done()
		if closer != nil {
			defer closer()
		}
		for {
			select {
			case <-l.stopChan:
				return
			case d, ok := <-l.logChan:
				if ok {
					l.writeLog(d)
				}
			}
		}
	}()
	return &l
}

func (w *AsyncLogWriter) writeLog(data []byte) {
	if w.w != nil {
		w.w.Write(data)
	}
}

func (w *AsyncLogWriter) Close() error {
	w.once.Do(func() {
		close(w.stopChan)
		w.wait.Wait()
	})
	return nil
}

func (w *AsyncLogWriter) Write(data []byte) (n int, err error) {
	if len(data) == 0 {
		return 0, nil
	}

	if w.block {
		w.logChan <- data
		return len(data), nil
	} else {
		select {
		case w.logChan <- data:
			return len(data), nil
		default:
			return 0, errors.New("write log failed ")
		}
	}
}
