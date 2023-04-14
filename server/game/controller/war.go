package controller

import (
	"github.com/mitchellh/mapstructure"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
)

var WarController = &warController{}
type warController struct {

}

func (w *warController) Router(router *net.Router)  {
	g := router.Group("war")
	g.Use(middleware.Log())
	g.AddRouter("report",w.report,middleware.CheckRole())
	g.AddRouter("read",w.read,middleware.CheckRole())
}

func (w *warController) report(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//查找战报表 得出数据
	rspObj := &model.WarReportRsp{}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	role, err := req.Conn.GetProperty("role")
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	rid := role.(*data.Role).RId

	reports,err := logic.WarService.GetWarReports(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.List = reports
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK
}

func (w *warController) read(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//查找战报表 得出数据
	reqObj := &model.WarReadReq{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rspObj := &model.WarReadRsp{}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj
	role, _ := req.Conn.GetProperty("role")
	rid := role.(*data.Role).RId

	rspObj.Id = reqObj.Id

	if reqObj.Id > 0 {
		//更新某一个战报
		wr := &data.WarReport{
			AttackIsRead:true,
			DefenseIsRead: true,
		}
		db.Engine.Table(wr).Where("id=? and a_rid=?",reqObj.Id,rid).Cols("a_is_read").Update(wr)
		db.Engine.Table(wr).Where("id=? and d_rid=?",reqObj.Id,rid).Cols("d_is_read").Update(wr)
	}else{
		//更新所有的战报
		//更新某一个战报
		wr := &data.WarReport{
			AttackIsRead:true,
			DefenseIsRead:true,
		}
		db.Engine.Table(wr).Where("a_rid=?",rid).Cols("a_is_read").Update(wr)
		db.Engine.Table(wr).Where("d_rid=?",rid).Cols("d_is_read").Update(wr)
	}
}
