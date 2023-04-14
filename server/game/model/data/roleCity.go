package data

import (
	"log"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/model"
	"sync"
	"time"
)

var RoleCityDao = &mapRoleCityDao{
	rcChan: make(chan *MapRoleCity, 100),
}

type mapRoleCityDao struct {
	rcChan chan *MapRoleCity
}

func (m *mapRoleCityDao) running() {
	for {
		select {
		case rc := <-m.rcChan:
			if rc.CityId > 0 {
				//where  city_id = ?
				_, err := db.Engine.Table(rc).ID(rc.CityId).Update(rc)
				if err != nil {
					log.Println("mapRoleCityDao running error", err)
				}
			}
		}
	}
}

func init() {
	go RoleCityDao.running()
}

type MapRoleCity struct {
	mutex      sync.Mutex `xorm:"-"`
	CityId     int        `xorm:"cityId pk autoincr"`
	RId        int        `xorm:"rid"`
	Name       string     `xorm:"name" validate:"min=4,max=20,regexp=^[a-zA-Z0-9_]*$"`
	X          int        `xorm:"x"`
	Y          int        `xorm:"y"`
	IsMain     int8       `xorm:"is_main"`
	CurDurable int        `xorm:"cur_durable"`
	CreatedAt  time.Time  `xorm:"created_at"`
	OccupyTime time.Time  `xorm:"occupy_time"`
}

func (m *MapRoleCity) TableName() string {
	return "tb_map_role_city_1"
}

func (m *MapRoleCity) ToModel() interface{} {
	p := model.MapRoleCity{}
	p.X = m.X
	p.Y = m.Y
	p.CityId = m.CityId
	p.UnionId = GetUnion(m.RId)
	p.UnionName = ""
	p.ParentId = 0
	p.MaxDurable = 1000
	p.CurDurable = m.CurDurable
	p.Level = 1
	p.RId = m.RId
	p.Name = m.Name
	p.IsMain = m.IsMain == 1
	p.OccupyTime = m.OccupyTime.UnixNano() / 1e6
	return p
}

func (m *MapRoleCity) IsWarFree() bool {
	//占领时间
	occupyTime := m.OccupyTime.Unix()
	cur := time.Now().Unix()
	if cur-occupyTime < gameConfig.Base.Build.WarFree {
		return true
	}
	return false
}

func (m *MapRoleCity) DurableChange(change int) {
	m.CurDurable += change
	if m.CurDurable < 0 {
		m.CurDurable = 0
	}
}

func (m *MapRoleCity) SyncExecute() {
	RoleCityDao.rcChan <- m
	m.Push()
}

/* 推送同步 begin */
func (m *MapRoleCity) IsCellView() bool {
	return true
}

func (m *MapRoleCity) IsCanView(rid, x, y int) bool {
	return true
}

func (m *MapRoleCity) BelongToRId() []int {
	return []int{m.RId}
}

func (m *MapRoleCity) PushMsgName() string {
	return "roleCity.push"
}

func (m *MapRoleCity) Position() (int, int) {
	return m.X, m.Y
}

func (m *MapRoleCity) TPosition() (int, int) {
	return -1, -1
}
func (m *MapRoleCity) Push() {
	net.Mgr.Push(m)
}
