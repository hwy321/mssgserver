package login

import (
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/login/controller"
)

var Router = net.NewRouter()
func Init()  {
	//测试数据库，并且初始化数据库
	db.TestDB()
	//还有别的初始化方法
	initRouter()
}

func initRouter()  {
	controller.DefaultAccount.Router(Router)
}
