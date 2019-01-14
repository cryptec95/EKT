package param

import "github.com/EducationEKT/EKT/core/types"

// 委托人节点由官方暂时托管, 节点数量由21个暂时改为3个, 于2018年10月31日21:05:19进行升级,区块高度重置为0
var MainNet = []types.Peer{
	{"909333d195d473124036954c4c91e5364becf022af244e1f9054deece671b666", "58.83.148.230", 19951, 4},
	{"f28b3c29b038972b11362db3da6615caa30504944c999c22e73b162f8f2531aa", "58.83.148.231", 19951, 4},
	{"f6679c55bb45938dd00c2967834a79a26335066b7e816ce3ed330e8c4ceed0d1", "58.83.148.232", 19951, 4},
}
