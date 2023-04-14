package net

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/forgoer/openssl"
	"github.com/gorilla/websocket"
	"github.com/mitchellh/mapstructure"
	"log"
	"mssgserver/constant"
	"mssgserver/utils"
	"sync"
	"time"
)

type syncCtx struct {
	//Goroutine 的上下文，包含 Goroutine 的运行状态、环境、现场等信息
	ctx context.Context
	cancel context.CancelFunc
	outChan chan *RspBody
}

func NewSyncCtx() *syncCtx  {
	ctx,cancel := context.WithTimeout(context.Background(),15*time.Second)
	return &syncCtx{
		ctx:ctx,
		cancel: cancel,
		outChan: make(chan *RspBody),
	}
}

func (s *syncCtx) wait () *RspBody {
	select {
	case msg := <- s.outChan:
		return msg
	case <- s.ctx.Done():
		log.Println("代理服务响应消息超时了...")
		return nil
	}
}
type ClientConn struct {
	wsConn *websocket.Conn
	isClosed bool
	property 		map[string]interface{}
	propertyLock  	sync.RWMutex
	Seq				int64
	handshake		bool
	handshakeChan 	chan bool
	onPush func(conn *ClientConn,body *RspBody)
	onClose    		func(conn*ClientConn)
	syncCtxMap map[int64]*syncCtx
	syncCtxLock sync.RWMutex
}

func (c *ClientConn) Start() bool {
	//做的事情 就是 一直不停的接收消息
	// 等待握手的消息返回
	c.handshake = false
	go c.wsReadLoop()
	return c.waitHandShake()
}

func (c *ClientConn) waitHandShake() bool {
	//等待握手的成功  等待握手的消息
	//万一 程序出现问题  超时了 一直响应不到 超时时间
	if !c.handshake {
		ctx,cancel := context.WithTimeout(context.Background(),5*time.Second)
		defer cancel()
		select {
		case _ = <-c.handshakeChan:
			log.Println("握手成功了")
			return true
		case <-ctx.Done():
			log.Println("握手超时了....")
			return false
		}
	}
	return true
}

func (c *ClientConn) wsReadLoop() {
	//for  {
	//	_, data, err := c.wsConn.ReadMessage()
	//	fmt.Println(data)
	//	fmt.Println(err)
	//	//读取消息 可能会收到很多消息 握手，心跳，请求信息（account.login）
	//	//服务端 写消息
	//
	//	//收到握手消息了
	//	c.handshake = true
	//	c.handshakeChan <- true
	//}

	defer func() {
		if err := recover(); err != nil {
			log.Println("客户端捕捉到异常",err)
			c.Close()
		}
	}()
	for  {
		_, data, err := c.wsConn.ReadMessage()
		if err != nil {
			log.Println("收消息出现错误:",err)
			break
		}
		//收到消息 解析消息 前端发送过来的消息 就是json格式
		//1. data 解压 unzip
		data,err = utils.UnZip(data)
		if err != nil {
			log.Println("解压数据出错，非法格式：",err)
			continue
		}
		//2. 前端的消息 加密消息 进行解密
		secretKey,err := c.GetProperty("secretKey")
		if err == nil {
			//有加密
			key := secretKey.(string)
			//客户端传过来的数据是加密的 需要解密
			d, err := utils.AesCBCDecrypt(data, []byte(key), []byte(key), openssl.ZEROS_PADDING)
			if err != nil {
				log.Println("数据格式有误，解密失败:",err)
			}else{
				data = d
			}
		}
		//3. data 转为body
		body := &RspBody{}
		err = json.Unmarshal(data,body)
		if err != nil {
			log.Println("数据格式有误，非法格式:",err)
		}else{
			//握手 别的一些请求
			if body.Seq == 0 {
				if body.Name == HandshakeMsg {
					//获取秘钥
					hs := &Handshake{}
					mapstructure.Decode(body.Msg,hs)
					if hs.Key != "" {
						c.SetProperty("secretKey",hs.Key)
					}else{
						c.RemoveProperty("secretKey")
					}
					c.handshake = true
					c.handshakeChan <- true
				}else{
					if c.onPush != nil {
						c.onPush(c,body)
					}
				}
			}else{
				c.syncCtxLock.RLock()
				ctx,ok := c.syncCtxMap[body.Seq]
				c.syncCtxLock.RUnlock()
				if ok {
					ctx.outChan <- body
				}else{
					log.Println("no seq syncCtx find")
				}

			}
		}
	}
	c.Close()

}

func (c *ClientConn) Close() {
	_ = c.wsConn.Close()
}

func NewClientConn(wsConn *websocket.Conn) *ClientConn  {
	return &ClientConn{
		wsConn: wsConn,
		handshakeChan: make(chan bool),
		Seq: 0,
		isClosed: false,
		property: make(map[string]interface{}),
		syncCtxMap: map[int64]*syncCtx{},
	}
}

func (c *ClientConn) SetProperty(key string, value interface{}){
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	c.property[key] = value
}

func (c *ClientConn) GetProperty(key string) (interface{}, error){
	c.propertyLock.RLock()
	defer c.propertyLock.RUnlock()
	if value, ok := c.property[key]; ok {
		return value, nil
	} else {
		return nil, errors.New("no property found")
	}
}
func (c *ClientConn) RemoveProperty(key string){
	c.propertyLock.Lock()
	defer c.propertyLock.Unlock()
	delete(c.property,key)
}
func (c *ClientConn) Addr() string {
	return c.wsConn.RemoteAddr().String()
}
func (c *ClientConn) Push(name string, data interface{}){
	rsp := &WsMsgRsp{Body: &RspBody{Name: name, Msg: data, Seq: 0}}
	c.write(rsp.Body)
}

func (c *ClientConn) SetOnPush(hook func(conn *ClientConn,body *RspBody)) {
	c.onPush = hook
}

func (c *ClientConn) write(body interface{}) error{
	data ,err := json.Marshal(body)
	if err != nil {
		log.Println(err)
		return err
	}
	//secretKey,err := c.GetProperty("secretKey")
	//if err == nil {
	//	//有加密
	//	key := secretKey.(string)
	//	//数据做加密
	//	data, err = utils.AesCBCEncrypt(data, []byte(key), []byte(key), openssl.ZEROS_PADDING)
	//	if err != nil {
	//		log.Println("加密失败",err)
	//		return err
	//	}
	//}
	//压缩
	if data, err := utils.Zip(data); err == nil{
		err := c.wsConn.WriteMessage(websocket.BinaryMessage,data)
		if err != nil {
			log.Println("写数据失败",err)
			return err
		}
	}else {
		log.Println("压缩数据失败",err)
		return err
	}
	return nil
}

func (c *ClientConn) Send(name string, msg interface{}) *RspBody {
	//把请求 发送给 代理服务器 登录服务器，等待返回
	c.Seq += 1
	seq := c.Seq
	sc := NewSyncCtx()
	c.syncCtxLock.Lock()
	c.syncCtxMap[seq] = sc
	c.syncCtxLock.Unlock()
	rsp := &RspBody{Name: name,Seq: seq,Code: constant.OK}
	//req请求
	req := &ReqBody{Seq: seq,Name: name,Msg: msg}
	err := c.write(req)

	if err != nil {
		sc.cancel()
	}else{
		r := sc.wait()
		if r == nil {
			rsp.Code = constant.ProxyConnectError
		}else{
			rsp = r
		}
	}
	c.syncCtxLock.Lock()
	delete(c.syncCtxMap,seq)
	c.syncCtxLock.Unlock()
	return rsp
}
