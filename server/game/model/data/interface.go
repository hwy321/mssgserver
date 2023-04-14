package data

var GetYield func(rid int) Yield

var GetUnion func(rid int) int

var GetParentId func(rid int) int

var MapResTypeLevel func(x, y int) (bool, int8, int8)

var GetMainMembers func(uid int) []int

var GetRoleNickName func(rid int) string
