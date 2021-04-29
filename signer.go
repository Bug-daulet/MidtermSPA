package main

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
)

var ExecutePipeline = func(jobs ...job) {
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, jobItem := range jobs {
		wg.Add(1)

		out := make(chan interface{})

		go func(jobFunc job, in chan interface{}, out chan interface{}, wg *sync.WaitGroup) {
			defer wg.Done()
			defer close(out)
			jobFunc(in, out)
		}(jobItem, in, out, wg)

		in = out
	}

	wg.Wait()
}

var SingleHash = func(in, out chan interface{}) {
	mu := &sync.Mutex{}
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)

		go func(mu *sync.Mutex, wg *sync.WaitGroup, out chan interface{}, inp interface{}) {
			defer wg.Done()

			data := strconv.Itoa(inp.(int))

			crc32 := make(chan string)

			go func(out chan string, data string) {
				out <- DataSignerCrc32(data)
			}(crc32, data)

			mu.Lock()
			md5 := DataSignerMd5(data)
			mu.Unlock()

			crc32Md5 := DataSignerCrc32(md5)
			s := <-crc32
			out <- s + "~" + crc32Md5
		}(mu, wg, out, data)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := &sync.WaitGroup{}
	for data := range in {
		wg.Add(1)
		go func(wg *sync.WaitGroup, in interface{}, out chan interface{}) {
			defer wg.Done()

			mu := &sync.Mutex{}
			crcWaitGroup := &sync.WaitGroup{}

			result := make([]string, 6)

			for i := 0; i < 6; i++ {
				crcWaitGroup.Add(1)

				go func(mu *sync.Mutex, crc *sync.WaitGroup, hashValue []string, input string, ind int) {

					defer crc.Done()

					data := DataSignerCrc32(input)

					mu.Lock()
					hashValue[ind] = data
					mu.Unlock()
				}(mu, crcWaitGroup, result, fmt.Sprint(i)+in.(string), i)
			}
			crcWaitGroup.Wait()

			mainResult := strings.Join(result, "")

			out <- mainResult
		}(wg, data.(string), out)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {

	var result []string

	for i := range in {
		result = append(result, i.(string))
	}

	sort.Strings(result)
	output := strings.Join(result, "_")
	out <- output
}