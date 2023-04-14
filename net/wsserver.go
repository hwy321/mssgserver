package net

import (
	"encoding/json"
	"errors"
	"github.com/forgoer/openssl"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"go.uber.org/zap"
	"log"
	myLog "mssgserver/log"
	"mssgserver/utils"
	"strings"
	"sync"
	"time"
)

//websocket服务
type wsServer struct {
	wsConn       *websocket.Conn
	router       *Router
	outChan      chan *WsMsgRsp
	Seq          int64
	property     map[string]interface{}
	propertyLock sync.RWMutex
	needSecret   bool
}

func (w *wsServer) Router(router *Router) {
	w.router = router
}

func (w *wsServer) SetProperty(key string, value interface{}) {
	w.propertyLock.Lock()
	defer w.propertyLock.Unlock()
	w.property[key] = value
}

func (w *wsServer) GetProperty(key string) (interface{}, error) {
	w.propertyLock.RLock()
	defer w.propertyLock.RUnlock()
	if value, ok := w.property[key]; ok {
		return value, nil
	} else {
		return nil, errors.New("no property found")
	}
}
func (w *wsServer) RemoveProperty(key string) {
	w.propertyLock.Lock()
	defer w.propertyLock.Unlock()
	delete(w.property, key)
}
func (w *wsServer) Addr() string {
	return w.wsConn.RemoteAddr().String()
}
func (w *wsServer) Push(name string, data interface{}) {
	rsp := &WsMsgRsp{Body: &RspBody{Name: name, Msg: data, Seq: 0}}
	w.outChan <- rsp
}

//通道一旦建立 ，那么 收发消息 就得要一直监听才行
func (w *wsServer) Start() {
	//启动读写数据的处理逻辑
	go w.readMsgLoop()
	go w.writeMsgLoop()
}

func (w *wsServer) writeMsgLoop() {
	for {
		select {
		case msg := <-w.outChan:
			w.Write(msg.Body)
		}
	}
}

func (w *wsServer) Write(msg interface{}) {
	data, err := json.Marshal(msg)
	if err != nil {
		log.Println(err)
	}
	secretKey, err := w.GetProperty("secretKey")
	if err == nil {
		//有加密
		key := secretKey.(string)
		//数据做加密
		data, _ = utils.AesCBCEncrypt(data, []byte(key), []byte(key), openssl.ZEROS_PADDING)
	}
	//压缩
	if data, err := utils.Zip(data); err == nil {
		err := w.wsConn.WriteMessage(websocket.BinaryMessage, data)
		if err != nil {
			log.Println("服务端写数据出错", err)
		}
		d, _ := json.Marshal(msg)

		// 不打印心跳
		if !strings.Contains(string(d), "heartbeat") {
			myLog.InfoLog.Info("服务端写数据", zap.String("response", string(d)))
		}
		//myLog.ErrorLog.Error("服务端写数据",zap.String("response", string(d)))
		//log.Println("服务端写数据",string(d))
	}
}

func (w *wsServer) readMsgLoop() {
	//先读到客户端 发送过来的数据，然后 进行处理，然后在回消息
	//经过路由 实际处理程序
	defer func() {
		if err := recover(); err != nil {
			log.Println("服务端捕捉到异常", err)
			w.Close()
		}
	}()
	for {
		_, data, err := w.wsConn.ReadMessage()
		if err != nil {
			log.Println("收消息出现错误:", err)
			break
		}
		//收到消息 解析消息 前端发送过来的消息 就是json格式
		//1. data 解压 unzip
		data, err = utils.UnZip(data)
		if err != nil {
			log.Println("解压数据出错，非法格式：", err)
			continue
		}
		//2. 前端的消息 加密消息 进行解密
		if w.needSecret {
			secretKey, err := w.GetProperty("secretKey")
			if err == nil {
				//有加密
				key := secretKey.(string)
				//客户端传过来的数据是加密的 需要解密
				d, err := utils.AesCBCDecrypt(data, []byte(key), []byte(key), openssl.ZEROS_PADDING)
				if err != nil {
					log.Println("数据格式有误，解密失败:", err)
					//出错后 发起握手
					w.Handshake()
				} else {
					data = d
				}
			}
		}
		//3. data 转为body
		body := &ReqBody{}
		err = json.Unmarshal(data, body)
		if err != nil {
			log.Println("服务端json格式解析有误，非法格式:", err)
		} else {
			// 获取到前端传递的数据了，拿上这些数据 去具体的业务进行处理
			context := &WsContext{
				property: make(map[string]interface{}),
			}
			req := &WsMsgReq{Conn: w, Body: body, Context: context}
			rsp := &WsMsgRsp{Body: &RspBody{Name: body.Name, Seq: req.Body.Seq}}
			if req.Body.Name == "heartbeat" {
				//回心跳的消息
				h := &Heartbeat{}
				mapstructure.Decode(req.Body.Msg, h)
				h.STime = time.Now().UnixNano() / 1e6
				rsp.Body.Msg = h
			} else {
				if w.router != nil {
					log.Println("路由执行", req)
					w.router.Run(req, rsp)
				}
			}
			w.outChan <- rsp
		}
	}
	w.Close()
}

func (w *wsServer) Close() {
	_ = w.wsConn.Close()
}

const HandshakeMsg = "handshake"

//当游戏客户端 发送请求的时候 会先进行握手协议
//后端会发送对应的加密key给客户端
//客户端在发送数据的时候 就会使用此key进行加密处理
func (w *wsServer) Handshake() {
	secretKey := ""
	key, err := w.GetProperty("secretKey")
	if err == nil {
		secretKey = key.(string)
	} else {
		secretKey = utils.RandSeq(16)
	}
	handshake := &Handshake{Key: secretKey}

	body := &RspBody{Name: HandshakeMsg, Msg: handshake}

	if data, err := json.Marshal(body); err == nil {
		if secretKey != "" {
			w.SetProperty("secretKey", secretKey)
		} else {
			w.RemoveProperty("secretKey")
		}
		if data, err := utils.Zip(data); err == nil {
			w.wsConn.WriteMessage(websocket.BinaryMessage, data)
		}
	}
}
