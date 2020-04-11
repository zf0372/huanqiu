package model

import (
	"database/sql"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
)

// 定义数据库链接信息
const dataSourceName string = "huanqiu:huanqiu@tcp(114.67.97.70:3306)/huanqiu?charset=utf8"

/*
链接数据库
*/
func Mysql() *sql.DB {

	db, err := sql.Open("mysql", dataSourceName)
	if err != nil {
		fmt.Println("链接错误！:")
	}
	return db
}
