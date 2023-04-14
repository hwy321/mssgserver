package data

import (
	"log"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/game/model"
	"time"
)

var RoleAttrDao = &roleAttrDao{
	raChan: make(chan *RoleAttribute, 100),
}

type roleAttrDao struct {
	raChan chan *RoleAttribute
}

func (r *roleAttrDao) running() {
	for {
		select {
		case rr := <-r.raChan:
			_, err := db.Engine.
				Table(new(RoleAttribute)).
				ID(rr.Id).
				Cols("parent_id", "collect_times", "last_collect_time", "pos_tags").
				Update(rr)
			if err != nil {
				log.Println("RoleResDao update error", err)
			}
		}
	}

}

func init() {
	go RoleAttrDao.running()
}

type RoleAttribute struct {
	Id              int            `xorm:"id pk autoincr"`
	RId             int            `xorm:"rid"`
	UnionId         int            `xorm:"-"`                 //联盟id
	ParentId        int            `xorm:"parent_id"`         //上级id（被沦陷）
	CollectTimes    int8           `xorm:"collect_times"`     //征收次数
	LastCollectTime time.Time      `xorm:"last_collect_time"` //最后征收的时间
	PosTags         string         `xorm:"pos_tags"`          //位置标记
	PosTagArray     []model.PosTag `xorm:"-"`
}

func (r *RoleAttribute) TableName() string {
	return "tb_role_attribute_1"
}

func (r *RoleAttribute) SyncExecute() {
	RoleAttrDao.raChan <- r
	r.Push()
}

/* 推送同步 begin */
func (r *RoleAttribute) IsCellView() bool {
	return false
}

func (r *RoleAttribute) IsCanView(rid, x, y int) bool {
	return false
}

func (r *RoleAttribute) BelongToRId() []int {
	return []int{r.RId}
}

func (r *RoleAttribute) PushMsgName() string {
	return "roleAttr.push"
}

func (r *RoleAttribute) ToModel() interface{} {
	return nil
}

func (r *RoleAttribute) Position() (int, int) {
	return -1, -1
}

func (r *RoleAttribute) TPosition() (int, int) {
	return -1, -1
}

func (r *RoleAttribute) Push() {
	net.Mgr.Push(r)
}
