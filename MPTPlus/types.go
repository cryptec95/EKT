package MPTPlus

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"github.com/EducationEKT/EKT/core/types"
	"github.com/EducationEKT/EKT/db"
	"sort"
)

/*
*TrieSonInfo存储的是当前节点的下一个节点的信息
*如果当前节点是叶子节点,则Sons的长度为1,且TrieSonInfo.Hash是当前路径key的Value值的hash
*如果当前节点不是叶子节点,则Sons的长度大于等于1,存储的是子节点的Hash值和PathValue
 */
type TrieSonInfo struct {
	Hash      types.HexBytes `json:"hash"`
	PathValue types.HexBytes `json:"pathValue"`
}

/*
*TrieNode存储的是当前节点的一些信息,包括PathValue 是否是叶子节点,子节点信息等等
*strings.Join(pathValue,"")就是用户要存储的key
 */
type TrieNode struct {
	Sons      SortedSon      `json:"sons"`
	Leaf      bool           `json:"leaf"`
	Root      bool           `json:"root"`
	PathValue types.HexBytes `json:"pathValue"`
}

type MTP struct {
	Root types.HexBytes
	DB   db.IKVDatabase
}

func (mtp *MTP) UnmarshalJSON(data []byte) error {
	data = data[:len(data)-1][1:]
	bytes, err := hex.DecodeString(string(data))
	mtp.Root = bytes
	mtp.DB = db.GetDBInst()
	return err
}

func (mtp MTP) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf(`"%s"`, hex.EncodeToString(mtp.Root))), nil
}

func MTP_Tree(db db.IKVDatabase, root []byte) *MTP {
	trie := &MTP{DB: db, Root: root}
	if len(root) != 0 {
		trie.Root = root
	} else {
		node := TrieNode{
			Root:      true,
			Leaf:      false,
			PathValue: nil,
			Sons:      *new(SortedSon),
		}
		trie.Root, _ = trie.SaveNode(node)
	}
	return trie
}

func NewMTP(db db.IKVDatabase) *MTP {
	node := TrieNode{
		Root:      true,
		Leaf:      false,
		PathValue: nil,
		Sons:      *new(SortedSon),
	}
	mtp := MTP_Tree(db, nil)
	mtp.Root, _ = mtp.SaveNode(node)
	return mtp
}

type SortedSon []TrieSonInfo

func (sonInfo SortedSon) Len() int {
	return len(sonInfo)
}

func (sonInfo SortedSon) Swap(i, j int) {
	sonInfo[i], sonInfo[j] = sonInfo[j], sonInfo[i]
}

func (sonInfo SortedSon) Less(i, j int) bool {
	length := len(sonInfo[i].PathValue)
	if len(sonInfo[j].PathValue) < length {
		length = len(sonInfo[j].PathValue)
	}
	for m := 0; m < length; m++ {
		if sonInfo[i].PathValue[m] < sonInfo[j].PathValue[m] {
			return true
		} else if sonInfo[i].PathValue[m] > sonInfo[j].PathValue[m] {
			return false
		}
	}
	return true
}

func (node *TrieNode) AddSon(hash, pathValue []byte) {
	if nil == node.Sons {
		node.Sons = *new(SortedSon)
	}
	for _, son := range node.Sons {
		if bytes.Equal(son.PathValue, pathValue) {
			node.DeleteSon(pathValue)
		}
	}
	node.Sons = append(node.Sons, TrieSonInfo{Hash: hash, PathValue: pathValue})
	sort.Sort(node.Sons)
}

func (node *TrieNode) DeleteSon(pathValue []byte) {
	if nil == node.Sons {
		return
	}
	for i, son := range node.Sons {
		if bytes.Equal(son.PathValue[:], pathValue) {
			node.Sons = append(node.Sons[:i], node.Sons[i+1:]...)
		}
	}
}
