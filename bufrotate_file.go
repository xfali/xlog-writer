// Copyright (C) 2019-2020, Xiongfa Li.
// @author xiongfa.li
// @version V1.0
// Description:

package writer

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

func NewBufferedRotateFileWriter(f *BufferedRotateFile, config ...Config) io.WriteCloser {
	if f == nil {
		return nil
	}

	conf := defaultConfig
	if len(config) > 0 {
		conf = config[0]
		if conf.FlushInterval == 0 {
			conf.FlushInterval = FlushTime
		}
		if conf.BufferSize == 0 {
			conf.BufferSize = BufferSize
		}
		if conf.FlushSize == 0 {
			conf.FlushSize = FlushSize
		}
	}
	err := f.Open(conf)
	if err != nil {
		return nil
	}

	return f
}

type BufferedRotateFile struct {
	//文件路径
	Path string
	// 文件的大小阈值
	MaxFileSize int64
	// 滚动频率
	RotateFrequency RotateFrequency
	// 滚动文件处理
	RotateFunc func(dir string, name string, files ...string) error

	// 滚动的时间格式
	timeFormat string
	// 滚动的时间间隔
	rotateDuration time.Duration

	stopChan chan struct{}
	logChan  chan []byte
	block    bool
	wait     sync.WaitGroup
	once     sync.Once

	timer      *time.Timer
	fileName   string
	dir        string
	file       *os.File
	curSize    int64
	part       int
	curTimeStr string

	flushSize int64
	buf       *bytes.Buffer
}

