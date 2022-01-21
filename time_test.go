// Copyright (C) 2019-2020, Xiongfa Li.
// @author xiongfa.li
// @version V1.0
// Description:

package writer

import (
	"testing"
	"time"
)

func TestRotateTime(t *testing.T) {
	f := RotateFile{}
	f.setFrequency(40 * RotateEveryDay)
	t.Log("40 day")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(RotateEveryDay)
	t.Log("1 day")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(25 * RotateEveryHour)
	t.Log("25 hour")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(RotateEveryHour)
	t.Log("1 hour")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(70 * RotateEveryMinute)
	t.Log("70 minute")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(RotateEveryMinute)
	t.Log("1 minute")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(70 * RotateEverySecond)
	t.Log("70 second")
	t.Log(time.Now())
	t.Log(f.nextTime())

	f.setFrequency(RotateEverySecond)
	t.Log("1 second")
	t.Log(time.Now())
	t.Log(f.nextTime())
}
