package payload

import (
	"bytes"
	"fmt"
	"math/rand"
	"strconv"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// GenCounter ...
func GenCounter(c int) string {
	name := "name" + strconv.Itoa(c)
	line := name + " " + strconv.Itoa(rand.Intn(1000))
	return line
}

// GenTimings ...
func GenTimings(t int) string {
	var line bytes.Buffer
	line.WriteString("@name" + strconv.Itoa(t) + "_time" + " ")

	nValues := 10 + rand.Intn(100)
	for i := 0; i < nValues; i++ {
		val := rand.Float32()
		valCount := 1 + rand.Intn(100)
		line.WriteString(fmt.Sprintf(" %0.3f", val) + "@" + strconv.Itoa(valCount))
	}
	return line.String()
}

// GenPayload ...
func GenPayload(gauges int, timings int) string {
	var payload bytes.Buffer
	for i := 0; i < gauges; i++ {
		payload.WriteString(GenCounter(i))
		payload.WriteByte('\n')
	}

	for i := 0; i < timings; i++ {
		payload.WriteString(GenTimings(i))
		payload.WriteByte('\n')
	}
	return payload.String()
}
