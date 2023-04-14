package logic

import (
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/server/common"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"sync"
	"time"
)

var CoalitionService = &coalitionService{
	unions: make(map[int]*data.Coalition),
}
type coalitionService struct {
	mutex  sync.RWMutex
	unions map[int]*data.Coalition
}
//members [1,2,3,4,5]  json的字符串
func (c *coalitionService) Load()  {
	rr := make([]*data.Coalition, 0)
	err := db.Engine.Table(new(data.Coalition)).Where("state=?", data.UnionRunning).Find(&rr)
	if err != nil {
		log.Println("coalitionService load error",err)
	}
	for _, v := range rr {
		c.unions[v.Id] = v
	}
}

func (c *coalitionService) List() ([]model.Union, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	uns := make([]model.Union,0)
	for _,v := range c.unions{
		mas := make([]model.Major,0)
		if role := RoleService.Get(v.Chairman);role != nil{
			ma := model.Major{
				RId: role.RId,
				Name: role.NickName,
				Title: model.UnionChairman,
			}
			mas = append(mas,ma)
		}
		if role := RoleService.Get(v.ViceChairman);role != nil{
			ma := model.Major{
				RId: role.RId,
				Name: role.NickName,
				Title: model.UnionChairman,
			}
			mas = append(mas,ma)
		}

		union := v.ToModel().(model.Union)
		union.Major = mas
		uns = append(uns, union)
	}
	return uns,nil
}

func (c *coalitionService) ListCoalition() []*data.Coalition {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	uns := make([]*data.Coalition,0)
	for _,v := range c.unions{
		uns = append(uns,v)
	}
	return uns
}

func (c *coalitionService) Get(id int) model.Union {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	coa,ok := c.unions[id]
	if ok {
		union := coa.ToModel().(model.Union)
		mas := make([]model.Major,0)
		if role := RoleService.Get(coa.Chairman);role != nil{
			ma := model.Major{
				RId: role.RId,
				Name: role.NickName,
				Title: model.UnionChairman,
			}
			mas = append(mas,ma)
		}
		if role := RoleService.Get(coa.ViceChairman);role != nil{
			ma := model.Major{
				RId: role.RId,
				Name: role.NickName,
				Title: model.UnionChairman,
			}
			mas = append(mas,ma)
		}
		union.Major = mas
		return union
	}
	return model.Union{}
}


func (c *coalitionService) GetById(id int) *data.Coalition {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	coa,ok := c.unions[id]
	if ok {
		return coa
	}
	return nil
}

func (c *coalitionService) GetCoalition(id int) *data.Coalition{
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	coa,ok := c.unions[id]
	if ok {
		return coa
	}
	return nil
}

func (c *coalitionService) GetListApply(unionId int, state int) ([]model.ApplyItem,error) {
	applys := make([]data.CoalitionApply,0)
	err := db.Engine.Table(new(data.CoalitionApply)).
		Where("union_id=? and state=?",unionId,state).
		Find(&applys)
	if err != nil {
		log.Println("coalitionService GetListApply find error",err)
		return nil,common.New(constant.DBError,"数据库错误")
	}
	ais := make([]model.ApplyItem,0)
	for _,v := range applys{
		var ai model.ApplyItem
		ai.Id = v.Id
		role := RoleService.Get(v.RId)
		ai.NickName = role.NickName
		ai.RId = role.RId
		ais = append(ais,ai)
	}
	return ais,nil
}

func (c *coalitionService) GetMainMembers(uid int) []int {
	rids := make([]int,0)
	coalition := c.GetCoalition(uid)
	if coalition != nil {
		chairman := coalition.Chairman
		viceChairman := coalition.ViceChairman
		rids = append(rids,chairman,viceChairman)
	}
	return rids
}

func (c *coalitionService) Create(name string, rid int) (*data.Coalition, bool){
	m := &data.Coalition{Name: name, Ctime: time.Now(),
		CreateId: rid, Chairman: rid, State: data.UnionRunning, MemberArray: []int{rid}}

	_, err := db.Engine.Table(new(data.Coalition)).InsertOne(m)
	if err == nil {

		c.mutex.Lock()
		c.unions[m.Id] = m
		c.mutex.Unlock()

		return m, true
	}else{
		return nil, false
	}
}

