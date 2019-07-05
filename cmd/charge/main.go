package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"sync/atomic"
	"time"

	"github.com/combaine/custom-load-test/payload"
	"github.com/jmcvetta/randutil"
	"github.com/vmihailenco/msgpack"
	"google.golang.org/grpc"
)

//go:generate msgp

// PluginCfg ...
type PluginCfg map[string]interface{}

var loadFactor int32 = 1

func main() {
	var c = make(chan int, 1)
	var block = make(chan bool)
	runLoaders(c)
	go runReporter(c, block)
	// disable input buffering
	exec.Command("stty", "-F", "/dev/tty", "cbreak", "min", "1").Run()

	var b = make([]byte, 1)
	for {
		os.Stdin.Read(b)
		block <- true
	}
}

func runReporter(c chan int, block chan bool) {
	var step int32 = 1
	var count int
	var i float64
	var n float64
	var tm = time.Now()
	var spent time.Duration
	fmt.Printf("Step %d ", step)
	for {
		select {
		case count = <-c:
			n += float64(count)
		case <-block:
			spent += time.Now().Sub(tm)
			<-block
			tm = time.Now()
		}
		if i > 50 {
			step++
			spent += time.Now().Sub(tm)
			tm = time.Now()
			secs := float64(spent) / float64(time.Second)
			fmt.Printf("p(%.0f):a(%.3f)/s in %.3f \n", i*n/secs, i/secs, secs)
			switch step {
			case 10, 30:
				atomic.StoreInt32(&loadFactor, 5)
				fmt.Printf("Current load factor: %dx\n", atomic.LoadInt32(&loadFactor))
			case 20, 40:
				atomic.StoreInt32(&loadFactor, 1)
				fmt.Printf("Current load factor: %dx\n", atomic.LoadInt32(&loadFactor))
			}
			fmt.Printf("Step %d ", step)

			i = 0
			n = 0
			spent = 0
		}
		fmt.Print(".")
		i++
	}
}

func runLoaders(c chan int) {
	localAddresses := []string{
		"[::1]", "127.0.0.1", "127.0.0.2", "127.0.0.3",
		"127.0.0.4", "127.0.0.5", "127.0.0.6", "127.0.0.7"}
	var connList []*grpc.ClientConn
	options := []grpc.DialOption{
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(1024*1024*128),
			grpc.MaxCallRecvMsgSize(1024*1024*128),
		),
	}
	for _, ip := range localAddresses {
		addr := fmt.Sprintf("%s:50051", ip)
		conn, err := grpc.Dial(addr, options...)
		if err != nil {
			log.Fatalf("Failed to dial grpc server: %v", err)
		}
		connList = append(connList, conn)
	}
	var tokens = make(chan int, 16)
	go func() {
		for {
			for _, conn := range connList {
				tokens <- 0
				go func(cc *grpc.ClientConn) {
					fire(cc, c)
					defer func() { <-tokens }()
				}(conn)
			}
		}

	}()

}

func fire(conn *grpc.ClientConn, ch chan int) {
	c := payload.NewCustomAggregatorClient(conn)
	choices := []randutil.Choice{
		randutil.Choice{Weight: 60, Item: 20},
		randutil.Choice{Weight: 20, Item: 50},
		randutil.Choice{Weight: 15, Item: 200},
		randutil.Choice{Weight: 5, Item: 400},
	}

	result, err := randutil.WeightedChoice(choices)
	if err != nil {
		log.Fatalf("Error while get random weighted chose: %v", err)
	}

	currentLoadFactor := int(atomic.LoadInt32(&loadFactor))
	hosts := 10 + rand.Intn(currentLoadFactor*result.Item.(int))
	aggHosts := doParsing(c, hosts)
	respBytes := doAggregate(c, aggHosts)

	var resp PluginCfg
	if err := msgpack.Unmarshal(respBytes, &resp); err != nil {
		log.Fatalf("Failed to unmarshal parsing result: %v", err)
	}
	ch <- hosts
}

func getConfig() []byte {
	cfg := PluginCfg{
		"some":       1,
		"key":        "here",
		"timings_is": "_time",
	}

	cfgBytes, err := cfg.MarshalMsg(nil)
	if err != nil {
		log.Fatalf("Marshal config object")
	}
	return cfgBytes

}

func doParsing(c payload.CustomAggregatorClient, count int) [][]byte {

	var aggHosts [][]byte
	currentLoadFactor := int(atomic.LoadInt32(&loadFactor))
	for i := 0; i < count; i++ {
		gauges := 10 + rand.Intn(currentLoadFactor*40)
		timings := 10 + rand.Intn(currentLoadFactor*20)
		text := payload.GenPayload(gauges, timings)
		req := &payload.AggregateHostRequest{
			Task: &payload.Task{
				Id:     fmt.Sprintf("id%v", i),
				Config: getConfig(),
			},
			ClassName: "Multimetrics",
			Payload:   text,
		}
		res, err := c.AggregateHost(context.Background(), req)
		if err != nil {
			log.Fatalf("Failed to do AggregateHost: %v", err)
		}
		aggHosts = append(aggHosts, res.GetResult())
	}
	return aggHosts
}

func doAggregate(c payload.CustomAggregatorClient, data [][]byte) []byte {
	req := &payload.AggregateGroupRequest{
		Task: &payload.Task{
			Id:     "idGroup",
			Config: getConfig(),
		},
		ClassName: "Multimetrics",
		Payload:   data,
	}
	res, err := c.AggregateGroup(context.Background(), req)
	if err != nil {
		log.Fatalf("Failed to do AggregateHost: %v", err)
	}
	return res.GetResult()
}
