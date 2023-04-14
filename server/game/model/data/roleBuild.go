package data

import (
	"log"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/game/gameConfig"
	"mssgserver/server/game/model"
	"mssgserver/utils"
	"time"
)

const (
	MapBuildSysFortress = 50 //系统要塞
	MapBuildSysCity     = 51 //系统城市
	MapBuildFortress    = 56 //玩家要塞
)

var MapRoleBuildDao = &mapRoleBuildDao{
	rbChan: make(chan *MapRoleBuild, 100),
}

type mapRoleBuildDao struct {
	rbChan chan *MapRoleBuild
}

func (m *mapRoleBuildDao) running() {
	for {
		select {
		case rb := <-m.rbChan:
			if rb.Id > 0 {
				_, err := db.Engine.Table(rb).ID(rb.Id).
					Cols("rid", "type", "level", "op_level", "cur_durable", "max_durable", "occupy_time", "end_time", "giveUp_time").
					Update(rb)
				if err != nil {
					log.Println("mapRoleBuildDao running error", err)
				}
			}
		}
	}
}

func init() {
	go MapRoleBuildDao.running()
}

type MapRoleBuild struct {
	Id         int       `xorm:"id pk autoincr"`
	RId        int       `xorm:"rid"`
	Type       int8      `xorm:"type"`
	Level      int8      `xorm:"level"`
	OPLevel    int8      `xorm:"op_level"` //操作level
	X          int       `xorm:"x"`
	Y          int       `xorm:"y"`
	Name       string    `xorm:"name"`
	Wood       int       `xorm:"-"`
	Iron       int       `xorm:"-"`
	Stone      int       `xorm:"-"`
	Grain      int       `xorm:"-"`
	Defender   int       `xorm:"-"`
	CurDurable int       `xorm:"cur_durable"`
	MaxDurable int       `xorm:"max_durable"`
	OccupyTime time.Time `xorm:"occupy_time"`
	EndTime    time.Time `xorm:"end_time"` //建造或升级完的时间
	GiveUpTime int64     `xorm:"giveUp_time"`
}

func (m *MapRoleBuild) TableName() string {
	return "tb_map_role_build_1"
}

/* 推送同步 begin */
func (m *MapRoleBuild) IsCellView() bool {
	return true
}

func (m *MapRoleBuild) IsCanView(rid, x, y int) bool {
	return true
}

func (m *MapRoleBuild) BelongToRId() []int {
	return []int{m.RId}
}

func (m *MapRoleBuild) PushMsgName() string {
	return "roleBuild.push"
}

func (m *MapRoleBuild) Position() (int, int) {
	return m.X, m.Y
}

func (m *MapRoleBuild) TPosition() (int, int) {
	return -1, -1
}

func (m *MapRoleBuild) Push() {
	net.Mgr.Push(m)
}

func (m *MapRoleBuild) ToModel() interface{} {
	p := model.MapRoleBuild{}
	p.RNick = "111"
	p.UnionId = 0
	p.UnionName = ""
	p.ParentId = 0
	p.X = m.X
	p.Y = m.Y
	p.Type = m.Type
	p.RId = m.RId
	p.Name = m.Name

	p.OccupyTime = m.OccupyTime.UnixNano() / 1e6
	p.GiveUpTime = m.GiveUpTime * 1000
	p.EndTime = m.EndTime.UnixNano() / 1e6

	p.CurDurable = m.CurDurable
	p.MaxDurable = m.MaxDurable
	p.Defender = m.Defender
	p.Level = m.Level
	p.OPLevel = m.OPLevel
	return p
}

func (m *MapRoleBuild) Init() {
	cfg := gameConfig.MapBuildConf.BuildConfig(m.Type, m.Level)
	if cfg != nil {
		m.Name = cfg.Name
		m.Level = cfg.Level
		m.Type = cfg.Type
		m.Wood = cfg.Wood
		m.Iron = cfg.Iron
		m.Stone = cfg.Stone
		m.Grain = cfg.Grain
		m.MaxDurable = cfg.Durable
		m.CurDurable = cfg.Durable
		m.Defender = cfg.Defender
	}
}

func (m *MapRoleBuild) IsWarFree() bool {
	//占领时间
	occupyTime := m.OccupyTime.Unix()
	cur := time.Now().Unix()
	if cur-occupyTime < gameConfig.Base.Build.WarFree {
		return true
	}
	return false
}

func (m *MapRoleBuild) Reset() {
	ok, t, level := MapResTypeLevel(m.X, m.Y)
	if ok {
		if cfg := gameConfig.MapBuildConf.BuildConfig(t, level); cfg != nil {
			m.Name = cfg.Name
			m.Level = cfg.Level
			m.Type = cfg.Type
			m.Wood = cfg.Wood
			m.Iron = cfg.Iron
			m.Stone = cfg.Stone
			m.Grain = cfg.Grain
			m.MaxDurable = cfg.Durable
			m.CurDurable = cfg.Durable
			m.Defender = cfg.Defender
		}
		m.GiveUpTime = 0
		m.RId = 0
		m.EndTime = time.Time{}
		m.OPLevel = m.Level
		m.CurDurable = utils.MinInt(m.MaxDurable, m.CurDurable)
	}
}

func (m *MapRoleBuild) SyncExecute() {
	MapRoleBuildDao.rbChan <- m
	//push数据到前端
	m.Push()
}

func (m *MapRoleBuild) IsCanRes() bool {
	return m.Wood > 0 || m.Stone > 0 || m.Grain > 0 || m.Iron > 0
}

func (m *MapRoleBuild) IsBusy() bool {
	if m.Level != m.OPLevel {
		return true
	}
	return false
}

func (m *MapRoleBuild) BuildOrUp(cfg gameConfig.BCLevelCfg) {
	m.Type = cfg.Type
	m.Level = cfg.Level - 1
	m.Name = cfg.Name
	m.OPLevel = cfg.Level
	m.GiveUpTime = 0

	m.Wood = 0
	m.Iron = 0
	m.Stone = 0
	m.Grain = 0
	m.EndTime = time.Now().Add(time.Duration(cfg.Time) * time.Second)
}

func (m *MapRoleBuild) IsHasTransferAuth() bool {
	return m.Type == MapBuildFortress || m.Type == MapBuildSysFortress
}

func (m *MapRoleBuild) IsRoleFortress() bool {
	return m.Type == gameConfig.MapBuildFortress
}

//玩家要塞才能升级
func (m *MapRoleBuild) IsHaveModifyLVAuth() bool {
	return m.Type == MapBuildFortress
}

func (m *MapRoleBuild) IsInGiveUp() bool {
	return m.GiveUpTime != 0
}
