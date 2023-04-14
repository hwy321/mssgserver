package controller

import (
	"github.com/mitchellh/mapstructure"
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"time"
)

var UnionController = &unionController{}
type unionController struct {

}

func (u *unionController) Router(router *net.Router)  {
	g := router.Group("union")
	g.Use(middleware.Log())
	g.AddRouter("list",u.list,middleware.CheckRole())
	g.AddRouter("info",u.info,middleware.CheckRole())
	g.AddRouter("applyList",u.applyList,middleware.CheckRole())
	g.AddRouter("create",u.create, middleware.CheckRole())
	g.AddRouter("join",u.join,middleware.CheckRole())
	g.AddRouter("verify",u.verify,middleware.CheckRole())
	g.AddRouter("member",u.member,middleware.CheckRole())
	g.AddRouter("notice",u.notice,middleware.CheckRole())
	g.AddRouter("exit",u.exit,middleware.CheckRole())
	g.AddRouter("dismiss",u.dismiss,middleware.CheckRole())
	g.AddRouter("appoint",u.appoint,middleware.CheckRole())
	g.AddRouter("log", u.log,middleware.CheckRole())
	g.AddRouter("modNotice", u.modNotice,middleware.CheckRole())
}

func (u *unionController) list(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//查询数据库 把所有的联盟 查询出来
	rspObj := &model.ListRsp{}
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK
	uns,err := logic.CoalitionService.List()
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.List = uns
	rsp.Body.Msg = rspObj
}

func (u *unionController) info(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj:= &model.InfoReq{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rspObj := &model.InfoRsp{}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj

	un := logic.CoalitionService.Get(reqObj.Id)
	rspObj.Info = un
	rspObj.Id = un.Id
}

func (u *unionController) applyList(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//根据联盟id 去查询申请列表，rid申请人，你角色表 查询详情即可
	// state 0 正在申请 1 拒绝 2 同意
	//什么人能看到申请列表 只有盟主和副盟主能看到申请列表
	reqObj := &model.ApplyReq{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rspObj := &model.ApplyRsp{}
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj

	r,_ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	//查询联盟
	un := logic.CoalitionService.GetCoalition(reqObj.Id)
	if un == nil {
		rsp.Body.Code = constant.DBError
		return
	}
	if un.Chairman != role.RId && un.ViceChairman != role.RId {
		rspObj.Id = reqObj.Id
		rspObj.Applys = make([]model.ApplyItem,0)
		return
	}

	ais,err := logic.CoalitionService.GetListApply(reqObj.Id,0)
	if err != nil {
		rsp.Body.Code = constant.DBError
		return
	}
	rspObj.Id = reqObj.Id
	rspObj.Applys = ais
}

func (u *unionController) create(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.CreateReq{}
	rspObj := &model.CreateRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK
	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	rspObj.Name = reqObj.Name

	has := logic.RoleAttrService.IsHasUnion(role.RId)
	if has {
		rsp.Body.Code = constant.UnionAlreadyHas
		return
	}

	c, ok := logic.CoalitionService.Create(reqObj.Name, role.RId)
	if ok {
		rspObj.Id = c.Id
		logic.CoalitionService.MemberEnter(role.RId, c.Id)
		logic.CoalitionService.NewCreateLog(role.NickName, c.Id, role.RId)
	}else{
		rsp.Body.Code = constant.UnionCreateError
	}
}

func (u *unionController) join(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.JoinReq{}
	rspObj := &model.JoinRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj

	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	has := logic.RoleAttrService.IsHasUnion(role.RId)
	if has {
		rsp.Body.Code = constant.UnionAlreadyHas
		return
	}

	union:= logic.CoalitionService.GetById(reqObj.Id)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}
	if len(union.MemberArray) >= gameConfig.Base.Union.MemberLimit{
		rsp.Body.Code = constant.PeopleIsFull
		return
	}

	//判断当前是否已经有申请
	has, _ = db.Engine.Table(data.CoalitionApply{}).Where(
		"union_id=? and state=? and rid=?",
		reqObj.Id, model.UnionUntreated, role.RId).Get(&data.CoalitionApply{})
	if has {
		rsp.Body.Code = constant.HasApply
		return
	}

	//写入申请列表
	apply := &data.CoalitionApply{
		RId:     role.RId,
		UnionId: reqObj.Id,
		Ctime:   time.Now(),
		State:   model.UnionUntreated}

	_, err := db.Engine.InsertOne(apply)
	if err != nil{
		rsp.Body.Code = constant.DBError
		return
	}

	//推送主、副盟主
	apply.SyncExecute()
}

func (un *unionController) verify(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.VerifyReq{}
	rspObj := &model.VerifyRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rspObj.Id = reqObj.Id
	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)


	apply := &data.CoalitionApply{}
	ok, err := db.Engine.Table(data.CoalitionApply{}).Where(
		"id=? and state=?", reqObj.Id, model.UnionUntreated).Get(apply)
	if ok && err == nil{
		targetRole := logic.RoleService.Get(apply.RId)
		if targetRole == nil{
			rsp.Body.Code = constant.RoleNotExist
			return
		}

		if u := logic.CoalitionService.GetById(apply.UnionId); u != nil {

			if u.Chairman != role.RId && u.ViceChairman != role.RId {
				rsp.Body.Code = constant.PermissionDenied
				return
			}

			if len(u.MemberArray) >= gameConfig.Base.Union.MemberLimit{
				rsp.Body.Code = constant.PeopleIsFull
				return
			}

			if ok := logic.RoleAttrService.IsHasUnion(apply.RId); ok {
				rsp.Body.Code = constant.UnionAlreadyHas
			}else{
				if reqObj.Decide == model.UnionAdopt {
					//同意
					c := logic.CoalitionService.GetById(apply.UnionId)
					if c != nil {
						c.MemberArray = append(c.MemberArray, apply.RId)
						logic.CoalitionService.MemberEnter(apply.RId, apply.UnionId)
						c.SyncExecute()
						logic.CoalitionService.NewJoin(targetRole.NickName, apply.UnionId, role.RId, apply.RId)
					}
				}
			}
			apply.State = reqObj.Decide
			db.Engine.Table(apply).ID(apply.Id).Cols("state").Update(apply)
		}else{
			rsp.Body.Code = constant.UnionNotFound
			return
		}

	}else{
		rsp.Body.Code = constant.InvalidParam
	}
}

