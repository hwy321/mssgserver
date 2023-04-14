package main

import (
	"mssgserver/config"
	"mssgserver/net"
	"mssgserver/server/gate"
)

/**
	1.登录功能 account.login 需要通过网关 转发 登录服务器
	2. 网关（websocket的客户端） 如何和 登录服务器（websocket服务端）交互
	3. 网关又和游戏客户端 进行交互，网关是 websocket的服务端
	4. websocket的服务端 已经实现了
	5. websocket的客户端
    6. 网关 ： 代理服务器 （代理地址 代理的连接通道）  客户端连接（websocket连接）
	7. 路由：路由 接收所有的请求（*） 网关的 websocket服务端的功能
	8. 握手协议 检测第一次建立连接的时候 授信
*/

func main() {
	host := config.File.MustValue("gate_server", "host", "127.0.0.1")
	port := config.File.MustValue("gate_server", "port", "8004")
	s := net.NewServer(host + ":" + port)
	s.NeedSecret(true)
	gate.Init()
	s.Router(gate.Router)
	s.Start()
}
