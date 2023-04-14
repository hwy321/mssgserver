package controller

import (
	"github.com/mitchellh/mapstructure"
	"mssgserver/constant"
	"mssgserver/net"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"time"
)

var InteriorController = &interiorController{}
type interiorController struct {

}

func (i *interiorController) Router(router *net.Router)  {
	g := router.Group("interior")
	g.Use(middleware.Log())
	g.AddRouter("openCollect",i.openCollect,middleware.CheckRole())
	g.AddRouter("collect",i.collect,middleware.CheckRole())
	g.AddRouter("transform",i.transform,middleware.CheckRole())
}

func (i *interiorController) openCollect(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	rspObj := &model.OpenCollectionRsp{}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj

	r,_ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	ra := logic.RoleAttrService.Get(role.RId)
	if ra != nil {
		//征收次数
		rspObj.CurTimes = ra.CollectTimes
		rspObj.Limit = gameConfig.Base.Role.CollectTimesLimit
		//征收间隔时间
		interval := gameConfig.Base.Role.CollectInterval
		if ra.LastCollectTime.IsZero() {
			rspObj.NextTime = 0
		}else{
			if rspObj.CurTimes >= rspObj.Limit {
				//今天已经完成征收了，下一次征收就是第二天(最后一次征收时间为准)
				//第二天 从0点就开始了
				y,m,d := ra.LastCollectTime.Add(24 * time.Hour).Date()
				//东八区time.FixedZone("CST",8*3600)
				ti := time.Date(y,m,d,0,0,0,0,time.FixedZone("CST",8*3600))
				rspObj.NextTime = ti.UnixNano()/1e6
			}else{
				ti := ra.LastCollectTime.Add(time.Duration(interval) * time.Second)
				rspObj.NextTime = ti.UnixNano()/1e6
			}
		}
	}
}

func (i *interiorController) collect(req *net.WsMsgReq, rsp *net.WsMsgRsp) {

	//查询角色资源 得到当前的金币
	//查询角色属性 获取征收的相关信息
	//查询获取当前的产量 征收的金币是多少

	rspObj := &model.CollectionRsp{}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
	r,_ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	//属性
	ra := logic.RoleAttrService.Get(role.RId)
	if ra == nil {
		rsp.Body.Code = constant.DBError
		return
	}
	//角色资源
	rs := logic.RoleResService.GetRoleRes(role.RId)
	if rs == nil {
		rsp.Body.Code = constant.DBError
		return
	}
	//产量
	yield := logic.RoleResService.GetYield(role.RId)
	rs.Gold += yield.Gold
	//go channel 一旦需要更新 发一个需要更新的信号 接收方 消费方 接收消息 进行更新
	rs.SyncExecute()
	rspObj.Gold = yield.Gold
	//计算征收
	curTime := time.Now()
	limit := gameConfig.Base.Role.CollectTimesLimit
	interval := gameConfig.Base.Role.CollectInterval
	lastTime := ra.LastCollectTime
	if curTime.YearDay() != lastTime.YearDay() || curTime.Year() != lastTime.Year() {
		ra.CollectTimes = 0
		ra.LastCollectTime = time.Time{}
	}
	ra.CollectTimes += 1
	ra.LastCollectTime = curTime
	ra.SyncExecute()
	rspObj.Limit = limit
	rspObj.CurTimes = ra.CollectTimes
	if rspObj.CurTimes >= rspObj.Limit {
		//今天已经完成征收了，下一次征收就是第二天(最后一次征收时间为准)
		//第二天 从0点就开始了
		y,m,d := ra.LastCollectTime.Add(24 * time.Hour).Date()
		//东八区time.FixedZone("CST",8*3600)
		ti := time.Date(y,m,d,0,0,0,0,time.FixedZone("CST",8*3600))
		rspObj.NextTime = ti.UnixNano()/1e6
	}else{
		ti := ra.LastCollectTime.Add(time.Duration(interval) * time.Second)
		rspObj.NextTime = ti.UnixNano()/1e6
	}
}

func (i *interiorController) transform(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//查询资源
	//查询集市是否符合要求
	//Form To From减去 To增加  0-3 0 木材 1 铁矿 2 石头 3 粮食
	reqObj := &model.TransformReq{}
	rspObj := &model.TransformRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	roleRes := logic.RoleResService.GetRoleRes(role.RId)
	if roleRes == nil {
		rsp.Body.Code = constant.DBError
		return
	}
	//做交易的时候 主城做交易
	rc := logic.RoleCityService.GetMainCity(role.RId)
	if rc == nil {
		rsp.Body.Code = constant.DBError
		return
	}

	level := logic.CityFacilityService.GetFacilityLevel(rc.CityId,gameConfig.JiShi)
	if level <= 0 {
		rsp.Body.Code = constant.NotHasJiShi
		return
	}

	roleRes.Wood -= reqObj.From[0]
	roleRes.Wood += reqObj.To[0]
	roleRes.Iron -= reqObj.From[1]
	roleRes.Iron += reqObj.To[1]
	roleRes.Stone -= reqObj.From[2]
	roleRes.Stone += reqObj.To[2]
	roleRes.Grain -= reqObj.From[3]
	roleRes.Grain += reqObj.To[3]

	roleRes.SyncExecute()
}
