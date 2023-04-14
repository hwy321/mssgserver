package controller

import (
	"github.com/mitchellh/mapstructure"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/logic/pos"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"mssgserver/utils"
	"time"
)

var DefaultRoleController = &RoleController{}
type RoleController struct {

}

func (r *RoleController) Router(router *net.Router)  {
	g := router.Group("role")
	g.Use(middleware.Log())
	g.AddRouter("create",r.create)
	g.AddRouter("enterServer",r.enterServer)
	g.AddRouter("myProperty",r.myProperty,middleware.CheckRole())
	g.AddRouter("posTagList",r.posTagList,middleware.CheckRole())
	g.AddRouter("upPosition",r.upPosition,middleware.CheckRole())
}

func (r *RoleController) enterServer(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//进入的游戏的逻辑
	//Session 需要验证是否合法 合法的情况下 可以取出登录的用户id
	//根据用户id 去查询对应的游戏角色，如果有 就继续 没有 提示无角色
	//根据角色id 查询角色拥有的资源 roleRes，如果资源有 返回，没有 初始化资源
	reqObj := &model.EnterServerReq{}
	rspObj := &model.EnterServerRsp{}
	err := mapstructure.Decode(req.Body.Msg,reqObj)
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	if err != nil {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	session := reqObj.Session
	_,claim,err := utils.ParseToken(session)
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	uid := claim.Uid
	err = logic.RoleService.EnterServer(uid,rspObj,req)
	if err != nil {
		rspObj.Time = time.Now().UnixNano()/1e6
		rsp.Body.Msg = rspObj
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}

func (r *RoleController) myProperty(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//分别根据角色id 去查询 军队 资源 建筑 城池 武将
	role, err := req.Conn.GetProperty("role")
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	rid := role.(*data.Role).RId
	rspObj := &model.MyRolePropertyRsp{}
	//资源
	rspObj.RoleRes,err = logic.RoleService.GetRoleRes(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	//城池
	rspObj.Citys,err = logic.RoleCityService.GetRoleCitys(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	//建筑
	rspObj.MRBuilds,err = logic.RoleBuildService.GetBuilds(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	//军队
	rspObj.Armys,err = logic.ArmyService.GetArmys(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	//武将
	rspObj.Generals,err = logic.GeneralService.GetGenerals(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}

func (r *RoleController) posTagList(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	rspObj := &model.PosTagListRsp{}

	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	//去 角色属性 表去查询
	role, err := req.Conn.GetProperty("role")
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	rid := role.(*data.Role).RId
	pts,err := logic.RoleAttrService.GetTagList(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.PosTags = pts
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}

func (r *RoleController) create(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.CreateRoleReq{}
	rspObj := &model.CreateRoleRsp{}
	mapstructure.Decode(req.Body.Msg,reqObj)

	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	role := &data.Role{}
	ok,err := db.Engine.Where("uid=?",reqObj.UId).Get(role)
	if err!=nil  {
		rsp.Body.Code = constant.DBError
		return
	}
	if ok {
		rsp.Body.Code = constant.RoleAlreadyCreate
		return
	}
	role.UId = reqObj.UId
	role.Sex = reqObj.Sex
	role.NickName = reqObj.NickName
	role.Balance = 0
	role.HeadId = reqObj.HeadId
	role.CreatedAt = time.Now()
	role.LoginTime = time.Now()
	_,err = db.Engine.InsertOne(role)
	if err!=nil  {
		rsp.Body.Code = constant.DBError
		return
	}
	rspObj.Role = role.ToModel().(model.Role)
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
}

func (rc *RoleController) upPosition(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.UpPositionReq{}
	rspObj := &model.UpPositionRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	rspObj.X = reqObj.X
	rspObj.Y = reqObj.Y

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	pos.RPMgr.Push(reqObj.X,reqObj.Y,role.RId)

}
