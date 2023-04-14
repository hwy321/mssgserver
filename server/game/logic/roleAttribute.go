package logic

import (
	"encoding/json"
	"log"
	"mssgserver/constant"
	"mssgserver/db"
	"mssgserver/net"
	"mssgserver/server/common"
	"mssgserver/server/game/model"
	"mssgserver/server/game/model/data"
	"sync"
	"xorm.io/xorm"
)

var RoleAttrService = &roleAttrService{
	attrs: make(map[int]*data.RoleAttribute),
}
type roleAttrService struct {
	mutex sync.RWMutex
	attrs map[int]*data.RoleAttribute
}

func ( r *roleAttrService) Load()  {
	ras := make([]*data.RoleAttribute,0)
	err := db.Engine.Table(new(data.RoleAttribute)).Find(&ras)
	if err != nil {
		log.Println("roleAttrService Load err",err)
	}
	for _,v := range ras{
		r.attrs[v.RId] = v
	}
	//查询所有的联盟，进行匹配
	uns := CoalitionService.ListCoalition()
	for _, un := range uns{
		for _, ma := range un.MemberArray{
			ra,ok := r.attrs[ma]
			if ok {
				ra.UnionId = un.Id
				r.attrs[ma] = ra
			}
		}
	}

}

func (r *roleAttrService) TryCreate(rid int, req *net.WsMsgReq) error {
	role := &data.RoleAttribute{}
	ok,err := db.Engine.Table(role).Where("rid=?",rid).Get(role)
	if err != nil {
		log.Println("查询角色属性出错",err)
		return common.New(constant.DBError,"数据库出错")
	}
	if ok {
		//r.mutex.Lock()
		//defer r.mutex.Unlock()
		//r.attrs[rid] = role
		return 	nil
	}else{
		//初始化
		role.RId = rid
		role.UnionId = 0
		role.ParentId = 0
		role.PosTags = ""
		if session := req.Context.Get("dbSession");session != nil {
			_,err = session.(*xorm.Session).Table(role).Insert(role)
		}else{
			_,err = db.Engine.Table(role).Insert(role)
		}
		if err != nil {
			log.Println("插入角色属性出错",err)
			return common.New(constant.DBError,"数据库出错")
		}
		r.mutex.Lock()
		defer r.mutex.Unlock()
		r.attrs[rid] = role
	}
	return nil
}


func (ra *roleAttrService) TryCreateRA(rid int) (*data.RoleAttribute, bool){
	attr := ra.Get(rid)
	if attr != nil {
		return attr, true
	}else{
		ra.mutex.Lock()
		defer ra.mutex.Unlock()
		attr := ra.create(rid)
		return attr, attr != nil
	}
}

func (ra *roleAttrService) create(rid int) *data.RoleAttribute {
	roleAttr := &data.RoleAttribute{RId: rid, ParentId: 0, UnionId: 0}
	if _ , err := db.Engine.Insert(roleAttr); err != nil {
		return nil
	}else{
		ra.attrs[rid] = roleAttr
		return roleAttr
	}
}

func (r *roleAttrService) GetTagList(rid int) ([]model.PosTag,error){
	ra,ok := r.attrs[rid]
	if !ok {
		ra = &data.RoleAttribute{}
		var err error
		ok,err = db.Engine.Table(ra).Where("rid=?",rid).Get(ra)
		if err != nil {
			log.Println("GetTagList",err)
			return nil, common.New(constant.DBError,"数据库错误")
		}
	}

	posTags := make([]model.PosTag,0)
	if ok {
		tags := ra.PosTags
		if tags != "" {
			err := json.Unmarshal([]byte(tags),&posTags)
			if err != nil {
				return nil, common.New(constant.DBError,"数据库错误")
			}
		}
	}
	return posTags, nil
}

func (r *roleAttrService) Get(rid int) *data.RoleAttribute {

	r.mutex.RLock()
	defer r.mutex.RUnlock()
	ra,ok := r.attrs[rid]
	if ok {
		return ra
	}
	return nil
}


func (r *roleAttrService) GetUnion(rid int) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	ra ,ok := r.attrs[rid]
	if ok{
		return ra.UnionId
	}
	return 0
}

func (r *roleAttrService) GetParentId(rid int) int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()
	ra ,ok := r.attrs[rid]
	if ok{
		return ra.ParentId
	}
	return 0
}

func (ra *roleAttrService) IsHasUnion(rid int) bool {
	ra.mutex.RLock()
	r, ok := ra.attrs[rid]
	ra.mutex.RUnlock()
	if ok {
		return r.UnionId != 0
	}else {
		return  false
	}
}