package logic

import (
	"context"
	"encoding/json"
	"mssgserver/net"
	"mssgserver/redis"
	"mssgserver/server/chat/model"
	"sync"
	"time"
)

//聊天频道
type ChatGroup struct {
	userMutex sync.RWMutex
	msgMutex  sync.RWMutex
	//用户
	users     map[int]*User
	//消息列表
	msgs      ItemQueue

	unionId int
}

func (c *ChatGroup) Enter(user *User) {
	c.userMutex.Lock()
	defer c.userMutex.Unlock()
	c.users[user.rid] = user
}

func (c *ChatGroup) GetUser(rid int) (*User,bool)  {
	c.userMutex.RLock()
	defer c.userMutex.RUnlock()
	u,ok := c.users[rid]
	return u,ok
}

func (c *ChatGroup) Exit(rid int) {
	c.userMutex.Lock()
	defer c.userMutex.Unlock()
	delete(c.users, rid)
}

func (c *ChatGroup) History(t int8) []model.ChatMsg {
	//消息列表
	c.msgMutex.RLock()
	defer c.msgMutex.RUnlock()
	msgs := c.msgs
	//从redis获取
	if t == 0 {
		//世界频道
		result, _ := redis.Pool.LRange(context.Background(), "chat_world", 0, -1).Result()
		for _,message := range result{
			msg := &Msg{}
			json.Unmarshal([]byte(message),msg)
			msgs.Enqueue(msg)
		}
	}
	items := msgs.items
	chatMsgs := make([]model.ChatMsg,0)
	for _, item :=range items{
		msg := item.(*Msg)
		cm := model.ChatMsg{RId: msg.RId, NickName: msg.NickName, Time: msg.Time.Unix(), Msg: msg.Msg}
		chatMsgs = append(chatMsgs,cm)
	}
	return chatMsgs
}

func (c *ChatGroup) PushMsg(rid int, msg string, t int8) *model.ChatMsg {
	c.userMutex.RLock()
	u,ok := c.users[rid]
	if !ok {
		return nil
	}
	c.userMutex.RUnlock()


	m := &Msg{
		Msg: msg,
		RId: rid,
		NickName: u.nickName,
		Time: time.Now(),
	}
	//c.msgMutex.Lock()
	//if c.msgs.Size() > 100 {
	//	c.msgs.Dequeue()
	//}
	//c.msgs.Enqueue(m)
	//
	//c.msgMutex.Unlock()

	///redis list数据结构  右进 左出
	jsonMsg,_ := json.Marshal(m)
	redis.Pool.RPush(context.Background(), "chat_world", jsonMsg)

	chatMsg := &model.ChatMsg{
		RId: rid,
		Msg: msg,
		NickName:u.nickName,
		Type: t,
		Time: time.Now().Unix(),
	}
	//消息要广播出去 所有的频道用户广播出去
	for _, user := range c.users{
		net.Mgr.PushByRoleId(user.rid,"chat.push",chatMsg)
	}
	return chatMsg
}

func NewGroup() *ChatGroup {
	return &ChatGroup{users: map[int]*User{}}
}

