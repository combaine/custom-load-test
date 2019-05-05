package main

import (
	"context"
	"fmt"
	"log"

	"github.com/sakateka/custom-load/payload"
	"github.com/vmihailenco/msgpack"
	"google.golang.org/grpc"
)

//go:generate msgp

// PluginCfg ...
type PluginCfg map[string]interface{}

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		log.Fatalf("Failed to dial grpc server: %v", err)
	}
	c := payload.NewCustomAggregatorClient(conn)
	aggHosts := doParsing(c, 80)
	respBytes := doAggregate(c, aggHosts)

	var resp PluginCfg
	if err := msgpack.Unmarshal(respBytes, &resp); err != nil {
		log.Fatalf("Failed to unmarshal parsing result: %v", err)
	}
	fmt.Println(resp)
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
		text := payload.GenPayload(30, 15)
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
