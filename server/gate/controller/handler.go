package controller

import (
	"github.com/mitchellh/mapstructure"
	"log"
	"mssgserver/config"
	"mssgserver/constant"
	"mssgserver/net"
	chatModel "mssgserver/server/chat/model"
	"mssgserver/server/game/model"
	"strings"
	"sync"
)

var GateHandler = &Handler{
	proxyMap: make(map[string]map[int64]*net.ProxyClient),
}

type Handler struct {
	proxyMutex sync.Mutex
	//代理地址 -》客户端连接（游戏客户端的id -》连接）
	proxyMap map[string]map[int64]*net.ProxyClient
	loginProxy string
	gameProxy string
	chatProxy string
}

func (h *Handler) Router(r *net.Router)  {
	h.loginProxy = config.File.MustValue("gate_server", "login_proxy", "ws://127.0.0.1:8003")
	h.gameProxy = config.File.MustValue("gate_server", "game_proxy", "ws://127.0.0.1:8001")
	h.chatProxy = config.File.MustValue("gate_server", "chat_proxy", "ws://127.0.0.1:8002")
	g := r.Group("*")
	g.AddRouter("*",h.all)
}

func (h *Handler) all(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	log.Println("接收到请求",req.Body.Name)
	h.deal(req,rsp)

	if req.Body.Name == "role.enterServer" && rsp.Body.Code == constant.OK {
		//进入游戏成功了 进入聊天服 第一步 登录聊天服务器
		rspObj := &model.EnterServerRsp{}
		mapstructure.Decode(rsp.Body.Msg, rspObj)
		r := &chatModel.LoginReq{RId: rspObj.Role.RId, NickName: rspObj.Role.NickName, Token: rspObj.Token}
		reqBody := &net.ReqBody{Seq: 0, Name: "chat.login", Msg: r, Proxy: ""}
		rspBody := &net.RspBody{Seq: 0, Name: "chat.login", Msg: r, Code: 0}
		h.deal(&net.WsMsgReq{Body: reqBody, Conn:req.Conn}, &net.WsMsgRsp{Body: rspBody})
	}
}

func (h *Handler) onPush(conn *net.ClientConn, body *net.RspBody) {
	gc ,err := conn.GetProperty("gateConn")
	if err != nil {
		log.Println("onPush gateConn ",err)
		return
	}
	gateConn := gc.(net.WSConn)
	gateConn.Push(body.Name,body.Msg)
}

func (h *Handler) deal(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//account 转发
	name := req.Body.Name
	proxyStr := ""
	if isAccount(name) {
		proxyStr = h.loginProxy
	}else if isChat(name) {
		proxyStr = h.chatProxy
	}else{
		proxyStr = h.gameProxy
	}
	if proxyStr == "" {
		rsp.Body.Code = constant.ProxyNotInConnect
		return
	}
	h.proxyMutex.Lock()
	_, ok := h.proxyMap[proxyStr]
	if !ok {
		h.proxyMap[proxyStr] = make(map[int64]*net.ProxyClient)
	}
	h.proxyMutex.Unlock()
	//客户端id
	c,err := req.Conn.GetProperty("cid")
	if err != nil {
		log.Println("cid未取到",err)
		rsp.Body.Code = constant.InvalidParam
		return
	}
	cid := c.(int64)
	proxy := h.proxyMap[proxyStr][cid]
	if proxy == nil {
		proxy = net.NewProxyClient(proxyStr)
		err := proxy.Connect()
		if err != nil {
			h.proxyMutex.Lock()
			delete(h.proxyMap[proxyStr],cid)
			h.proxyMutex.Unlock()
			rsp.Body.Code = constant.ProxyConnectError
			return
		}
		h.proxyMap[proxyStr][cid] = proxy
		proxy.SetProperty("cid",cid)
		proxy.SetProperty("proxy",proxyStr)
		proxy.SetProperty("gateConn",req.Conn)
		proxy.SetOnPush(h.onPush)
	}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	r, err := proxy.Send(req.Body.Name,req.Body.Msg)
	if r != nil {
		rsp.Body.Code = r.Code
		rsp.Body.Msg = r.Msg
	}else{
		rsp.Body.Code = constant.ProxyConnectError
		return
	}
}

func isAccount(name string) bool {
	return strings.HasPrefix(name,"account.")
}

func isChat(name string) bool {
	return strings.HasPrefix(name,"chat.")
}
