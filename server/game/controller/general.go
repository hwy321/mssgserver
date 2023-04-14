package controller

import (
	"github.com/mitchellh/mapstructure"
	"mssgserver/constant"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
)

var DefaultGeneralController = &GeneralController{}
type GeneralController struct {

}

func (r *GeneralController) Router(router *net.Router)  {
	g := router.Group("general")
	g.Use(middleware.Log())
	g.AddRouter("myGenerals",r.myGenerals,middleware.CheckRole())
	g.AddRouter("drawGeneral",r.drawGeneral,middleware.CheckRole())
}

func (r *GeneralController) myGenerals(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//查询武将的时候 角色拥有的武将 查询出来即可
	// 如果初始化 进入游戏 武将没有 需要随机三个武将 很多游戏 初始化武将是一样的

	rspObj := &model.MyGeneralRsp{}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	role, err := req.Conn.GetProperty("role")
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	rid := role.(*data.Role).RId

	gs , err := logic.GeneralService.GetGenerals(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.Generals = gs
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}

func (r *GeneralController) drawGeneral(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//1. 计算抽卡花费的金钱
	//2. 判断金钱是否足够
	//3. 抽卡的次数 + 已有的武将 卡池是否足够
	//4. 随机生成武将即可（之前有实现）
	//5. 金币的扣除
	reqObj := &model.DrawGeneralReq{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rspObj := &model.DrawGeneralRsp{}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
	role, _ := req.Conn.GetProperty("role")
	rid := role.(*data.Role).RId
	cost := gameConfig.Base.General.DrawGeneralCost * reqObj.DrawTimes
	if !logic.RoleResService.IsEnoughGold(rid,cost) {
		rsp.Body.Code = constant.GoldNotEnough
		return
	}
	limit := gameConfig.Base.General.Limit

	gs,err := logic.GeneralService.GetGenerals(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	if len(gs) + reqObj.DrawTimes > limit {
		rsp.Body.Code = constant.OutGeneralLimit
		return
	}
	mgs := logic.GeneralService.Draw(rid,reqObj.DrawTimes)
	logic.RoleResService.CostGold(rid,cost)
	rspObj.Generals = mgs
}
