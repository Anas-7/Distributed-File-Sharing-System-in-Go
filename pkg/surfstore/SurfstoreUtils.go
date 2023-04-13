package surfstore

import (
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"os"
	"strings"
)

// Implement the logic for a client syncing with the server here.
func ClientSync(client RPCClient) {
	//panic("todo")

	/*
		Check if index.db exists in the client's base directory. If it does not exist, create it

		Calculate the number of blocks using the blocksize variable in client

		Generate the fileMap for all files except index.db in the directory of the form map[string][]string
		The []string in the map refers to the hash values of the blocks in the file

		If the index.db file exists, then create the localFileMap from the index.db file using the LoadMetaFromMetaFile function

		Iterate over the fileMap and check if the file exists in the localFileMap

		If the file doesnt exist, then add the file into the localFileMap and set the version to 1

		If the file exists and the hashValues are different, then update the localFileMap with the new hashValues and increase the version by 1

		If the file exists and the hashValues are the same, then do nothing

		Now, connect to the remote metastore and access the remoteFileMap using the GetFileInfoMap function

		Iterate over the localFileMap and check if the file exists in the remoteFileMap

		If the file doesnt exist in remoteFileMap, then add the file into the remoteFileMap and set the version to whatver the localFileMap has

		If the file exists in the remoteFileMap, then check if the version of the localFileMap is greater than the version of the remoteFileMap. If it is, then update the remoteFileMap with the localFileMap's version and hashValues

		If the version in remoteFileMap is greater than the version in localFileMap, then update the localFileMap with the remoteFileMap's version and hashValues

		If the updateFile function returns a version of -1, then the file has been adjusted in the remoteFileMap by some other server, which means you have to update the localFileMap with the remoteFileMap's version and hashValues

		If file exists in remoteFileMap but not in localFileMap, then download the file from the remoteFileMap and add it to the localFileMap and storage using the GetBlock function
	*/
	base_dir := client.BaseDir
	block_size := client.BlockSize
	// Check if index.db exists in the client's base directory. If it does not exist, create it
	path := ConcatPath(base_dir, "index.db")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		//_, err := os.OpenFile(path, os.O_CREATE | os.O_RDWR, 0777)
		//fmt.Println("Creating Index.db")
		_, err := os.Create(path)
		if err != nil {
			log.Fatal("Error Creating Index.db")
		}
	}
	// Iterate over all files and generate a file map of the form map[string][]string where the key is the file name and the value is the hash values of the blocks in the file
	generatedFileMap := make(map[string][]string)
	// Iterate over all files in the base directory
	files, err := os.ReadDir(base_dir)
	if err != nil {
		log.Fatal("Error Reading Base Directory")
	}
	for _, file := range files {
		// Check if the file is index.db
		if file.Name() != "index.db" && strings.Contains(file.Name(), ",") == false && strings.Contains(file.Name(), "/") == false {
			// Get the hash values of the blocks in the file
			hashValues := GetFileBlockHashes(ConcatPath(base_dir, file.Name()), block_size)
			generatedFileMap[file.Name()] = hashValues
		}
	}
	////fmt.Println(generatedFileMap)
	// Now we have the map that we generated from the files in the base directory and their hash values array as the value

	// Generate a localFileMap from the index.db file
	localFileMap, err := LoadMetaFromMetaFile(base_dir)
	if err != nil {
		log.Fatal("Error Loading Meta From Meta File")
	}
	// //fmt.Print("Local File Map: ")
	// //fmt.Println(localFileMap)
	// Iterate over the generatedFileMap and check if the file exists in the localFileMap
	for fileName, hashValues := range generatedFileMap {
		// Check if the file exists in the localFileMap
		if _, ok := localFileMap[fileName]; ok {
			// Check if the hash values are the same
			if AreSlicesSame(localFileMap[fileName].BlockHashList, hashValues) == false {
				// Update the localFileMap with the new hashValues and increase the version by 1
				localFileMap[fileName].BlockHashList = hashValues
				localFileMap[fileName].Version += 1
			}
		} else {
			// Add the file into the localFileMap and set the version to 1
			localFileMap[fileName] = &FileMetaData{Filename: fileName, Version: 1, BlockHashList: hashValues}
		}
	}
	// Check if user deleted the files from the base directory
	for fileName, fileMetaData := range localFileMap {
		if _, ok := generatedFileMap[fileName]; !ok {
			if len(fileMetaData.BlockHashList) > 1 || fileMetaData.BlockHashList[0] != "0" {
				fileMetaData.Version += 1
				fileMetaData.BlockHashList = []string{"0"}
			}
		}
	}
	// Now our localFileMap is updated with the files in the base directory

	// Connect to the remote metastore and access the remoteFileMap using the GetFileInfoMap function
	var remoteFileMap map[string]*FileMetaData
	err = client.GetFileInfoMap(&remoteFileMap)
	if err != nil {
		log.Fatal("Error Getting File Info Map")
	}
	// Now I get the list of all the blockstore addresses
	var blockStoreAddrs []string
	err = client.GetBlockStoreAddrs(&blockStoreAddrs)
	// I can get the consistent hash ring from the blockstore addresses

	var consistentHashRing = NewConsistentHashRing(blockStoreAddrs)
	if err != nil {
		log.Fatal("Error Getting Block Store Address")
	}

	// Iterate over the localFileMap and check if the file exists in the remoteFileMap
	for fileName, fileMetaData := range localFileMap {
		// Check if the remoteFileMap needs to be updated
		if _, ok := remoteFileMap[fileName]; (!ok) || (ok && fileMetaData.Version > remoteFileMap[fileName].Version) {
			var latestVersion int32
			// No need to read data for a deleted file
			if len(fileMetaData.BlockHashList) == 1 && fileMetaData.BlockHashList[0] == "0" {
				err = client.UpdateFile(fileMetaData, &latestVersion)
				if err != nil {
					log.Fatal("Error Updating File")
				}
			} else {
				// Generate blocks of the local file and use PutBlock to put the blocks into the remote blockstore
				blocks, err := GetFileBlocks(ConcatPath(base_dir, fileName), client.BlockSize)
				if err != nil {
					log.Fatal("Error Getting File Blocks")
				}

				for _, block := range blocks {
					var succ bool
					//fmt.Println("Put Block Function: ")
					//fmt.Println(GetBlockHashString(block.BlockData))
					// First use the consistent hash ring to get the server address for the block
					serverAdd := consistentHashRing.GetResponsibleServer(GetBlockHashString(block.BlockData))
					err = client.PutBlock(&block, serverAdd, &succ)
					if err != nil {
						fmt.Println(err)
						log.Fatal("Error Putting Block")
					}
				}

				// Update the remoteFileMap with the localFileMap's version and hashValues
				err = client.UpdateFile(fileMetaData, &latestVersion)
				if err != nil {
					log.Fatal("Error Updating File")
				}
				fileMetaData.Version = latestVersion
			}
		}
	}
	// Now the remoteFileMap is updated with the files in the base directory

	// Now our localFileMap is updated with the files present in remoteFileMap
	for fileName, fileMetaData := range remoteFileMap {
		// Check if the file doesnt exist in the localFileMap
		if _, ok := localFileMap[fileName]; (!ok) || (ok && fileMetaData.Version > localFileMap[fileName].Version) || (ok && fileMetaData.Version == localFileMap[fileName].Version && AreSlicesSame(localFileMap[fileName].BlockHashList, fileMetaData.BlockHashList) == false) {
			// Add the file into the localFileMap and set the version to whatver the remoteFileMap has
			// Download the blocks from the remote blockstore and write that data into the local file
			// Update the map first
			localFileMap[fileName] = &FileMetaData{Filename: fileName, Version: fileMetaData.Version, BlockHashList: fileMetaData.BlockHashList}
			if len(fileMetaData.BlockHashList) == 1 && fileMetaData.BlockHashList[0] == "0" {
				if _, err := os.Stat(ConcatPath(base_dir, fileName)); err == nil {
					e := os.Remove(ConcatPath(base_dir, fileName))
					if e != nil {
						log.Fatal("Error During removal of file")
					}
				}
				continue
			}
			var receivedData = make([]byte, 0)
			for _, hashValue := range fileMetaData.BlockHashList {
				var block Block
				//fmt.Println(hashValue)
				// First use the consistent hash ring to get the server address for the block
				serverAddr := consistentHashRing.GetResponsibleServer(hashValue)
				err = client.GetBlock(hashValue, serverAddr, &block)
				if err != nil {
					log.Fatal("Error Getting Block")
				}
				receivedData = append(receivedData, block.BlockData...)
			}
			// Write the data into the file
			err = ioutil.WriteFile(ConcatPath(base_dir, fileName), receivedData, 0644)
			if err != nil {
				log.Fatal("Error Writing File")
			}
		}
	}
	//fmt.Print("New Local File Map: ")
	//fmt.Println(localFileMap)

	err = client.GetFileInfoMap(&remoteFileMap)
	if err != nil {
		log.Fatal("Error Getting File Info Map")
	}
	// //fmt.Print("New Remote File Map: ")
	// //fmt.Println(remoteFileMap)

	err = WriteMetaFile(localFileMap, base_dir)
	if err != nil {
		log.Fatal("Error Writing Meta File")
	}
}

