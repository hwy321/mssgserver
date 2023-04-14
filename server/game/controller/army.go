package controller

import (
	"github.com/mitchellh/mapstructure"
	"mssgserver/constant"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/gameConfig/general"
	"mssgserver/server/game/global"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"time"
)

var DefaultArmyController = &ArmyController{}
type ArmyController struct {

}

func (a *ArmyController) Router(router *net.Router)  {
	g := router.Group("army")
	g.Use(middleware.Log())
	g.AddRouter("myList",a.myList,middleware.CheckRole())
	g.AddRouter("dispose",a.dispose,middleware.CheckRole())
	g.AddRouter("conscript",a.conscript,middleware.CheckRole())
	g.AddRouter("myOne",a.myOne,middleware.CheckRole())
	g.AddRouter("assign",a.assign,middleware.CheckRole())
}

func (a *ArmyController) myList(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.ArmyListReq{}
	rspObj := &model.ArmyListRsp{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	role, err := req.Conn.GetProperty("role")
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	rid := role.(*data.Role).RId

	armys , err := logic.ArmyService.GetArmysByCity(rid,reqObj.CityId)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.Armys = armys
	rspObj.CityId = reqObj.CityId
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}

func (a *ArmyController) dispose(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//判断参数是否符合要求
	reqObj := &model.DisposeReq{}
	rspObj := &model.DisposeRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	if reqObj.Order < 0 || reqObj.Position < -1 || reqObj.Position > 2 || reqObj.Order > 5 {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	rc , ok := logic.RoleCityService.Get(reqObj.CityId)
	if !ok {
		rsp.Body.Code = constant.CityNotExist
		return
	}
	if role.RId != rc.RId {
		rsp.Body.Code = constant.CityNotMe
		return
	}
	//判断校场的等级 3 只能配置3个队伍
	level := logic.CityFacilityService.GetFacilityLevel(reqObj.CityId,gameConfig.JiaoChang)
	if level <= 0 || reqObj.Order > level {
		rsp.Body.Code = constant.ArmyNotEnough
		return
	}
	//武将是否存在
	newGen,ok := logic.GeneralService.Get(reqObj.GeneralId)
	if !ok  {
		rsp.Body.Code = constant.GeneralNotFound
		return
	}
	//武将是否是当前角色的
	if newGen.RId != role.RId {
		rsp.Body.Code = constant.GeneralNotMe
		return
	}
	//查询当前位置的军队 没有就创建
	army,ok := logic.ArmyService.GetCreate(reqObj.CityId,role.RId,reqObj.Order)
	//需要判断军队是否是在城外 如果在城外不能进行上下阵
	if (army.FromX > 0 && army.FromX != rc.X )|| (army.FromY > 0 && army.FromY != rc.Y) {
		rsp.Body.Code = constant.ArmyIsOutside
		return
	}
	//上下阵处理
	if reqObj.Position == -1 {
		//下阵
		for position,g := range army.Gens{
			if g != nil && g.Id == reqObj.GeneralId {
				//检测武将是否在征兵中
				if !army.PositionCanModify(position) {
					rsp.Body.Code = constant.GeneralBusy
					return
				}
				army.GeneralArray[position] = 0
				army.SoldierArray[position] = 0
				army.Gens[position] = nil
				army.SyncExecute()
			}

		}
		newGen.CityId = 0
		newGen.Order = 0
		newGen.SyncExecute()
	}else{
		//上阵
		//检测武将是否在征兵中
		if !army.PositionCanModify(reqObj.Position) {
			rsp.Body.Code = constant.GeneralBusy
			return
		}
		//武将已经上阵过了
		if newGen.CityId != 0{
			rsp.Body.Code = constant.GeneralBusy
			return
		}
		if logic.ArmyService.IsRepeat(role.RId, newGen.CfgId){
			rsp.Body.Code = constant.GeneralRepeat
			return
		}
		level := logic.CityFacilityService.GetFacilityLevel(rc.CityId, gameConfig.TongShuaiTing)
		if reqObj.Position == 2 && (  level < reqObj.Order) {
			rsp.Body.Code = constant.TongShuaiNotEnough
			return
		}
		//判断cost
		cost := general.General.Cost(newGen.CfgId)
		for _,g := range army.Gens {
			if g != nil {
				cost += general.General.Cost(g.CfgId)
			}
		}
		cityCost := logic.RoleCityService.GetCityCost(reqObj.CityId)
		if cityCost < cost {
			rsp.Body.Code = constant.CostNotEnough
			return
		}
		oldG := army.Gens[reqObj.Position]
		if oldG != nil {
			//旧的下阵
			oldG.CityId = 0
			oldG.Order = 0
			oldG.SyncExecute()
		}
		army.GeneralArray[reqObj.Position] = reqObj.GeneralId
		army.SoldierArray[reqObj.Position] = 0
		army.Gens[reqObj.Position] = newGen
		newGen.Order = reqObj.Order
		newGen.CityId = reqObj.CityId
		newGen.SyncExecute()
	}
	army.FromX = rc.X
	army.FromY = rc.Y
	army.SyncExecute()
	//队伍
	rspObj.Army = army.ToModel().(model.Army)
}

func (a *ArmyController) conscript(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.ConscriptReq{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rspObj := &model.ConscriptRsp{}
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	//征兵 army 更新征兵的数量和征兵的完成时间 以及状态
	//判断逻辑 征兵能不能进行 资源是否足够  参数是否正常 募兵所的设施 等级>=1

	//参数是否合法
	if reqObj.ArmyId <= 0 || len(reqObj.Cnts) != 3 {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	if reqObj.Cnts[0] <0 || reqObj.Cnts[1] <0 || reqObj.Cnts[2] <0 {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	//角色
	r ,_ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	//军队是否存在
	army := logic.ArmyService.Get(reqObj.ArmyId)
	if army == nil {
		rsp.Body.Code = constant.ArmyNotFound
		return
	}
	if role.RId != army.RId {
		rsp.Body.Code = constant.ArmyNotMe
		return
	}
	//募兵所
	level := logic.CityFacilityService.GetFacilityLevel(army.CityId,gameConfig.MBS)
	if level <= 0 {
		rsp.Body.Code = constant.BuildMBSNotFound
		return
	}

	////判断位置是否可以征兵
	for pos,v :=range army.Gens{
		if reqObj.Cnts[pos] > 0 {
			if v == nil {
				rsp.Body.Code = constant.InvalidParam
				return
			}
			if !army.PositionCanModify(pos) {
				rsp.Body.Code = constant.GeneralBusy
				return
			}
		}
	}
	//判断征兵数量是否合法 判断资源是否合法
	for pos,v :=range army.Gens{
		if v == nil {
			continue
		}
		lv := v.Level
		gLevel := general.GeneralBasic.GetLevel(lv)
		add := logic.CityFacilityService.GetSoldier(army.CityId)
		if gLevel.Soldiers + add < reqObj.Cnts[pos] + army.SoldierArray[pos] {
			rsp.Body.Code = constant.InvalidParam
			return
		}
		var cur = time.Now().Unix()
		army.ConscriptCntArray[pos]= reqObj.Cnts[pos]
		army.ConscriptTimeArray[pos] = int64(reqObj.Cnts[pos] * gameConfig.Base.ConScript.CostTime) + cur - 2
	}
	var total int
	for _, v:= range reqObj.Cnts{
		total += v
	}
	//资源是否合法
	needRes := gameConfig.NeedRes{
		Decree: 0,
		Gold: gameConfig.Base.ConScript.CostGold * total,
		Wood: gameConfig.Base.ConScript.CostWood * total,
		Stone: gameConfig.Base.ConScript.CostStone * total,
		Iron: gameConfig.Base.ConScript.CostIron * total,
		Grain: gameConfig.Base.ConScript.CostGrain * total,
	}
	ok := logic.RoleResService.TryUseNeed(role.RId,needRes)
	if !ok {
		rsp.Body.Code = constant.ResNotEnough
		return
	}
	//足了 就可以更新了
	army.Cmd = data.ArmyCmdConscript
	army.SyncExecute()
	rspObj.Army = army.ToModel().(model.Army)

	res := logic.RoleResService.GetRoleRes(role.RId)
	if res != nil {
		rspObj.RoleRes = res.ToModel().(model.RoleRes)
	}
}

func (a *ArmyController) myOne(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.ArmyOneReq{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rspObj := &model.ArmyOneRsp{}
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	//角色
	r ,_ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	city,ok := logic.RoleCityService.Get(reqObj.CityId)
	if !ok {
		rsp.Body.Code = constant.CityNotExist
		return
	}
	if role.RId != city.RId {
		rsp.Body.Code = constant.CityNotMe
		return
	}
	army := logic.ArmyService.GetArmy(reqObj.CityId,reqObj.Order)
	rspObj.Army = army.ToModel().(model.Army)
}

func (a *ArmyController) assign(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.AssignArmyReq{}
	rspObj := &model.AssignArmyRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	//查询军队
	army := logic.ArmyService.Get(reqObj.ArmyId)
	if army == nil {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	if role.RId != army.RId {
		rsp.Body.Code = constant.ArmyNotMe
		return
	}
	if reqObj.Cmd == data.ArmyCmdBack{
		rsp.Body.Code = a.back(army)
	}else if reqObj.Cmd == data.ArmyCmdAttack {
		rsp.Body.Code = a.attack(reqObj, army, role)
	}else if reqObj.Cmd == data.ArmyCmdDefend {
		rsp.Body.Code = a.defend(reqObj, army, role)
	}else if reqObj.Cmd == data.ArmyCmdReclamation {
		rsp.Body.Code = a.reclamation(reqObj, army, role)
	}else if reqObj.Cmd == data.ArmyCmdTransfer {
		rsp.Body.Code = a.transfer(reqObj, army, role)
	}
	rspObj.Army = army.ToModel().(model.Army)
}

func (a *ArmyController) back(army *data.Army) int {
	//从哪里来 回哪里去
	if army.Cmd == data.ArmyCmdAttack||
		army.Cmd == data.ArmyCmdDefend ||
		army.Cmd == data.ArmyCmdReclamation {
		logic.ArmyService.ArmyBack(army)
	}else{
		city, ok := logic.RoleCityService.Get(army.CityId)
		if ok {
			if city.X != army.FromX || city.Y != army.FromY{
				logic.ArmyService.ArmyBack(army)
			}
		}
	}

	return constant.OK
}

func (a *ArmyController) defend(req *model.AssignArmyReq, army *data.Army, role *data.Role) int {
	if code := a.pre(req,army,role);code != constant.OK {
		return code
	}
	if logic.IsCanDefend(req.X,req.Y,role.RId) {
		return constant.BuildCanNotDefend
	}
	//计算体力
	power := gameConfig.Base.General.CostPhysicalPower
	for _,v := range army.Gens{
		if v == nil {
			continue
		}
		if v.PhysicalPower < power {
			return constant.PhysicalPowerNotEnough
		}
	}
	//扣减体力
	logic.GeneralService.TryUsePhysicalPower(army,power)

	army.State = data.ArmyRunning

	army.Cmd = req.Cmd
	army.ToX = req.X
	army.ToY = req.Y
	army.Start = time.Now()
	//实际情况 要根据最慢速度进行计算
	army.End = time.Now().Add(time.Second*10)

	//后台监听程序 一个在监听是否有部队调动并且到了指定的位置
	logic.ArmyService.PushAction(army)
	return constant.OK
}

func (a *ArmyController) reclamation(obj *model.AssignArmyReq, army *data.Army, role *data.Role) int {
	return constant.OK
}

func (a *ArmyController) transfer(req *model.AssignArmyReq, army *data.Army, role *data.Role) int {
	if code := a.pre(req,army,role);code != constant.OK {
		return code
	}
	if army.FromX == req.X && army.FromY == req.Y{
		return constant.CanNotTransfer
	}
	build, ok := logic.RoleBuildService.PositionBuild(req.X, req.Y)
	if ok {
		if build.RId != role.RId {
			return constant.BuildNotMe
		}
	}else{
		return constant.BuildNotMe
	}
	if build.Level <= 0 || build.IsHasTransferAuth() == false {
		return constant.CanNotTransfer
	}
	cnt := 0
	if build.IsRoleFortress() {
		cnt = gameConfig.MapBCConf.GetHoldArmyCnt(build.Type, build.Level)
	}else{
		cnt = 5
	}

	if cnt > logic.ArmyService.BelongPosArmyCnt(build.RId, build.X, build.Y) {
		//计算体力
		power := gameConfig.Base.General.CostPhysicalPower
		for _,v := range army.Gens{
			if v == nil {
				continue
			}
			if v.PhysicalPower < power {
				return constant.PhysicalPowerNotEnough
			}
		}
		//扣减体力
		logic.GeneralService.TryUsePhysicalPower(army,power)

		//调动和屯垦 需要消耗令牌
		if req.Cmd == data.ArmyCmdTransfer {
			cost := gameConfig.Base.General.ReclamationCost
			if logic.RoleResService.DecreeIsEnough(army.RId, cost) == false{
				return constant.DecreeNotEnough
			}else{
				logic.RoleResService.TryUseDecree(army.RId, cost)
			}
		}
		army.State = data.ArmyRunning

		army.Cmd = req.Cmd
		army.ToX = req.X
		army.ToY = req.Y
		army.Start = time.Now()
		//实际情况 要根据最慢速度进行计算
		army.End = time.Now().Add(time.Second*10)

		//后台监听程序 一个在监听是否有部队调动并且到了指定的位置
		logic.ArmyService.PushAction(army)
		return constant.OK
	}else{
		return constant.HoldIsFull
	}
}

func (a *ArmyController) pre(reqObj *model.AssignArmyReq, army *data.Army, role *data.Role) int {
	//判断是否合法
	if reqObj.X < 0 || reqObj.X > global.MapWith || reqObj.Y < 0 || reqObj.Y > global.MapHeight {
		return constant.InvalidParam
	}
	//是否能出战
	if !army.IsCanOutWar() {
		return constant.ArmyBusy
	}
	if !army.IsIdle() {
		return constant.ArmyBusy
	}
	//判断此土地是否是能攻击的类型 比如山地
	nm,ok := gameConfig.MapRes.ToPositionMap(reqObj.X,reqObj.Y)
	if !ok {
		return constant.InvalidParam
	}
	//山地不能移动到此
	if nm.Type == 0{
		return constant.InvalidParam
	}
	return constant.OK
}


func (a *ArmyController) attack(req *model.AssignArmyReq, army *data.Army, role *data.Role) int {
	//占领
	//参数验证
	if code := a.pre(req,army,role);code != constant.OK {
		return code
	}
	//是否免战 比如刚占领 不能被攻击
	if logic.IsWarFree(req.X,req.Y) {
		return constant.BuildWarFree
	}
	//自己的城池 和联盟的城池 都不能攻击
	if logic.IsCanDefend(req.X,req.Y,role.RId) {
		return constant.BuildCanNotAttack
	}
	//计算体力
	power := gameConfig.Base.General.CostPhysicalPower
	for _,v := range army.Gens{
		if v == nil {
			continue
		}
		if v.PhysicalPower < power {
			return constant.PhysicalPowerNotEnough
		}
	}
	//扣减体力
	logic.GeneralService.TryUsePhysicalPower(army,power)

	army.State = data.ArmyRunning

	army.Cmd = req.Cmd
	army.ToX = req.X
	army.ToY = req.Y
	army.Start = time.Now()
	//实际情况 要根据最慢速度进行计算
	army.End = time.Now().Add(time.Second*10)

	//后台监听程序 一个在监听是否有部队调动并且到了指定的位置
	logic.ArmyService.PushAction(army)
	return constant.OK
}
