// Copyright (C) 2019-2020, Xiongfa Li.
// @author xiongfa.li
// @version V1.0
// Description:

package test

import (
	"encoding/base64"
	"fmt"
	writer "github.com/xfali/xlog-writer"
	"math/rand"
	"os"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestAsyncWriter(t *testing.T) {
	w := writer.NewAsyncWriter(os.Stdout, nil, 10, true)
	var count int32 = 0
	wait := sync.WaitGroup{}
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			b := make([]byte, 10)
			for i := 0; i < 10; i++ {
				atomic.AddInt32(&count, 1)
				rand.Read(b)
				_, err := w.Write([]byte(strconv.Itoa(int(count)) + base64.StdEncoding.EncodeToString(b) + "\n"))
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	wait.Wait()
	t.Log(count)
	w.Close()
}

func TestAsyncBufWriter(t *testing.T) {
	w := writer.NewAsyncBufferWriter(os.Stdout, nil, writer.Config{
		FlushSize:     100,
		BufferSize:    10,
		FlushInterval: 1 * time.Millisecond,
		Block:         true,
	})
	var count int32 = 0
	wait := sync.WaitGroup{}
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			b := make([]byte, 10)
			for i := 0; i < 10; i++ {
				atomic.AddInt32(&count, 1)
				rand.Read(b)
				_, err := w.Write([]byte(strconv.Itoa(int(count)) + base64.StdEncoding.EncodeToString(b) + "\n"))
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	wait.Wait()
	w.Close()
	t.Log(count)
}

func TestRotateFile(t *testing.T) {
	w := writer.NewRotateFileWriter(&writer.RotateFile{
		Path: "./target/test.log",
	}, writer.Config{
		FlushSize:     100,
		BufferSize:    10,
		FlushInterval: 1 * time.Millisecond,
		Block:         true,
	})
	var count int32 = 0
	wait := sync.WaitGroup{}
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			b := make([]byte, 10)
			for i := 0; i < 10; i++ {
				atomic.AddInt32(&count, 1)
				rand.Read(b)
				_, err := w.Write([]byte(strconv.Itoa(int(count)) + base64.StdEncoding.EncodeToString(b) + "\n"))
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	wait.Wait()
	w.Close()
	t.Log(count)
}

func TestRotateFilePart(t *testing.T) {
	w := writer.NewRotateFileWriter(&writer.RotateFile{
		Path:        "./target/test.log",
		MaxFileSize: 10,
	}, writer.Config{
		FlushSize:     100,
		BufferSize:    10,
		FlushInterval: 1 * time.Millisecond,
		Block:         true,
	})
	var count int32 = 0
	wait := sync.WaitGroup{}
	wait.Add(20)
	for i := 0; i < 20; i++ {
		go func() {
			defer wait.Done()
			b := make([]byte, 10)
			for i := 0; i < 10; i++ {
				atomic.AddInt32(&count, 1)
				rand.Read(b)
				_, err := w.Write([]byte(strconv.Itoa(int(count)) + base64.StdEncoding.EncodeToString(b) + "\n"))
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	wait.Wait()
	w.Close()
	t.Log(count)
}

func TestRotateFilePartAndTime(t *testing.T) {
	w := writer.NewRotateFileWriter(&writer.RotateFile{
		Path:            "./target/test.log",
		MaxFileSize:     60,
		RotateFrequency: 1 * writer.RotateEverySecond,
	}, writer.Config{
		FlushSize:     10,
		BufferSize:    10,
		FlushInterval: 1 * time.Millisecond,
		Block:         true,
	})

	var count int32 = 0
	wait := sync.WaitGroup{}
	wait.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wait.Done()
			b := make([]byte, 10)
			for j := 0; j < 10; j++ {
				time.Sleep(300 * time.Millisecond)
				atomic.AddInt32(&count, 1)
				rand.Read(b)
				_, err := w.Write([]byte(strconv.Itoa(int(count)) + base64.StdEncoding.EncodeToString(b) + "\n"))
				if err != nil {
					t.Fatal(err)
				}
			}
		}()
	}

	wait.Wait()
	w.Close()
	t.Log(count)
}

func TestRotateFilePartAndTimeWithZip(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		w := writer.NewRotateFileWriter(&writer.RotateFile{
			Path:            "./target/test.log",
			MaxFileSize:     1000,
			RotateFrequency: writer.RotateEveryMinute,
			RotateFunc:      writer.ZipLogsAsync,
		}, writer.Config{
			FlushSize:     1000,
			BufferSize:    1024,
			FlushInterval: 50 * time.Millisecond,
			Block:         false,
		})

		for i:= 0; i<300; i++ {
			time.Sleep(300 * time.Millisecond)
			_, err := w.Write([]byte(fmt.Sprintf("[%d][%s]\n", i, time.Now().Format("2006-01-02-15-04-05"))))
			if err != nil {
				t.Fatal(err)
			}
		}
		time.Sleep(time.Second)
		w.Close()
	})

	t.Run("fast rotate", func(t *testing.T) {
		w := writer.NewRotateFileWriter(&writer.RotateFile{
			Path:            "./target/test.log",
			MaxFileSize:     10,
			RotateFrequency: writer.RotateEverySecond,
			RotateFunc:      writer.ZipLogsAsync,
		}, writer.Config{
			FlushSize:     100,
			BufferSize:    10,
			FlushInterval: 1 * time.Millisecond,
			Block:         true,
		})

		var count int32 = 0
		wait := sync.WaitGroup{}
		wait.Add(10)
		for i := 0; i < 10; i++ {
			go func() {
				defer wait.Done()
				b := make([]byte, 10)
				for j := 0; j < 10; j++ {
					time.Sleep(300 * time.Millisecond)
					atomic.AddInt32(&count, 1)
					rand.Read(b)
					_, err := w.Write([]byte(strconv.Itoa(int(count)) + base64.StdEncoding.EncodeToString(b) + "\n"))
					if err != nil {
						t.Fatal(err)
					}
				}
			}()
		}

		wait.Wait()
		w.Close()
		t.Log(count)
	})
}

func TestBufferedRotateFileWriterePartAndTimeWithZip(t *testing.T) {
	t.Run("normal", func(t *testing.T) {
		w := writer.NewBufferedRotateFileWriter(&writer.BufferedRotateFile{
			Path:            "./target/test.log",
			MaxFileSize:     1000,
			RotateFrequency: writer.RotateEveryMinute,
			RotateFunc:      writer.ZipLogsAsync,
		}, writer.Config{
			FlushSize:     1000,
			BufferSize:    1024,
			FlushInterval: 50 * time.Millisecond,
			Block:         false,
		})

		for i:= 0; i<300; i++ {
			time.Sleep(300 * time.Millisecond)
			_, err := w.Write([]byte(fmt.Sprintf("[%d][%s]\n", i, time.Now().Format("2006-01-02-15-04-05"))))
			if err != nil {
				t.Fatal(err)
			}
		}
		w.Close()
	})

	t.Run("file test", func(t *testing.T) {
		w := writer.NewBufferedRotateFileWriter(&writer.BufferedRotateFile{
			Path:            "./target/test.log",
			MaxFileSize:     10,
			RotateFrequency: writer.RotateEverySecond,
			RotateFunc:      writer.ZipLogsAsync,
		}, writer.Config{
			FlushSize:     100,
			BufferSize:    0,
			FlushInterval: 1 * time.Second,
			Block:         true,
		})

		var count int32 = 0
		wait := sync.WaitGroup{}
		wait.Add(10)
		buf := make([]byte, 102400)
		for i := 0; i < 10; i++ {
			go func() {
				defer wait.Done()
				b := make([]byte, 10)
				for j := 0; j < 10; j++ {
					time.Sleep(300 * time.Millisecond)
					v := atomic.AddInt32(&count, 1)
					rand.Read(b)
					_, err := w.Write([]byte(fmt.Sprintf("[%d][%s][%s]\n", v, time.Now().Format("2006-01-02-15-04-05"), base64.StdEncoding.EncodeToString(b))))
					_, err = w.Write([]byte(fmt.Sprintf("%s\n", string(buf))))
					if err != nil {
						t.Fatal(err)
					}
				}
			}()
		}

		wait.Wait()
		w.Close()
		t.Log(count)
	})
}
