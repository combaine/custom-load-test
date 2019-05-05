package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"time"

	"github.com/sakateka/custom-load/payload"
	"github.com/vmihailenco/msgpack"
	"google.golang.org/grpc"
)

//go:generate msgp

// PluginCfg ...
type PluginCfg map[string]interface{}

func main() {
	conn, err := grpc.Dial(
		"localhost:50051",
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallSendMsgSize(1024*1024*128),
			grpc.MaxCallRecvMsgSize(1024*1024*128),
		),
	)
	if err != nil {
		log.Fatalf("Failed to dial grpc server: %v", err)
	}
	var c = make(chan int, 12)
	var tokens = make(chan int, 12)
	go func() {
		for {
			tokens <- 0
			go func() {
				fire(conn, c)
				defer func() { <-tokens }()
			}()
		}

	}()
	var i float64
	var n float64
	var tm = time.Now()
	for {
		n += float64(<-c)
		i++
		if i > 60 {
			spent := time.Now().Sub(tm)
			tm = time.Now()
			secs := float64(spent / time.Second)
			fmt.Printf("p(%f):a(%f)/s in %s \n", i*n/secs, i/secs, spent)
			i = 0
			n = 0
		}
		fmt.Print(".")
	}
}

func fire(conn *grpc.ClientConn, ch chan int) {
	c := payload.NewCustomAggregatorClient(conn)
	hosts := 10 + rand.Intn(200)
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
	for i := 0; i < count; i++ {
		gauges := 20 + rand.Intn(50)
		timings := 20 + rand.Intn(30)
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
