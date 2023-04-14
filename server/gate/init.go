package gate

import (
	"mssgserver/net"
	"mssgserver/server/gate/controller"
)

var Router = &net.Router{}
func Init()  {
	initRouter()
}

func initRouter() {
	controller.GateHandler.Router(Router)
}
