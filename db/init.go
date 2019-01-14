package db

var EktDB IKVDatabase

func InitEKTDB(filePath string) {
	EktDB = NewComposedKVDatabase(filePath)
}

func GetDBInst() IKVDatabase {
	if EktDB == nil {
		InitEKTDB("~/.cache/EKT/db")
	}
	return EktDB
}
