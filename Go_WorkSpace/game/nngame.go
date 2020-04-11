package main

import (
	"../model"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"log"
	"net/http"
	"strconv"
)

func main() {

	// 定义监听路由
	http.HandleFunc("/bets", NnBets)                     //下注
	http.HandleFunc("/cancelbets", NncancelBets)         //取消下注
	http.HandleFunc("/gameresult", NngameResult)         //判断结果
	http.HandleFunc("/nnGameRecord", NnGameRecord)       //牛牛游戏开奖结果
	http.HandleFunc("/userBetsRecord", NnuserBetsRecord) // 用户下注记录
	log.Println("server running...")
	log.Fatal(http.ListenAndServe("localhost:8888", nil))

}

/*
   用户下注
*/
func NnBets(w http.ResponseWriter, r *http.Request) {

	// 获取header头userid
	user_id, _ := strconv.Atoi(r.Header.Get("userid"))
	desk_id, _ := strconv.Atoi(r.PostFormValue("desk_id"))   //桌号
	boot_num, _ := strconv.Atoi(r.PostFormValue("boot_num")) //靴次
	pave_num, _ := strconv.Atoi(r.PostFormValue("pave_num")) //铺次

	//下注金额
	x1_double, _ := strconv.Atoi(r.PostFormValue("x1_double")) //闲1翻倍
	x1_equal, _ := strconv.Atoi(r.PostFormValue("x1_equal"))   //闲1平倍
	x2_double, _ := strconv.Atoi(r.PostFormValue("x2_double")) //闲2翻倍
	x2_equal, _ := strconv.Atoi(r.PostFormValue("x2_equal"))   //闲2平
	x3_double, _ := strconv.Atoi(r.PostFormValue("x3_double")) //闲3翻倍
	x3_equal, _ := strconv.Atoi(r.PostFormValue("x3_equal"))   //闲3平

	// 判断下注金额
	sum_double := x1_double + x2_equal + x3_equal //    go run翻倍下注金额
	sum_equal := x1_equal + x2_equal + x3_equal
	summoney := model.GetUserMoney(user_id)

	//判断用户金额
	if ((sum_double * 3) + sum_equal) > summoney {

		// 用户进行下注
		var Betsinfos model.Betsinfo
		Betsinfos.User_id = user_id
		Betsinfos.Desk_id = desk_id
		Betsinfos.Table_name = 1
		Betsinfos.Boot_num = boot_num
		Betsinfos.Pave_num = pave_num
		Betsinfos.X1_double = x1_double
		Betsinfos.X1_equal = x1_equal
		Betsinfos.X2_double = x2_double
		Betsinfos.X2_equal = x2_equal
		Betsinfos.X3_double = x3_double
		Betsinfos.X3_equal = x3_equal

		model.Betsno(Betsinfos, w)

	} else {

		model.JosnData(w, 200, "余额不足", nil)

		return
	}

}

/*
	取消下注
*/

func NncancelBets(w http.ResponseWriter, r *http.Request) {

	// 获取header头userid
	user_id, _ := strconv.Atoi(r.Header.Get("userid"))
	desk_id, _ := strconv.Atoi(r.PostFormValue("desk_id"))   //桌号
	boot_num, _ := strconv.Atoi(r.PostFormValue("boot_num")) //靴次
	pave_num, _ := strconv.Atoi(r.PostFormValue("pave_num")) //铺次

	// 取消下注记录
	err := model.CancelBetsRecord(w, user_id, desk_id, boot_num, pave_num)

	if err == nil {
		// 返回json
		model.JosnData(w, 200, "取消成功", nil)
	} else {
		// 返回json
		model.JosnData(w, 100, "取消失败", nil)
	}

	return

}

/*
	判断结果
*/
func NngameResult(w http.ResponseWriter, r *http.Request) {

	// 获取
	desk_id, _ := strconv.Atoi(r.PostFormValue("desk_id"))   //桌号
	boot_num, _ := strconv.Atoi(r.PostFormValue("boot_num")) //靴次
	pave_num, _ := strconv.Atoi(r.PostFormValue("pave_num")) //铺次

	var desk_info model.Desk_info // 存入结构体房间信息
	desk_info.Desk_id = desk_id
	desk_info.Boot_num = boot_num
	desk_info.Pave_num = pave_num

	resultnum := r.PostFormValue("resultnum") //游戏点数
	result := r.PostFormValue("result")       //游戏结果

	// 点数转换为map
	var resultnums map[string]string
	err := json.Unmarshal([]byte(resultnum), &resultnums)
	model.Check(err)

	// 结果转换为map
	var resultMap map[string]string
	errs := json.Unmarshal([]byte(result), &resultMap)
	model.Check(errs)

	//存储游戏结果
	go model.GameRecord(desk_info, resultMap, resultnums)

	// 游戏金额结算
	go model.Settlement(desk_info, resultMap, resultnums)

}

//游戏开奖记录返回数据格式
type results struct {
	Code int
	Msg  string
	Data map[int]map[string]string
}

// 获取游戏开奖记录

func NnGameRecord(w http.ResponseWriter, r *http.Request) {

	desk_id, _ := strconv.Atoi(r.PostFormValue("desk_id"))   //桌号
	boot_num, _ := strconv.Atoi(r.PostFormValue("boot_num")) //靴次
	pave_num, _ := strconv.Atoi(r.PostFormValue("pave_num")) //铺次

	GameRecordMap := model.GetGameRecord(desk_id, boot_num, pave_num)
	arr := &results{
		200,
		"牛牛游戏记录",
		GameRecordMap,
	}

	bejson, _ := json.Marshal(arr)
	fmt.Println("bejson", string(bejson), "\n")
	fmt.Fprintln(w, string(bejson))

}

// 用户投注记录

func NnuserBetsRecord(w http.ResponseWriter, r *http.Request) {

	// 获取header头userid

	user_id, _ := strconv.Atoi(r.Header.Get("userid"))

	// 根据日期订单表
	//starttime := r.Header.Get("starttime") //日期

	BetsRecord := model.GetUserBetsRecord(user_id)

	arr := &results{
		200,
		"下注记录",
		BetsRecord,
	}

	bejson, _ := json.Marshal(arr)
	fmt.Println("bejson", string(bejson), "\n")
	fmt.Fprintln(w, "返回结果:", string(bejson))

}
