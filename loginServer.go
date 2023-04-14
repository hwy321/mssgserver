package main

import (
	"mssgserver/config"
	"mssgserver/net"
	"mssgserver/server/login"
)

// http://localhost:8080/api/login
//localhost:8080 服务器  /api/login 路由
//websocket 区别 ：ws://localhost:8080 服务器  发消息 （封装为路由）

func main()  {
	host := config.File.MustValue("login_server","host","127.0.0.1")
	port := config.File.MustValue("login_server","port","8003")
	s := net.NewServer(host + ":" + port)
	s.NeedSecret(false)
	login.Init()
	s.Router(login.Router)
	s.Start()
}
