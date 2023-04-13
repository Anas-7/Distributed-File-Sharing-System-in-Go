package surfstore

import (
	"crypto/sha256"
	"encoding/hex"
	"sort"
)

type ConsistentHashRing struct {
	ServerMap map[string]string
}

func (c ConsistentHashRing) GetResponsibleServer(blockId string) string {
	blockHash := (blockId)
	var hashes []string
	for hash := range c.ServerMap {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)
	for i := 0; i < len(hashes); i++ {
		if hashes[i] > blockHash {
			return c.ServerMap[hashes[i]]
		}
	}
	return c.ServerMap[hashes[0]]
}

func (c ConsistentHashRing) Hash(addr string) string {
	h := sha256.New()
	h.Write([]byte(addr))
	return hex.EncodeToString(h.Sum(nil))

}

func NewConsistentHashRing(serverAddrs []string) *ConsistentHashRing {
	c := ConsistentHashRing{}
	c.ServerMap = make(map[string]string)
	for _, addr := range serverAddrs {
		c.ServerMap[c.Hash("blockstore"+addr)] = addr
	}
	return &c
}
