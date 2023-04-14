package controller

import (
	"github.com/mitchellh/mapstructure"
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/login/model"
	"mssgserver/server/login/proto"
	"mssgserver/server/models"
	"mssgserver/utils"
	"time"
)

var DefaultAccount = &Account{}
type Account struct {

}

func (a *Account) Router(r *net.Router)  {
	g := r.Group("account")
	g.AddRouter("login",a.login)
	g.AddRouter("logout",a.logout)
	g.AddRouter("reLogin",a.reLogin)
}

func (a *Account) login(req *net.WsMsgReq, rsp *net.WsMsgRsp)  {
	/**
		1. 用户名 密码 硬件id
		2. 根据用户名 查询user表 得到数据
		3. 进行密码比对，如果密码正确 登录成功
	    4. 保存用户登录记录
		5. 保存用户的最后一次登录信息
	    6. 客户端 需要 一个session，jwt 生成一个加密字符串的加密算法
	    7. 客户端 在发起需要用户登录的行为时，判断用户是否合法
	 */
	loginReq := &proto.LoginReq{}
	loginRes := &proto.LoginRsp{}
	mapstructure.Decode(req.Body.Msg,loginReq)
	user := &models.User{}
	ok, err := db.Engine.Table(user).Where("username=?",loginReq.Username).Get(user)
	if err != nil {
		log.Println("用户表查询出错",err)
		return
	}
	if !ok {
		//有没有查出来数据
		rsp.Body.Code = constant.UserNotExist
		return
	}
	pwd := utils.Password(loginReq.Password,user.Passcode)
	if pwd != user.Passwd {
		rsp.Body.Code = constant.PwdIncorrect
		return
	}
	//jwt A.B.C 三部分 A定义加密算法 B定义放入的数据 C部分 根据秘钥+A和B生成加密字符串
	token,_ := utils.Award(user.UId)
	rsp.Body.Code = constant.OK
	loginRes.UId = user.UId
	loginRes.Username = user.Username
	loginRes.Session = token
	loginRes.Password = ""
	rsp.Body.Msg = loginRes

	//保存用户登录记录
	ul := &model.LoginHistory{
		UId: user.UId, CTime: time.Now(), Ip: loginReq.Ip,
		Hardware: loginReq.Hardware, State: model.Login,
	}
	db.Engine.Table(ul).Insert(ul)
	//最后一次登录的状态记录
	ll := &model.LoginLast{}
	ok ,_ = db.Engine.Table(ll).Where("uid=?",user.UId).Get(ll)
	if ok {
		//有数据 更新
		ll.IsLogout = 0
		ll.Ip = loginReq.Ip
		ll.LoginTime = time.Now()
		ll.Session = token
		ll.Hardware = loginReq.Hardware
		db.Engine.Table(ll).ID(ll.Id).Update(ll)
	}else{
		ll.IsLogout = 0
		ll.Ip = loginReq.Ip
		ll.LoginTime = time.Now()
		ll.Session = token
		ll.Hardware = loginReq.Hardware
		ll.UId = user.UId
		_, err := db.Engine.Table(ll).Insert(ll)
		if err != nil {
			log.Println(err)
		}
	}
	//缓存一下 此用户和当前的ws连接
	net.Mgr.UserLogin(req.Conn,user.UId,token)
}

func (a *Account) logout(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &proto.LogoutReq{}
	rspObj := &proto.LogoutRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rspObj.UId = reqObj.UId
	rsp.Body.Code = constant.OK

	tt := time.Now()
	//登出，写记录
	lh := &model.LoginHistory{UId: reqObj.UId, CTime: tt, State: model.Logout}
	db.Engine.Insert(lh)

	ll := &model.LoginLast{}
	ok, _ := db.Engine.Table(ll).Where("uid=?", reqObj.UId).Get(ll)
	if ok {
		ll.IsLogout = 1
		ll.LogoutTime = time.Now()
		db.Engine.ID(ll.Id).Cols("is_logout", "logout_time").Update(ll)

	}else{
		ll = &model.LoginLast{UId: reqObj.UId, LogoutTime: tt, IsLogout: 0}
		db.Engine.Insert(ll)
	}

	net.Mgr.UserLogout(req.Conn)
}

func (a *Account) reLogin(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &proto.ReLoginReq{}
	rspObj := &proto.ReLoginRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	if reqObj.Session == ""{
		rsp.Body.Code = constant.SessionInvalid
		return
	}

	rsp.Body.Msg = rspObj
	rspObj.Session = reqObj.Session

	_,c, err := utils.ParseToken(reqObj.Session)
	if err != nil{
		rsp.Body.Code = constant.SessionInvalid
	}else{
		//数据库验证一下
		ll := &model.LoginLast{}
		db.Engine.Table(ll).Where("uid=?", c.Uid).Get(ll)

		if ll.Session == reqObj.Session {
			if ll.Hardware == reqObj.Hardware {
				rsp.Body.Code = constant.OK
				net.Mgr.UserLogin(req.Conn, c.Uid, reqObj.Session)
			}else{
				rsp.Body.Code = constant.HardwareIncorrect
			}
		}else{
			rsp.Body.Code = constant.SessionInvalid
		}
	}
}