func (u *unionController) member(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.MemberReq{}
	rspObj := &model.MemberRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rspObj.Id = reqObj.Id
	rsp.Body.Code = constant.OK

	union := logic.CoalitionService.GetById(reqObj.Id)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	rspObj.Members = make([]model.Member, 0)
	for _, rid := range union.MemberArray {
		if role := logic.RoleService.Get(rid); role != nil {
			m := model.Member{RId: role.RId, Name: role.NickName }
			if main := logic.RoleCityService.GetMainCity(role.RId); main != nil {
				m.X = main.X
				m.Y = main.Y
			}

			if rid == union.Chairman {
				m.Title = model.UnionChairman
			}else if rid == union.ViceChairman {
				m.Title = model.UnionViceChairman
			}else {
				m.Title = model.UnionCommon
			}
			rspObj.Members = append(rspObj.Members, m)
		}
	}
}

func (u *unionController) notice(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.NoticeReq{}
	rspObj := &model.NoticeRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	union := logic.CoalitionService.GetById(reqObj.Id)
	if union == nil {
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	rspObj.Text = union.Notice
}

func (u *unionController) exit(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.ExitReq{}
	rspObj := &model.ExitRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj

	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	if ok := logic.RoleAttrService.IsHasUnion(role.RId); ok == false {
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	attribute := logic.RoleAttrService.Get(role.RId)
	union := logic.CoalitionService.GetById(attribute.UnionId)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	//盟主不能退出
	if union.Chairman == role.RId {
		rsp.Body.Code = constant.UnionNotAllowExit
		return
	}

	for i, rid := range union.MemberArray {
		if rid == role.RId{
			union.MemberArray = append(union.MemberArray[:i], union.MemberArray[i+1:]...)
		}
	}

	if union.ViceChairman == role.RId{
		union.ViceChairman = 0
	}

	logic.CoalitionService.MemberExit(role.RId)
	union.SyncExecute()
	logic.CoalitionService.NewExit(role.NickName, union.Id, role.RId)
}

func (u *unionController) dismiss(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.DismissReq{}
	rspObj := &model.DismissRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj

	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	if ok := logic.RoleAttrService.IsHasUnion(role.RId); ok == false {
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	attribute := logic.RoleAttrService.Get(role.RId)
	union := logic.CoalitionService.GetById(attribute.UnionId)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	//盟主才能解散
	if union.Chairman != role.RId {
		rsp.Body.Code = constant.PermissionDenied
		return
	}
	unionId := attribute.UnionId
	logic.CoalitionService.Dismiss(unionId)

	logic.CoalitionService.NewDismiss(role.NickName, unionId, role.RId)
}

func (u *unionController) appoint(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.AppointReq{}
	rspObj := &model.AppointRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rspObj.RId = reqObj.RId

	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	if ok := logic.RoleAttrService.IsHasUnion(role.RId); ok == false {
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	opAr := logic.RoleAttrService.Get(role.RId)
	union := logic.CoalitionService.GetById(opAr.UnionId)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	if union.Chairman != role.RId {
		rsp.Body.Code = constant.PermissionDenied
		return
	}

	targetRole := logic.RoleService.Get(reqObj.RId)
	if targetRole ==nil {
		rsp.Body.Code = constant.RoleNotExist
		return
	}

	target := logic.RoleAttrService.Get(reqObj.RId)
	if target != nil {
		if target.UnionId == union.Id{
			if reqObj.Title == model.UnionViceChairman {
				union.ViceChairman = reqObj.RId
				rspObj.Title = reqObj.Title
				union.SyncExecute()
				logic.CoalitionService.NewAppoint(role.NickName, targetRole.NickName, union.Id, role.RId, targetRole.RId, reqObj.Title)
			}else if reqObj.Title == model.UnionCommon {
				if union.ViceChairman == reqObj.RId{
					union.ViceChairman = 0
				}
				rspObj.Title = reqObj.Title
				logic.CoalitionService.NewAppoint(role.NickName, targetRole.NickName, union.Id, role.RId, targetRole.RId, reqObj.Title)
			}else{
				rsp.Body.Code = constant.InvalidParam
			}
		}else{
			rsp.Body.Code = constant.NotBelongUnion
		}
	}else{
		rsp.Body.Code = constant.NotBelongUnion
	}
}

func (u *unionController) log(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.LogReq{}
	rspObj := &model.LogRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rspObj.Logs = make([]model.UnionLog, 0)

	rsp.Body.Code = constant.OK

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	opAr := logic.RoleAttrService.Get(role.RId)
	union:= logic.CoalitionService.GetById(opAr.UnionId)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	//开始查询日志
	logs := make([]*data.CoalitionLog, 0)
	err := db.Engine.Table(data.CoalitionLog{}).Where(
		"union_id=?", union.Id).Desc("ctime").Find(&logs)
	if err != nil{
		log.Println(err)
	}

	for _, cLog := range logs {
		rspObj.Logs = append(rspObj.Logs, cLog.ToModel().(model.UnionLog))
	}
}

func (u *unionController) modNotice(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.ModNoticeReq{}
	rspObj := &model.ModNoticeRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj

	rsp.Body.Code = constant.OK
	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	if len(reqObj.Text) > 200 {
		rsp.Body.Code = constant.ContentTooLong
		return
	}

	if ok := logic.RoleAttrService.IsHasUnion(role.RId); ok == false {
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	attribute:= logic.RoleAttrService.Get(role.RId)
	union := logic.CoalitionService.GetById(attribute.UnionId)
	if union == nil{
		rsp.Body.Code = constant.UnionNotFound
		return
	}

	if union.Chairman != role.RId && union.ViceChairman != role.RId {
		rsp.Body.Code = constant.PermissionDenied
		return
	}

	rspObj.Text = reqObj.Text
	rspObj.Id = union.Id
	union.Notice = reqObj.Text
	union.SyncExecute()

	logic.CoalitionService.NewModNotice(role.NickName, union.Id, role.RId)
}
