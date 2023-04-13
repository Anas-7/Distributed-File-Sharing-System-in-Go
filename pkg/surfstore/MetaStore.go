package surfstore

import (
	context "context"

	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

type MetaStore struct {
	FileMetaMap        map[string]*FileMetaData
	BlockStoreAddrs    []string
	ConsistentHashRing *ConsistentHashRing
	UnimplementedMetaStoreServer
}

func (m *MetaStore) GetFileInfoMap(ctx context.Context, _ *emptypb.Empty) (*FileInfoMap, error) {
	//panic("todo")
	return &FileInfoMap{FileInfoMap: m.FileMetaMap}, nil
}

func (m *MetaStore) UpdateFile(ctx context.Context, fileMetaData *FileMetaData) (*Version, error) {
	// panic("todo")
	filename := fileMetaData.Filename
	// If the file is not in the map, add it
	if _, ok := m.FileMetaMap[filename]; !ok {
		m.FileMetaMap[filename] = fileMetaData
		return &Version{Version: m.FileMetaMap[filename].Version}, nil
	} else {
		if fileMetaData.Version == m.FileMetaMap[filename].Version+1 {
			m.FileMetaMap[filename] = fileMetaData
			return &Version{Version: fileMetaData.Version}, nil
		}
		return &Version{Version: -1}, nil
	}
}

func (m *MetaStore) GetBlockStoreMap(ctx context.Context, blockHashesIn *BlockHashes) (*BlockStoreMap, error) {
	//panic("todo")
	blockStoreMap := make(map[string]*BlockHashes)
	for _, hash := range blockHashesIn.Hashes {
		if _, ok := blockStoreMap[m.ConsistentHashRing.GetResponsibleServer(hash)]; !ok {
			blockStoreMap[m.ConsistentHashRing.GetResponsibleServer(hash)] = &BlockHashes{Hashes: []string{hash}}
		} else {
			blockStoreMap[m.ConsistentHashRing.GetResponsibleServer(hash)].Hashes = append(blockStoreMap[m.ConsistentHashRing.GetResponsibleServer(hash)].Hashes, hash)
		}
	}
	return &BlockStoreMap{BlockStoreMap: blockStoreMap}, nil
}

// func (m *MetaStore) GetBlockStoreMap(ctx context.Context) (*BlockStoreMap, error) {
// 	//panic("todo")
// 	blockStoreMap := make(map[string]string)
// 	blockStoreMap[m.BlockStoreAddr] = m.BlockStoreAddr
// 	return &BlockStoreMap{BlockStoreMap: blockStoreMap}, nil
// }

func (m *MetaStore) GetBlockStoreAddrs(ctx context.Context, _ *emptypb.Empty) (*BlockStoreAddrs, error) {
	// panic("todo")
	return &BlockStoreAddrs{BlockStoreAddrs: m.BlockStoreAddrs}, nil
}

// func (m *MetaStore) GetBlockStoreAddrs(ctx context.Context, _ *emptypb.Empty) (*BlockStoreAddrs, error) {
// 	//panic("todo")
// 	return &BlockStoreAddr{Addr: m.BlockStoreAddr}, nil
// }

// This line guarantees all method for MetaStore are implemented
var _ MetaStoreInterface = new(MetaStore)

func NewMetaStore(blockStoreAddrs []string) *MetaStore {
	return &MetaStore{
		FileMetaMap:        map[string]*FileMetaData{},
		BlockStoreAddrs:    blockStoreAddrs,
		ConsistentHashRing: NewConsistentHashRing(blockStoreAddrs),
	}
}
