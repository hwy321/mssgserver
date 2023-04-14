package net

import "sync"

type ReqBody struct {
	Seq     int64		`json:"seq"`
	Name 	string 		`json:"name"`
	Msg		interface{}	`json:"msg"`
	Proxy	string		`json:"proxy"`
}

type RspBody struct {
	Seq     int64		`json:"seq"`
	Name 	string 		`json:"name"`
	Code	int			`json:"code"`
	Msg		interface{}	`json:"msg"`
}

type WsMsgReq struct {
	Body	*ReqBody
	Conn	WSConn
	Context *WsContext
}
type WsContext struct {
	mutex sync.RWMutex
	property map[string]interface{}
}

func (ws *WsContext) Set(key string,value interface{})  {
	ws.mutex.Lock()
	defer ws.mutex.Unlock()
	ws.property[key] = value
}

func (ws *WsContext) Get(key string) interface{} {
	ws.mutex.RLock()
	defer ws.mutex.RUnlock()
	value ,ok := ws.property[key]
	if ok {
		return value
	}
	return nil
}
type WsMsgRsp struct {
	Body*	RspBody
}
//理解为 request请求 请求会有参数 请求中放参数 取参数
type WSConn interface {
	SetProperty(key string, value interface{})
	GetProperty(key string) (interface{}, error)
	RemoveProperty(key string)
	Addr() string
	Push(name string, data interface{})
}

type Handshake struct {
	Key string `json:"key"`
}
type Heartbeat struct {
	CTime int64	`json:"ctime"`
	STime int64	`json:"stime"`
}
