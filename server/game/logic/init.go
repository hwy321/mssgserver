package logic

import "mssgserver/server/game/model/data"

func BeforeInit()  {
	data.GetYield = RoleResService.GetYield
	data.GetUnion = RoleAttrService.GetUnion
	data.GetParentId = RoleAttrService.GetParentId
	data.MapResTypeLevel = RoleBuildService.MapResTypeLevel
	data.GetMainMembers = CoalitionService.GetMainMembers
	data.GetRoleNickName = RoleService.GetRoleNickName
}
