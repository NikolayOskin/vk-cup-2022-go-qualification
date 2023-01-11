package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"strconv"
	"sync/atomic"
	"testing"
	"time"
)

func TestCustom(t *testing.T) {
	var (
		DataSignerSalt     string = "" // на сервере будет другое значение
		OverLockCounter    uint32
		OverUnlockCounter  uint32
		SignerMd5Counter   uint32
		SignerCrc32Counter uint32
	)
	OverLock = func() {
		atomic.AddUint32(&OverLockCounter, 1)
		for {
			if swapped := atomic.CompareAndSwapUint32(&dataOverheat, 0, 1); !swapped {
				fmt.Println("OverheatLock happend")
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}
	OverUnlock = func() {
		atomic.AddUint32(&OverUnlockCounter, 1)
		for {
			if swapped := atomic.CompareAndSwapUint32(&dataOverheat, 1, 0); !swapped {
				fmt.Println("OverheatUnlock happend")
				time.Sleep(time.Second)
			} else {
				break
			}
		}
	}
	FastPredict = func(data string) string {
		atomic.AddUint32(&SignerMd5Counter, 1)
		OverLock()
		defer OverUnlock()
		data += DataSignerSalt
		dataHash := fmt.Sprintf("%x", md5.Sum([]byte(data)))
		time.Sleep(10 * time.Millisecond)
		return dataHash
	}
	SlowPredict = func(data string) string {
		atomic.AddUint32(&SignerCrc32Counter, 1)
		data += DataSignerSalt
		crcH := crc32.ChecksumIEEE([]byte(data))
		dataHash := strconv.FormatUint(uint64(crcH), 10)
		time.Sleep(time.Second)
		return dataHash
	}

	var inputData []int

	for i := 1; i <= 64; i++ {
		inputData = append(inputData, i)
	}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(GetProfile),
		job(GetGroup),
		job(ConcatProfiles),
		job(func(in, out chan interface{}) {
			<-in
		}),
	}

	start := time.Now()

	Advertise(hashSignJobs...)

	end := time.Since(start)

	expectedTime := 3 * time.Second

	if end > expectedTime {
		t.Errorf("execition too long\nGot: %s\nExpected: <%s", end, time.Second*3)
	}

	if int(OverLockCounter) != len(inputData) ||
		int(OverUnlockCounter) != len(inputData) ||
		int(SignerMd5Counter) != len(inputData) ||
		int(SignerCrc32Counter) != len(inputData)*8 {
		t.Errorf("not enough hash-func calls")
	}

}
