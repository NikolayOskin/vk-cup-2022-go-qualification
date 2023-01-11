package main

import (
	"bytes"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Advertise - function starts the entire pipeline
func Advertise(freeFlowJobs ...job) {
	inCh := make(chan interface{})
	outCh := make(chan interface{})

	wg := sync.WaitGroup{}
	wg.Add(len(freeFlowJobs))

	for _, freeFlowJob := range freeFlowJobs {
		go func(j job, in, out chan interface{}) {
			j(in, out)
			close(out)
			wg.Done()
		}(freeFlowJob, inCh, outCh)

		// to make a pipeline, assign out channel of scheduled goroutine to in channel of next goroutine
		inCh = outCh
		outCh = make(chan interface{})
	}

	wg.Wait()
}

func GetProfile(in, out chan interface{}) {
	type predicted struct {
		profileID     string
		fastPredicted string
		leftPart      string
		rightPart     string
	}

	predictedList := make([]*predicted, 0, MaxInputDataLen)

	// FastPredict func need to be called synchronously due to penalties
	// iterate and store FastPredict result in struct
	for val := range in {
		profileID := val.(int)
		profileIDStr := strconv.Itoa(profileID)
		p := &predicted{
			profileID:     profileIDStr,
			fastPredicted: FastPredict(profileIDStr),
		}
		predictedList = append(predictedList, p)
	}

	// SlowPredict func has to be called in goroutines due to long execution of each call (1 sec)
	wg := sync.WaitGroup{}
	wg.Add(len(predictedList) * 2)

	for _, p := range predictedList {
		go func(p *predicted) {
			p.leftPart = SlowPredict(p.profileID)
			wg.Done()
		}(p)
		go func(p *predicted) {
			p.rightPart = SlowPredict(p.fastPredicted)
			wg.Done()
		}(p)
	}
	wg.Wait()

	for _, p := range predictedList {
		out <- p.leftPart + "-" + p.rightPart
	}
}

func GetGroup(in, out chan interface{}) {
	type profile struct {
		profile string

		// each profile has exactly 6 groups
		groups [6]string
	}

	// storing input results in slice (to keep the order),
	// result of SlowPredict execution for each group will be stored in array
	profiles := make([]*profile, 0, MaxInputDataLen)
	for val := range in {
		p := profile{
			profile: val.(string),
		}
		profiles = append(profiles, &p)
	}

	// for each profile run 6 goroutines (1 goroutine per group) and call SlowPredict
	wg := sync.WaitGroup{}
	wg.Add(len(profiles) * 6)

	for _, p := range profiles {
		for i := range p.groups {
			go func(p *profile, i int) {
				p.groups[i] = SlowPredict(strconv.Itoa(i) + p.profile)
				wg.Done()
			}(p, i)
		}
	}
	wg.Wait()

	for _, p := range profiles {
		var b bytes.Buffer
		for _, group := range p.groups {
			b.WriteString(group)
		}
		out <- b.String()
	}
}

func ConcatProfiles(in, out chan interface{}) {
	profiles := make([]string, 0, MaxInputDataLen)

	for val := range in {
		valStr := val.(string)
		profiles = append(profiles, valStr)
	}
	sort.Strings(profiles)

	out <- strings.Join(profiles, "_")
}
