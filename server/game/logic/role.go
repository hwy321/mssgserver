package logic

import (
	"fmt"
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"mssgserver/utils"
	"time"
)

var RoleService = &roleService{}

type roleService struct {
}

func (r *roleService) EnterServer(uid int, rsp *model.EnterServerRsp, req *net.WsMsgReq) error {
	role := &data.Role{}
	session := db.Engine.NewSession()
	defer session.Close()
	if err := session.Begin(); err != nil {
		log.Println("事务开启出错", err)
		return common.New(constant.DBError, "数据库出错")
	}
	req.Context.Set("dbSession", session)
	ok, err := db.Engine.Table(role).Where("uid=?", uid).Get(role)
	if err != nil {
		log.Println("查询角色出错", err)
		return common.New(constant.DBError, "数据库出错")
	}
	if ok {
		rid := role.RId
		roleRes := &data.RoleRes{}
		ok, err = db.Engine.Table(roleRes).Where("rid=?", rid).Get(roleRes)
		if err != nil {
			log.Println("查询角色资源出错", err)
			return common.New(constant.DBError, "数据库出错")
		}
		if !ok {
			roleRes.RId = rid
			roleRes.Gold = gameConfig.Base.Role.Gold
			roleRes.Decree = gameConfig.Base.Role.Decree
			roleRes.Grain = gameConfig.Base.Role.Grain
			roleRes.Iron = gameConfig.Base.Role.Iron
			roleRes.Stone = gameConfig.Base.Role.Stone
			roleRes.Wood = gameConfig.Base.Role.Wood
			_, err := session.Table(roleRes).Insert(roleRes)
			if err != nil {
				log.Println("插入角色资源出错", err)
				return common.New(constant.DBError, "数据库出错")
			}
		}
		rsp.RoleRes = roleRes.ToModel().(model.RoleRes)
		rsp.Role = role.ToModel().(model.Role)
		rsp.Time = time.Now().UnixNano() / 1e6
		token, _ := utils.Award(rid)
		rsp.Token = token
		req.Conn.SetProperty("role", role)
		// 初始化玩家属性
		if err := RoleAttrService.TryCreate(rid, req); err != nil {
			session.Rollback()
			return common.New(constant.DBError, "数据库错误")
		}
		//初始化城池
		if err := RoleCityService.InitCity(rid, role.NickName, req); err != nil {
			session.Rollback()
			return common.New(constant.DBError, "数据库错误")
		}

	} else {
		fmt.Println("角色不存在")
		return common.New(constant.RoleNotExist, "角色不存在")
	}
	err = session.Commit()
	if err != nil {
		log.Println("事务提交出错", err)
		return common.New(constant.DBError, "数据库出错")
	}
	net.Mgr.RoleEnter(req.Conn, role.RId)
	return nil
}

func (r *roleService) GetRoleRes(rid int) (model.RoleRes, error) {
	roleRes := &data.RoleRes{}
	ok, err := db.Engine.Table(roleRes).Where("rid=?", rid).Get(roleRes)
	if err != nil {
		log.Println("查询角色资源出错", err)
		return model.RoleRes{}, common.New(constant.DBError, "数据库出错")
	}
	if ok {
		return roleRes.ToModel().(model.RoleRes), nil
	}
	return model.RoleRes{}, common.New(constant.DBError, "角色资源不存在")
}

func (r *roleService) Get(rid int) *data.Role {
	role := &data.Role{}
	ok, err := db.Engine.Table(role).Where("rid=?", rid).Get(role)
	if err != nil {
		log.Println("查询角色出错", err)
		return nil
	}
	if ok {
		return role
	}
	return nil
}

func (r *roleService) GetRoleNickName(rid int) string {
	role := &data.Role{}
	ok, err := db.Engine.Table(role).Where("rid=?", rid).Get(role)
	if err != nil {
		log.Println("查询角色出错", err)
		return ""
	}
	if ok {
		return role.NickName
	}
	return ""
}
