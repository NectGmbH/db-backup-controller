package main

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/m1ome/randstr"
)

func main() {
	start := time.Now()

	nSamples := mustIntArg(1)
	for i := 0; i < nSamples; i++ {
		fmt.Printf("INSERT INTO kv(key,value) VALUES('hd%s', '%s');\n", randstr.GetString(10), randstr.GetString(mustIntArg(2)))

		percDone := float64(i+1) / float64(nSamples)
		if i%1000 == 0 {
			log.Printf("Progress: %7d / %7d generated: ETF %s", i+1, nSamples, time.Duration(float64(time.Since(start))/percDone)-time.Since(start))
		}
	}
}

func mustIntArg(idx int) int {
	if len(os.Args) <= idx {
		panic(fmt.Errorf("too few args"))
	}
	v, err := strconv.Atoi(os.Args[idx])
	if err != nil {
		panic(err)
	}
	return v
}
