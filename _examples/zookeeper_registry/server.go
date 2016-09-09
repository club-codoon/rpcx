package main

import (
	"flag"
	"time"

	"github.com/rcrowley/go-metrics"
	"github.com/smallnest/rpcx"
	"github.com/smallnest/rpcx/plugin"
)

type Args struct {
	A int `msg:"a"`
	B int `msg:"b"`
}

type Reply struct {
	C int `msg:"c"`
}

type Arith int

func (t *Arith) Mul(args *Args, reply *Reply) error {
	reply.C = args.A * args.B
	return nil
}

func (t *Arith) Error(args *Args, reply *Reply) error {
	panic("ERROR")
}

var addr = flag.String("s", "127.0.0.1:8972", "service address")
var zk = flag.String("zk", "127.0.0.1:2181", "zookeeper URL")
var n = flag.String("n", "127.0.0.1:2181", "Arith")

func main() {
	flag.Parse()

	server := rpcx.NewServer()
	plugin := &plugin.ZooKeeperRegisterPlugin{
		ServiceAddress:   "tcp@" + *addr,
		ZooKeeperServers: []string{*zk},
		BasePath:         "/rpcx",
		Metrics:          metrics.NewRegistry(),
		Services:         make([]string, 1),
		UpdateInterval:   10 * time.Second,
	}
	plugin.Start()
	server.PluginContainer.Add(plugin)
	server.RegisterName(*n, new(Arith), "weight=5&state=active")
	server.Serve("tcp", *addr)
}