func (c *coalitionService) MemberEnter(rid int, unionId int) {
	attr, ok := RoleAttrService.TryCreateRA(rid)
	if ok {
		attr.UnionId = unionId
		if attr.ParentId == unionId{
			c.DelChild(unionId, attr.RId)
		}
	}

	if rcs, ok := RoleCityService.GetByRId(rid); ok {
		for _, rc := range rcs {
			rc.SyncExecute()
		}
	}
}

func (c *coalitionService) NewCreateLog(opNickName string, unionId int, opRId int) {
	ulog := &data.CoalitionLog{
		UnionId: unionId,
		OPRId: opRId,
		TargetId: 0,
		State: data.UnionOpCreate,
		Des: opNickName + " 创建了联盟",
		Ctime: time.Now(),
	}

	db.Engine.InsertOne(ulog)
}

func (c *coalitionService) DelChild(unionId int, rid int) {
	attr := RoleAttrService.Get(rid)
	if attr != nil {
		attr.ParentId = 0
		attr.SyncExecute()
	}
}

func  (c *coalitionService)  NewJoin(targetNickName string, unionId int, opRId int, targetId int) {
	ulog := &data.CoalitionLog{
		UnionId: unionId,
		OPRId: opRId,
		TargetId: targetId,
		State: data.UnionOpJoin,
		Des: targetNickName + " 加入了联盟",
		Ctime: time.Now(),
	}
	db.Engine.InsertOne(ulog)
}

func (c *coalitionService) MemberExit(rid int) {
	if ra := RoleAttrService.Get(rid); ra != nil {
		ra.UnionId = 0
	}

	if rcs, ok := RoleCityService.GetByRId(rid); ok {
		for _, rc := range rcs {
			rc.SyncExecute()
		}
	}
}

func (c *coalitionService) NewExit(opNickName string, unionId int, opRId int) {
	ulog := &data.CoalitionLog{
		UnionId: unionId,
		OPRId: opRId,
		TargetId: opRId,
		State: data.UnionOpExit,
		Des: opNickName + " 退出了联盟",
		Ctime: time.Now(),
	}
	db.Engine.InsertOne(ulog)
}

func (c *coalitionService) Dismiss(unionId int) {
	u := c.GetById(unionId)
	if u !=nil {
		for _, rid := range u.MemberArray {
			c.MemberExit(rid)
		}
		u.State = data.UnionDismiss
		u.MemberArray = []int{}
		u.SyncExecute()
	}
}

func (c *coalitionService) NewDismiss(opNickName string, unionId int, opRId int) {
	ulog := &data.CoalitionLog{
		UnionId: unionId,
		OPRId: opRId,
		TargetId: 0,
		State: data.UnionOpDismiss,
		Des: opNickName + " 解散了联盟",
		Ctime: time.Now(),
	}
	db.Engine.InsertOne(ulog)
}

func (c *coalitionService) NewAppoint(opNickName string, targetNickName string,
	unionId int, opRId int, targetId int, memberType int) {

		title := ""
		if memberType == model.UnionChairman{
			title = "盟主"
		}else if memberType == model.UnionViceChairman{
			title = "副盟主"
		}else{
			title = "普通成员"
		}

		ulog := &data.CoalitionLog{
			UnionId: unionId,
			OPRId: opRId,
			TargetId: targetId,
			State: data.UnionOpAppoint,
			Des: opNickName + " 将 " + targetNickName + " 任命为 " + title,
			Ctime: time.Now(),
		}
		db.Engine.InsertOne(ulog)
}

func (c *coalitionService) NewModNotice(opNickName string, unionId int, opRId int) {
	ulog := &data.CoalitionLog{
		UnionId: unionId,
		OPRId: opRId,
		TargetId: 0,
		State: data.UnionOpModNotice,
		Des: opNickName + " 修改了公告",
		Ctime: time.Now(),
	}
	db.Engine.InsertOne(ulog)
}


