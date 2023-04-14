package chat

import (
	"mssgserver/net"
	"mssgserver/server/chat/controller"
)

var Router = &net.Router{}

func Init(){
	initRouter()
}

func initRouter() {

	controller.ChatController.Router(Router)

}
