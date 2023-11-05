package discover

import (
	"errors"
	"log"

	"github.com/wagfog/Myseckill_demo/pkg/bootstrap"
	"github.com/wagfog/Myseckill_demo/pkg/loadbalance"
)

var ConsulService DiscoveryClient
var LoadBalance loadbalance.LoadBalance
var Logger *log.Logger
var NoInstanceExistedErr = errors.New("no available client")

func init() {
	// 1.实例化一个 Consul 客户端，此处实例化了原生态实现版本
	ConsulService = New(bootstrap.DiscoverConfig.Host, bootstrap.DiscoverConfig.Port)
}