func AreSlicesSame(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
func GetFileBlocks(filePath string, blockSize int) ([]Block, error) {
	// Calculate the number of blocks using the size of the file and the blocksize variable
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := fileInfo.Size()
	numBlocks := int(math.Ceil(float64(fileSize) / float64(blockSize)))
	// Generate the blocks of the file
	blocks := make([]Block, numBlocks)
	for i := 0; i < numBlocks; i++ {
		block_data := make([]byte, blockSize)
		n, _ := file.Read(block_data)
		block_data = block_data[:n]
		if n == 0 {
			break
		}
		var block Block
		block.BlockData = block_data
		block.BlockSize = int32(n)
		blocks[i] = block
	}
	return blocks, nil
}
func GetFileBlockHashes(filePath string, blockSize int) []string {
	// Calculate the number of blocks using the size of the file and the blocksize variable
	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal("Error Opening File")
	}
	defer file.Close()
	fileInfo, err := file.Stat()
	if err != nil {
		log.Fatal("Error Getting File Info")
	}
	fileSize := fileInfo.Size()
	numBlocks := int(math.Ceil(float64(fileSize) / float64(blockSize)))
	////fmt.Fprintln(os.Stdout, "Filesize, blocksize, blocks, %s, %s, %s", fileSize, blockSize, numBlocks)
	// Generate the hash values of the blocks in the file
	hashValues := make([]string, numBlocks)
	for i := 0; i < numBlocks; i++ {
		block := make([]byte, blockSize)
		n, err := file.Read(block)
		if n == 0 {
			break
		}
		block = block[:n]
		var blockBytes []byte = block
		if err != nil {
			log.Fatal("Error Reading File")
		}
		hashValues[i] = GetBlockHashString(blockBytes)
	}
	return hashValues
}
