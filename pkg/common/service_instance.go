package common

type ServiceInstance struct {
	Host      string // Host
	Port      int    //Post
	Weight    int    // 权重
	CurWeight int    // 当前权重

	GrpcPort int // RPC 服务的端口号
}