func (f *BufferedRotateFile) Open(conf Config) error {
	var logChan chan []byte
	// Channel without buffer
	if conf.BufferSize <= 0 {
		logChan = make(chan []byte)
	} else {
		logChan = make(chan []byte, conf.BufferSize)
	}
	f.block = conf.Block
	f.logChan = logChan
	f.stopChan = make(chan struct{})

	if f.MaxFileSize == 0 {
		// no limit
		f.MaxFileSize = math.MaxInt64
	}
	if f.timeFormat == "" {
		f.timeFormat = "2006-01-02"
	}
	dir := filepath.Dir(f.Path)
	_, err := os.Stat(dir)
	if err != nil {
		err = os.Mkdir(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	f.flushSize = conf.FlushSize
	f.buf = bytes.NewBuffer(nil)
	f.dir = dir
	f.fileName = filepath.Base(f.Path)

	f.file, err = os.OpenFile(f.Path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	info, err := f.file.Stat()
	if err != nil {
		return err
	}
	f.curSize = info.Size()
	if f.RotateFrequency != RotateNone {
		f.setFrequency(f.RotateFrequency)
		f.setTimer()
	}
	err = f.calcPart()
	if err == nil {
		f.wait.Add(1)
		go func() {
			ticker := time.NewTicker(conf.FlushInterval)
			defer ticker.Stop()
			defer f.wait.Done()
			defer func() {
				size := len(f.logChan)
				for i := 0; i < size; i++ {
					_, _ = f.tryWrite(<-f.logChan)
				}
				_, _ = f.writeFile()
			}()
			for {
				select {
				case <-f.stopChan:
					return
				case d, ok := <-f.logChan:
					if ok {
						_, _ = f.tryWrite(d)
					}
				case <-ticker.C:
					_, _ = f.writeFile()
				}
				select {
				case <-f.stopChan:
					return
				case <-ticker.C:
					_, _ = f.writeFile()
				default:
					_ = f.tryRotateByTime()
				}
			}
		}()
	}
	return err
}

func (f *BufferedRotateFile) setTimer() {
	f.curTimeStr = time.Now().Format(f.timeFormat)
	t := f.nextTime()
	duration := t.Sub(time.Now())
	if duration < 0 {
		duration = 1
	}
	f.timer = time.NewTimer(duration)
}

func (f *BufferedRotateFile) Write(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	if f.block {
		f.logChan <- data
		return len(data), nil
	} else {
		select {
		case f.logChan <- data:
			return len(data), nil
		default:
			return 0, errors.New("write log failed ")
		}
	}
}

func (f *BufferedRotateFile) tryRotateByTime() error {
	if f.timer != nil {
		select {
		case <-f.timer.C:
			//n, err := f.writeFile()
			//if err != nil {
			//	return n, err
			//}
			err := f.rotateByTime()
			f.setTimer()
			if err != nil {
				return err
			}
		default:
		}
	}
	return nil
}

func (f *BufferedRotateFile) tryWrite(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}

	n, berr := f.buf.Write(data)

	if int64(f.buf.Len()) >= f.flushSize {
		_, err := f.writeFile()
		return n, err
	}
	return n, berr
}

func (f *BufferedRotateFile) writeFile() (int, error) {
	if f.file == nil {
		return 0, errors.New("file not opened. ")
	}
	err := f.tryRotateByTime()
	if err != nil {
		return 0, err
	}
	if f.buf.Len() == 0 {
		return 0, nil
	}
	defer f.buf.Reset()
	//fmt.Println("flush: ", f.buf.String())
	n, err := f.file.Write(f.buf.Bytes())
	f.curSize += int64(n)
	if err != nil {
		return n, err
	}
	if f.curSize >= f.MaxFileSize {
		err := f.rotatePart()
		if err != nil {
			return n, err
		}
	}
	return n, err
}

func (f *BufferedRotateFile) rotateByTime() error {
	//err := f.changeFile(fmt.Sprintf("%s-%s", f.curTimeStr, f.fileName))
	err := f.rotatePart()
	if err != nil {
		return err
	}
	if f.RotateFunc != nil {
		oldTimeStr := f.curTimeStr
		files, err := ioutil.ReadDir(f.dir)
		if err != nil {
			return err
		}
		var partsFiles []string
		for _, v := range files {
			i := strings.Index(v.Name(), oldTimeStr)
			if i != -1 {
				partsFiles = append(partsFiles, filepath.Join(f.dir, v.Name()))
			}
		}
		if len(partsFiles) > 0 {
			err = f.RotateFunc(f.dir, oldTimeStr+"-"+f.fileName, partsFiles...)
			if err != nil {
				return err
			}
		}
	}

	f.curTimeStr = time.Now().Format(f.timeFormat)
	f.part = 0
	return nil
}

func (f *BufferedRotateFile) calcPart() error {
	files, err := ioutil.ReadDir(f.dir)
	if err != nil {
		return err
	}
	part := 0
	prefix := ""
	if f.curTimeStr == "" {
		prefix = "part"
	} else {
		prefix = f.curTimeStr + "-part"
	}
	for _, v := range files {
		i := strings.Index(v.Name(), prefix)
		if i != -1 {
			part++
		}
	}
	f.part = part
	return nil
}

func (f *BufferedRotateFile) rotatePart() error {
	if f.curTimeStr == "" {
		err := f.changeFile(fmt.Sprintf("part%d-%s", f.part, f.fileName))
		f.part++
		return err
	} else {
		err := f.changeFile(fmt.Sprintf("%s-part%d-%s", f.curTimeStr, f.part, f.fileName))
		f.part++
		return err
	}
}

func (f *BufferedRotateFile) changeFile(filename string) error {
	err := f.file.Close()
	if err != nil {
		return err
	}
	err = os.Rename(filepath.Join(f.dir, f.fileName), filepath.Join(f.dir, filename))
	if err != nil {
		return err
	}
	f.file, err = os.OpenFile(filepath.Join(f.dir, f.fileName), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}
	f.curSize = 0
	return nil
}

func (f *BufferedRotateFile) nextTime() time.Time {
	timeStr := time.Now().Format(f.timeFormat)
	t, _ := time.ParseInLocation(f.timeFormat, timeStr, time.Local)
	return t.Add(f.rotateDuration)
}

func (f *BufferedRotateFile) Close() error {
	f.once.Do(func() {
		close(f.stopChan)
		f.wait.Wait()

		if f.timer != nil {
			f.timer.Stop()
		}
		if f.file != nil {
			f.file.Close()
		}
	})
	return nil
}

func (f *BufferedRotateFile) setFrequency(frequency RotateFrequency) {
	interval := frequency / RotateEveryDay
	if interval > 0 {
		f.rotateDuration = interval * RotateEveryDay
		f.timeFormat = "2006-01-02"
		return
	}

	interval = frequency / RotateEveryHour
	if interval > 0 {
		f.rotateDuration = interval * RotateEveryHour
		f.timeFormat = "2006-01-02-15"
		return
	}

	interval = frequency / RotateEveryMinute
	if interval > 0 {
		f.rotateDuration = interval * RotateEveryMinute
		f.timeFormat = "2006-01-02-15-04"
		return
	}

	interval = frequency / RotateEverySecond
	if interval > 0 {
		f.rotateDuration = interval * RotateEverySecond
		f.timeFormat = "2006-01-02-15-04-05"
		return
	}
}
