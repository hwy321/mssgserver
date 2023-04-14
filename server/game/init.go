package game

import (
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/game/controller"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/gameConfig/general"
	"mssgserver/server/game/logic"
)

var Router = &net.Router{}
func Init()  {
	db.TestDB()
	//加载基础配置
	gameConfig.Base.Load()
	//加载地图的资源配置
	gameConfig.MapBuildConf.Load()
	//加载地图单元格配置
	gameConfig.MapRes.Load()
	//加载城池设施配置
	gameConfig.FacilityConf.Load()
	//加载武将配置信息
	general.General.Load()
	general.GeneralBasic.Load()
	//加载技能配置信息
	gameConfig.Skill.Load()
	gameConfig.MapBCConf.Load()

	general.GeneralArms.Load()

	logic.BeforeInit()

	//加载所有的建筑信息
	logic.RoleBuildService.Load()
	//加载所有的城池信息
	logic.RoleCityService.Load()
	//加载联盟的所有信息
	logic.CoalitionService.Load()
	//加载所有的角色属性
	logic.RoleAttrService.Load()
	//加载角色资源
	logic.RoleResService.Load()
	logic.ArmyService.Init()

	initRouter()
}

func initRouter() {
	controller.DefaultRoleController.Router(Router)
	controller.DefaultNationMapController.Router(Router)
	controller.DefaultGeneralController.Router(Router)
	controller.DefaultArmyController.Router(Router)
	controller.WarController.Router(Router)
	controller.SkillController.Router(Router)
	controller.InteriorController.Router(Router)
	controller.UnionController.Router(Router)
	controller.CityController.Router(Router)
}
