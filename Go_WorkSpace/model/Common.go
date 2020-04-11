package model

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

const Ordertable = "hq_order_20200401" // order表

const Billflowtable = "hq_user_billflow_20200402"
const Gamerecordtable = "hq_nngame_record_20200403"

const Desktable = "hq_desk" // order表

const Nnbei = 3  // 牛牛费率
const Nnfee = 97 // 牛牛费率

//定义返回数据格式
type result struct {
	Code int
	Msg  string
	Data map[string]interface{}
}

//返回数据json
func JosnData(w http.ResponseWriter, code int, msg string, data map[string]interface{}) {
	arr := &result{
		code,
		msg,
		data,
	}
	jsonReturn, json_err := json.Marshal(arr) //json化结果集
	if json_err != nil {
		fmt.Println("encoding faild")
	}

	fmt.Fprintln(w, "返回结果:", string(jsonReturn))

}

func GetUserMoney(user_id int) int {
	// 链接数据库
	db := Mysql()
	var summoney int
	rows, err := db.Query("SELECT balance FROM hq_user_account WHERE user_id=?", user_id)
	Check(err)
	for rows.Next() {
		err = rows.Scan(&summoney)
	}
	return summoney
}

func Check(err error) { //因为要多次检查错误，所以干脆自己建立一个函数。
	if err != nil {
		fmt.Println("操作失败!", err)
	}

}

//无重复订单号
func GetOrderSn(game string) string {
	timeLayout := "20060102150405"
	t := time.Now()
	pre := t.Format(timeLayout)
	rand.Seed(time.Now().UnixNano())
	num := rand.Intn(99999999-11111111+11111111) + 11111111 //[11111111,99999999]
	suff := strconv.FormatInt(int64(num), 10)
	order_sn := game + pre + suff
	return order_sn
}

//根据订单号获取表
func GetTbaleSuff(order_sn string) string {
	suff := []rune(order_sn)
	return string(suff[1:9])
}

//定义房间信息数据格式
type Desk_info struct {
	Desk_id  int
	Boot_num int
	Pave_num int
}
