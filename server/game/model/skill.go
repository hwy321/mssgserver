package model

type Skill struct {
	Id             	int 	`json:"id"`
	CfgId          	int 	`json:"cfgId"`
	Generals 		[]int 	`json:"generals"`
}


type SkillListRsp struct {
	List []Skill `json:"list"`
}
