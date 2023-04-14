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
	"time"
)

var DefaultNationMapController = &nationMapController{}
type nationMapController struct {

}

func (r *nationMapController) Router(router *net.Router)  {
	g := router.Group("nationMap")
	g.Use(middleware.Log())
	g.AddRouter("config",r.config)
	g.AddRouter("scanBlock",r.scanBlock,middleware.CheckRole())
	g.AddRouter("build",r.build,middleware.CheckRole())
	g.AddRouter("giveUp",r.giveUp,middleware.CheckRole())
	g.AddRouter("upBuild",r.upBuild,middleware.CheckRole())
}

func (r *nationMapController) config(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//reqObj := &model.ConfigReq{}
	rspObj := &model.ConfigRsp{}

	cfgs := gameConfig.MapBuildConf.Cfg

	rspObj.Confs = make([]model.Conf, len(cfgs))
	for index, v := range cfgs {
		rspObj.Confs[index].Type = v.Type
		rspObj.Confs[index].Name = v.Name
		rspObj.Confs[index].Level = v.Level
		rspObj.Confs[index].Defender = v.Defender
		rspObj.Confs[index].Durable = v.Durable
		rspObj.Confs[index].Grain = v.Grain
		rspObj.Confs[index].Iron = v.Iron
		rspObj.Confs[index].Stone = v.Stone
		rspObj.Confs[index].Wood = v.Wood
	}
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	rsp.Body.Code = constant.OK
	rsp.Body.Msg = rspObj

}

func (r *nationMapController) scanBlock(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.ScanBlockReq{}
	rspObj := &model.ScanRsp{}
	mapstructure.Decode(req.Body.Msg,reqObj)
	rsp.Body.Seq = req.Body.Seq
	rsp.Body.Name = req.Body.Name
	rsp.Body.Code = constant.OK
	//扫描角色建筑
	mrb,err := logic.RoleBuildService.ScanBlock(reqObj)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.MRBuilds = mrb
	//扫描角色城池
	mrc,err := logic.RoleCityService.ScanBlock(reqObj)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.MCBuilds = mrc
	role,_ := req.Conn.GetProperty("role")
	rl := role.(*data.Role)
	//扫描玩家军队
	armys,err := logic.ArmyService.ScanBlock(rl.RId,reqObj)
	if err != nil {
		rsp.Body.Code = err.(*common.MyError).Code()
		return
	}
	rspObj.Armys = armys

	rsp.Body.Msg = rspObj
}

func (n *nationMapController) build(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//把表中现有的土地性质进行对应的变更
	reqObj := &model.BuildReq{}
	rspObj := &model.BuildRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	x := reqObj.X
	y := reqObj.Y

	rspObj.X = x
	rspObj.Y = y

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)
	//要只要建立哪些建筑
	rb,ok := logic.RoleBuildService.PositionBuild(x,y)
	if !ok {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	if rb.RId != role.RId {
		rsp.Body.Code = constant.BuildNotMe
		return
	}
	//判断是否能建立  特征 如果资源为0的土地 是不能建立建筑的
	if !rb.IsCanRes() || rb.IsBusy() {
		rsp.Body.Code = constant.CanNotBuildNew
		return
	}
	//判断建筑是否到达上限
	cnt := logic.RoleBuildService.RoleFortressCnt(role.RId)
	if cnt >= gameConfig.Base.Build.FortressLimit {
		rsp.Body.Code = constant.CanNotBuildNew
		return
	}
	//找到要建造的要塞 所需要的资源
	cfg,ok := gameConfig.MapBCConf.BuildConfig(reqObj.Type,1)
	if !ok {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	need := logic.RoleResService.TryUseNeed(role.RId, cfg.Need)
	if !need {
		rsp.Body.Code = constant.ResNotEnough
		return
	}
	//构建建筑
	rb.BuildOrUp(*cfg)
	rb.SyncExecute()

}

func (n *nationMapController) giveUp(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	//放弃意味着 此用户的所属进行变更 土地还是要还原成系统
	//放弃有时间，需要等时间完成之后 才能放弃
	//给一个放弃时间，然后通知客户端倒计时
	//开一个协程 一直监听 是否放弃时间到了，如果到了 执行放弃程序
	reqObj := &model.GiveUpReq{}
	rspObj := &model.GiveUpRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	x := reqObj.X
	y := reqObj.Y

	rspObj.X = x
	rspObj.Y = y

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	//判断土地是否是当前角色的
	build, ok := logic.RoleBuildService.PositionBuild(x, y)
	if !ok {
		rsp.Body.Code = constant.InvalidParam
		return
	}
	if build.RId != role.RId {
		rsp.Body.Code = constant.BuildNotMe
		return
	}
	rsp.Body.Code = logic.RoleBuildService.GiveUp(build)
}

func (n *nationMapController) upBuild(req *net.WsMsgReq, rsp *net.WsMsgRsp) {
	reqObj := &model.UpBuildReq{}
	rspObj := &model.UpBuildRsp{}
	mapstructure.Decode(req.Body.Msg, reqObj)
	rsp.Body.Msg = rspObj
	rsp.Body.Code = constant.OK

	x := reqObj.X
	y := reqObj.Y

	rspObj.X = x
	rspObj.Y = y

	r, _ := req.Conn.GetProperty("role")
	role := r.(*data.Role)

	if logic.RoleBuildService.BuildIsRId(x, y, role.RId) == false{
		rsp.Body.Code = constant.BuildNotMe
		return
	}

	b, ok := logic.RoleBuildService.PositionBuild(x, y)
	if ok == false {
		rsp.Body.Code = constant.BuildNotMe
		return
	}

	//判断上次是否升级完成
	if time.Now().Unix() >= b.EndTime.Unix() {
		//升级完成
		b.Level = b.OPLevel
	}

	if b.IsHaveModifyLVAuth() == false || b.IsInGiveUp() || b.IsBusy(){
		rsp.Body.Code = constant.CanNotUpBuild
		return
	}


	cfg, ok := gameConfig.MapBCConf.BuildConfig(b.Type, b.Level+1)
	if ok == false{
		rsp.Body.Code = constant.InvalidParam
		return
	}

	ok = logic.RoleResService.TryUseNeed(role.RId, cfg.Need)
	if !ok {
		rsp.Body.Code = constant.ResNotEnough
		return
	}
	b.BuildOrUp(*cfg)
	b.SyncExecute()
	rspObj.Build = b.ToModel().(model.MapRoleBuild)
}

