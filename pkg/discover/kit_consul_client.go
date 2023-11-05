package discover

import (
	"log"
	"strconv"

	"github.com/go-kit/kit/sd/consul"
	"github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/wagfog/Myseckill_demo/pkg/common"
)

func New(consulHost string, consulPort string) *DiscoveryClientInstance {
	//我不是很理解为什么要这么做
	port, _ := strconv.Atoi(consulPort)
	// 通过 Consul Host 和 Consul Port 创建一个 consul.Client
	consulConfig := api.DefaultConfig()                          //先创建一个默认的设置
	consulConfig.Address = consulHost + ":" + strconv.Itoa(port) //更改服务存储的地址
	apiClient, err := api.NewClient(consulConfig)                //创建新的，consul的apiclient
	if err != nil {
		return nil
	}

	client := consul.NewClient(apiClient) //kit框架中的client

	return &DiscoveryClientInstance{
		Host:   consulHost,
		port:   port,
		config: consulConfig,
		client: client,
	}
}

func (consulClient *DiscoveryClientInstance) Register(instanceId, svcHost, healthCheckUrl, svcPort string, svcName string, weight int, meta map[string]string, tags []string, logger *log.Logger) bool {
	port, _ := strconv.Atoi(svcPort)

	// 1. 构建服务实例元数据
	serviceRegistration := &api.AgentServiceRegistration{
		ID:      instanceId,
		Name:    svcName,
		Address: svcHost,
		Port:    port,
		Meta:    meta,
		Tags:    tags,
		Weights: &api.AgentWeights{
			Passing: weight,
		},
		Check: &api.AgentServiceCheck{
			DeregisterCriticalServiceAfter: "30s",  //// 30s 内健康检查失败，该服务实例会被 consul 主动下线
			HTTP:                           "http", //" + svcHost + ":" + strconv.Itoa(port) + healthCheckUrl,
			Interval:                       "15s",  // 健康检查时间间隔为 15s
		},
	}
	// 2. 发送服务注册到 Consul 中
	err := consulClient.client.Register(serviceRegistration)

	if err != nil {
		if logger != nil {
			logger.Println("Register Service Error!", err)
		}
		return false
	}
	if logger != nil {
		logger.Println("Register Service Success!")
	}
	return true
}

func (consulClient *DiscoveryClientInstance) DeRegister(instanceId string, logger *log.Logger) bool {
	// 构建包含服务实例 ID 的元数据结构体
	serviceRegistration := &api.AgentServiceRegistration{
		ID: instanceId,
	}
	// 发送服务注销请求
	err := consulClient.client.Deregister(serviceRegistration)

	if err != nil {
		if logger != nil {
			logger.Println("Deregister service Error!", err)
		}
		return false
	}
	if logger != nil {
		logger.Println("Deregister service Success!")
	}
	return true
}

func (consulClient *DiscoveryClientInstance) DiscoverServices(serviceName string, logger *log.Logger) []*common.ServiceInstance {
	////  该服务已监控并缓存
	instanceList, ok := consulClient.instancesMap.Load(serviceName)
	if ok {
		return instanceList.([]*common.ServiceInstance)
	}
	// 申请锁
	//这么做是为了避免多个 goroutine 都申请锁并重复请求服务实例列表。
	//缓存没有所以需要去consul中查找，用互斥锁控制并发量，保证每次只有一个线程去consul查
	consulClient.mutex.Lock()
	defer consulClient.mutex.Unlock()

	// 再次检查是否监控
	instanceList, ok = consulClient.instancesMap.Load(serviceName)
	if ok {
		return instanceList.([]*common.ServiceInstance)
	} else {
		// 注册监控
		go func() {
			params := make(map[string]interface{})
			params["type"] = "service"
			params["service"] = serviceName
			//Consul的Watch功能允许客户端监视Consul中的变化并获取实时通知。
			//通过使用Watch，客户端可以订阅某个特定的Consul资源，并在该资源发生更改时接收到通知。
			plan, _ := watch.Parse(params)
			//在Consul中，计划（Plan）是用来执行一系列操作的一种机制
			//例如注册服务、更新配置、执行健康检查等。通过运行计划，可以按顺序执行这些任务，确保它们在正确的顺序下得到处理。
			plan.Handler = func(u uint64, i interface{}) {
				if i == nil {
					return
				}
				v, ok := i.([]*api.ServiceEntry)
				if !ok {
					return
				}
				// 没有服务实例在线
				if len(v) == 0 {
					consulClient.instancesMap.Store(serviceName, []*common.ServiceInstance{}) //存入空切片
				}

				var healthServices []*common.ServiceInstance

				for _, service := range v {
					if service.Checks.AggregatedStatus() == api.HealthPassing {
						healthServices = append(healthServices, newServiceInstance(service.Service))
					}
				}
				consulClient.instancesMap.Store(serviceName, healthServices)
			}
			defer plan.Stop()
			//plan.Run()是一个函数调用，它表示执行计划
			//通过传递Consul服务的地址，plan.Run()可以与Consul服务器建立连接，并开始执行计划中定义的任务。
			plan.Run(consulClient.config.Address) //--->之前定义好的handler
		}()
	}

	// 根据服务名请求服务实例列表
	entries, _, err := consulClient.client.Service(serviceName, "", false, nil)
	if err != nil {
		consulClient.instancesMap.Store(serviceName, []*common.ServiceInstance{})
		if logger != nil {
			logger.Println("Discover serice Error!", err)
		}
		return nil
	}
	instances := make([]*common.ServiceInstance, len(entries))
	for i := 0; i < len(instances); i++ {
		instances[i] = newServiceInstance(entries[i].Service)
	}
	consulClient.instancesMap.Store(serviceName, instanceList)
	return instances
}

func newServiceInstance(service *api.AgentService) *common.ServiceInstance {
	rpcPOrt := service.Port - 1
	if service.Meta != nil {
		if rpcPortString, ok := service.Meta["rpcPort"]; ok {
			rpcPOrt, _ = strconv.Atoi(rpcPortString)
		}
	}
	return &common.ServiceInstance{
		Host:     service.Address,
		Port:     service.Port,
		GrpcPort: rpcPOrt,
		Weight:   service.Weights.Passing,
	}
}
