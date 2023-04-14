package net

import (
	"github.com/gorilla/websocket"
	"mssgserver/server/game/logic/conn"
	"mssgserver/server/game/logic/pos"
	"sync"
)

var Mgr = &WsMgr{
	userCache: make(map[int]WSConn),
	connCache: make(map[int64]WSConn),
	roleCache: make(map[int]WSConn),
}
type WsMgr struct {
	uc sync.RWMutex
	cc sync.RWMutex
	rc sync.RWMutex

	userCache map[int]WSConn
	connCache map[int64]WSConn
	roleCache map[int]WSConn
}

func (m *WsMgr) UserLogin(conn WSConn,uid int,token string)  {
	m.uc.Lock()
	defer m.uc.Unlock()
	oldConn := m.userCache[uid]
	if oldConn != nil {
		//有用户登录着呢
		if conn != oldConn {
			//通过旧客户端 有用户抢登录了
			oldConn.Push("robLogin",nil)
		}
	}
	m.userCache[uid] = conn
	conn.SetProperty("uid",uid)
	conn.SetProperty("token",token)
}

func (m *WsMgr) RoleEnter(conn WSConn,rid int)  {
	m.rc.Lock()
	defer m.rc.Unlock()
	conn.SetProperty("rid",rid)
	m.roleCache[rid] = conn
}

func (w *WsMgr) PushByRoleId(rid int, msgName string, data interface{}) bool {
	if rid <= 0	{
		return false
	}
	w.rc.Lock()
	defer w.rc.Unlock()
	c, ok := w.roleCache[rid]
	if ok {
		c.Push(msgName, data)
		return true
	}else{
		return false
	}
}

func (w *WsMgr) Push(pushSync conn.PushSync) {

	belongToRIds := pushSync.BelongToRId()
	model := pushSync.ToModel()
	//推送给当前视野内的所有玩家
	isCellView := pushSync.IsCellView()
	x, y := pushSync.Position()
	cells := make(map[int]int)
	//推送给开始位置
	if isCellView {
		cellRIds := pos.RPMgr.GetCellRoleIds(x, y, 8, 6)
		for _, rid := range cellRIds {
			//是否能出现在视野
			if can := pushSync.IsCanView(rid, x, y); can{
				w.PushByRoleId(rid, pushSync.PushMsgName(), model)
				cells[rid] = rid
			}
		}
	}
	//推送给目标位置
	tx, ty := pushSync.TPosition()
	if tx >= 0 && ty >= 0{
		var cellRIds []int
		if isCellView {
			cellRIds = pos.RPMgr.GetCellRoleIds(tx, ty, 8, 6)
		}else{
			cellRIds = pos.RPMgr.GetCellRoleIds(tx, ty, 0, 0)
		}

		for _, rid := range cellRIds {
			if _, ok := cells[rid]; ok == false{
				if can := pushSync.IsCanView(rid, tx, ty); can{
					w.PushByRoleId(rid, pushSync.PushMsgName(), model)
					cells[rid] = rid
				}
			}
		}
	}

	//推送给当前的所有角色 自己
	for _,role := range belongToRIds{
		w.PushByRoleId(role,pushSync.PushMsgName(),model)
	}
}

var cid int64

func (m *WsMgr) NewConn(wsConn *websocket.Conn, secret bool) *wsServer {
	s := NewWsServer(wsConn,secret)
	cid++
	s.SetProperty("cid",cid)
	return s
}

func (w *WsMgr) UserLogout(wsConn WSConn) {
	w.RemoveUser(wsConn)
}

func (w *WsMgr) RemoveUser(conn WSConn) {
	w.uc.Lock()
	uid, err := conn.GetProperty("uid")
	if err == nil {
		//只删除自己的conn
		id := uid.(int)
		c, ok := w.userCache[id]
		if ok && c == conn{
			delete(w.userCache, id)
		}
	}
	w.uc.Unlock()

	w.rc.Lock()
	rid, err := conn.GetProperty("rid")
	if err == nil {
		//只删除自己的conn
		id := rid.(int)
		c, ok := w.roleCache[id]
		if ok && c == conn{
			delete(w.roleCache, id)
		}
	}
	w.rc.Unlock()

	conn.RemoveProperty("session")
	conn.RemoveProperty("uid")
	conn.RemoveProperty("role")
	conn.RemoveProperty("rid")
}

func NewWsServer(wsConn *websocket.Conn,needSecret bool) *wsServer {
	s := &wsServer{
		wsConn: wsConn,
		outChan: make(chan *WsMsgRsp, 1000),
		property: make(map[string]interface{}),
		Seq: 0,
		needSecret: needSecret,
	}

	return  s
}



