package common

var UsingSQLite = false
var UsingPostgreSQL = false

var SQLitePath = "E:\\proejct\\one-api-main\\one-api.db"
var SQLiteBusyTimeout = GetOrDefault("SQLITE_BUSY_TIMEOUT", 3000)
