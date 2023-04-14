package controller

import (
	"mssgserver/constant"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/logic"
	"mssgserver/server/game/middleware"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
)

var SkillController = &skillController{}
type skillController struct {

}

func (s *skillController) Router(router *net.Router)  {
	g := router.Group("skill")
	g.Use(middleware.Log())
	g.AddRouter("list",s.list,middleware.CheckRole())
}

func (s *skillController) list(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//查找战报表 得出数据
	rspObj := &model.SkillListRsp{}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	role, err := req.Conn.GetProperty("role")
	if err != nil {
		rsp.Body.Code = constant.SessionInvalid
		return
	}
	rid := role.(*data.Role).RId

	skills,err := logic.SkillService.GetSkills(rid)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.List = skills
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK
}
