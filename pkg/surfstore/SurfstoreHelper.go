package surfstore

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

/* Hash Related */
func GetBlockHashBytes(blockData []byte) []byte {
	h := sha256.New()
	h.Write(blockData)
	return h.Sum(nil)
}

func GetBlockHashString(blockData []byte) string {
	blockHash := GetBlockHashBytes(blockData)
	return hex.EncodeToString(blockHash)
}

/* File Path Related */
func ConcatPath(baseDir, fileDir string) string {
	return baseDir + "/" + fileDir
}

/*
	Writing Local Metadata File Related
*/

const createTable string = `create table if not exists indexes (
		fileName TEXT, 
		version INT,
		hashIndex INT,
		hashValue TEXT
	);`

const insertTuple string = `insert into indexes(fileName, version, hashIndex, hashValue) values (?, ?, ?, ?);`

// WriteMetaFile writes the file meta map back to local metadata file index.db
func WriteMetaFile(fileMetas map[string]*FileMetaData, baseDir string) error {
	// remove index.db file if it exists
	outputMetaPath := ConcatPath(baseDir, DEFAULT_META_FILENAME)
	if _, err := os.Stat(outputMetaPath); err == nil {
		e := os.Remove(outputMetaPath)
		if e != nil {
			log.Fatal("Error During Meta Write Back")
		}
	}
	db, err := sql.Open("sqlite3", outputMetaPath)
	if err != nil {
		log.Fatal("Error During Meta Write Back")
	}
	statement, err := db.Prepare(createTable)
	if err != nil {
		log.Fatal("Error During Meta Write Back")
	}
	statement.Exec()
	//panic("todo")

	// Iterate over the file meta map
	for fileName, fileMeta := range fileMetas {
		// Iterate over the block hash list
		for hashIndex, blockHash := range fileMeta.BlockHashList {
			// Insert the tuple into the database
			statement, err := db.Prepare(insertTuple)
			if err != nil {
				return err
			}
			statement.Exec(fileName, fileMeta.Version, hashIndex, blockHash)
		}
	}
	db.Close()
	return nil
}

/*
Reading Local Metadata File Related
*/
const getDistinctFileName string = `select distinct fileName from indexes;`

const getTuplesByFileName string = `select * from indexes where fileName = ?;`

// LoadMetaFromMetaFile loads the local metadata file into a file meta map.
// The key is the file's name and the value is the file's metadata.
// You can use this function to load the index.db file in this project.
func LoadMetaFromMetaFile(baseDir string) (fileMetaMap map[string]*FileMetaData, e error) {
	metaFilePath, _ := filepath.Abs(ConcatPath(baseDir, DEFAULT_META_FILENAME))
	fileMetaMap = make(map[string]*FileMetaData)
	metaFileStats, e := os.Stat(metaFilePath)
	if e != nil || metaFileStats.IsDir() {
		return fileMetaMap, nil
	}
	db, err := sql.Open("sqlite3", metaFilePath)
	if err != nil {
		log.Fatal("Error When Opening Meta")
	}
	//panic("todo")
	// Declare the map to be returned
	var returnMap = make(map[string]*FileMetaData)
	//fmt.Println(db)
	// Select all distinct file names from the database
	fileNameRows, err := db.Query(getDistinctFileName)
	if fileNameRows == nil {
		return returnMap, nil
	}
	//fmt.Println(fileNameRows)
	for fileNameRows.Next() {
		// Generate a map for each file
		var fileName string
		fileNameRows.Scan(&fileName)
		// Select all rows where fileName = fileName
		rows, err := db.Query(getTuplesByFileName, fileName)
		if err != nil {
			log.Fatal("Error When Opening Meta")
		}
		// Declare the array for storing hashValue
		var hashValueArray []string
		// Declare the version
		var version int32
		// Use a flag to set the version once
		var onceSet bool = false
		// Iterate over the rows
		for rows.Next() {
			var d0 string
			var d1 int32
			var d2 int
			var hashValue string
			rows.Scan(&d0, &d1, &d2, &hashValue)
			//fmt.Fprintln(os.Stdout, "Rows: %s, %d, %d, %s", d0, d1, d2, hashValue)
			if !onceSet {
				version = d1
				onceSet = true
			}
			hashValueArray = append(hashValueArray, hashValue)
		}
		returnMap[fileName] = &FileMetaData{Filename: fileName, Version: version, BlockHashList: hashValueArray}
	}
	return returnMap, nil
}

/*
	Debugging Related
*/

// PrintMetaMap prints the contents of the metadata map.
// You might find this function useful for debugging.
func PrintMetaMap(metaMap map[string]*FileMetaData) {

	fmt.Println("--------BEGIN PRINT MAP--------")

	for _, filemeta := range metaMap {
		fmt.Println("\t", filemeta.Filename, filemeta.Version)
		for _, blockHash := range filemeta.BlockHashList {
			fmt.Println("\t", blockHash)
		}
	}

	fmt.Println("---------END PRINT MAP--------")

}
