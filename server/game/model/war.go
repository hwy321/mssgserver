package model

type WarReportRsp struct {
	List	[]WarReport `json:"list"`
}

//正常做业务 数据库的对应的结构体 -》 data目录
//客户端需要的数据 model目录

type WarReport struct {
	Id             		int    		`json:"id"`
	AttackRid      		int    		`json:"a_rid"`
	DefenseRid     		int    		`json:"d_rid"`
	BegAttackArmy  		string 		`json:"b_a_army"`
	BegDefenseArmy 		string 		`json:"b_d_army"`
	EndAttackArmy  		string 		`json:"e_a_army"`
	EndDefenseArmy 		string 		`json:"e_d_army"`
	BegAttackGeneral  	string    	`json:"b_a_general"`
	BegDefenseGeneral 	string    	`json:"b_d_general"`
	EndAttackGeneral  	string    	`json:"e_a_general"`
	EndDefenseGeneral 	string    	`json:"e_d_general"`
	Result				int      	`json:"result"`	//0失败，1打平，2胜利
	Rounds				string		`json:"rounds"` //回合
	AttackIsRead   		bool   		`json:"a_is_read"`
	DefenseIsRead  		bool   		`json:"d_is_read"`
	DestroyDurable 		int    		`json:"destroy"`
	Occupy         		int    		`json:"occupy"`
	X              		int    		`json:"x"`
	Y              		int    		`json:"y"`
	CTime          		int  		`json:"ctime"`
}

type WarReadReq struct {
	Id		uint		`json:"id"`	//0全部已读
}

type WarReadRsp struct {
	Id		uint		`json:"id"`
}

