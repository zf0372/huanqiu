package model

import (
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"net/http"
	"strconv"
	"time"
)

type Betsinfo struct {
	User_id    int
	X1_double  int
	X1_equal   int
	X2_double  int
	X2_equal   int
	X3_double  int
	X3_equal   int
	Desk_id    int
	Table_name int
	Boot_num   int
	Pave_num   int
}

func RoomXianHong(Desk_id int) map[string]string {

	var min_limit string
	var min_tie_limit string
	var max_limit string
	var mat_tie_limit string

	// 查看房间限红
	db := Mysql()
	db.QueryRow("select min_limit,min_tie_limit,max_limit,max_tie_limit from "+Desktable+" where id=?", Desk_id).Scan(&min_limit, &min_tie_limit, &max_limit, &mat_tie_limit)
	var min_limitmap map[string]string
	var min_tie_limitmap map[string]string
	var max_limitmap map[string]string
	var mat_tie_limitmap map[string]string
	json.Unmarshal([]byte(min_limit), &min_limitmap)         // 平倍最小限红
	json.Unmarshal([]byte(min_tie_limit), &min_tie_limitmap) //翻倍最小限红
	json.Unmarshal([]byte(max_limit), &max_limitmap)         //平倍最大限红
	json.Unmarshal([]byte(mat_tie_limit), &mat_tie_limitmap) //翻倍最大限红

	var XianHongMap map[string]string

	XianHongMap["min_tie_limitmap"] = min_tie_limitmap["c"] //翻倍最小限红
	XianHongMap["mat_tie_limitmap"] = mat_tie_limitmap["c"] //翻倍最大限红
	XianHongMap["min_limitmap"] = min_limitmap["c"]         // 平倍最小限红
	XianHongMap["max_limitmap"] = max_limitmap["c"]         // //平倍最大限红

	return min_limitmap

}

// 用户下注
func Betsno(B Betsinfo, w http.ResponseWriter) {

	// 筛选下注金额
	betsMap := make(map[string]int)
	var score int
	if B.X1_double > 0 {
		betsMap["x1_double"] = B.X1_double
	}

	if B.X1_equal > 0 {
		betsMap["x1_equal"] = B.X1_equal
	}
	if B.X2_double > 0 {
		betsMap["x2_double"] = B.X2_double
	}
	if B.X2_equal > 0 {
		betsMap["x2_equal"] = B.X2_equal
	}
	if B.X3_double > 0 {
		betsMap["x3_double"] = B.X3_double
	}
	if B.X3_equal > 0 {
		betsMap["x3_equal"] = B.X3_equal
	}

	// 下注记录转json 存入数据库
	bejson, _ := json.Marshal(betsMap)

	// 查看房间限红
	XianHongMap := RoomXianHong(B.Desk_id)

	// 获取用户下注金额
	for k, v := range betsMap {
		// 判断限红
		if k == "x1_double" || k == "x2_double" || k == "x3_double" {
			mintielimit, _ := strconv.Atoi(XianHongMap["min_tie_limitmap"])
			mattielimitmap, _ := strconv.Atoi(XianHongMap["mat_tie_limitmap"])
			if v < mintielimit || v > mattielimitmap {
				JosnData(w, 200, "限红条件不满足", nil)
				return
			}
			score += v * 3
		} else {
			minlimitmap, _ := strconv.Atoi(XianHongMap["min_limitmap"])
			maxlimitmap, _ := strconv.Atoi(XianHongMap["max_limitmap"])
			if v < minlimitmap || v > maxlimitmap {
				JosnData(w, 200, "限红条件不满足", nil)
				return
			}
			score += v
		}
	}

	db := Mysql()
	//开启事务
	tx, err := db.Begin()
	Check(err)

	// 添加下注订单信息
	orderinfo, _ := db.Prepare("insert into " + Ordertable + "(user_id,order_sn,desk_id,table_name,boot_num,pave_num,bet_money,status,creatime)values(?,?,?,?,?,?,?,?,?)")
	order_sn := GetOrderSn("N") //订单号
	now := time.Now().Unix()    //时间戳
	_, orderinfoerr := orderinfo.Exec(B.User_id, order_sn, B.Desk_id, B.Table_name, B.Boot_num, B.Pave_num, bejson, 0, now)
	Check(orderinfoerr)

	//查询并扫描用户金额 获取到扣款前的金额
	var Balance int
	_ = db.QueryRow("select balance from hq_user_account where user_id=?", B.User_id).Scan(&Balance)

	// 插入资金明细表
	billflow, _ := db.Prepare("insert  into " + Billflowtable + "(user_id,score,bet_before,bet_after,order_sn,status,remark,creatime)values(?,?,?,?,?,?)")
	_, billflowerr := billflow.Exec(B.User_id, -score, Balance, Balance-score, order_sn, 2, "牛牛下注", now)

	// 更新用户账变信息
	_, accounterr := db.Exec("update hq_user_account set balance=? where user_id=?", Balance-score, B.User_id)

	if orderinfoerr == nil && billflowerr == nil && accounterr == nil {
		tx.Commit() //执行成功 提交事务
	} else {
		tx.Rollback() //执行失败 事务回滚
	}

}

// 取消下注记录
func CancelBetsRecord(w http.ResponseWriter, user_id int, desk_id int, boot_num int, pave_num int) error {

	db := Mysql()
	// 开启事务
	tx, err := db.Begin()
	Check(err)

	// 取消下注记录
	_, errs := db.Exec("update "+Ordertable+" set status=? where user_id=? and desk_id=? and boot_num=? and pave_num=?", 2, user_id, desk_id, boot_num, pave_num)
	Check(errs)

	//查询用户下注订单
	rows, sqlerr := db.Query("SELECT order_sn FROM "+Ordertable+" WHERE user_id=? and desk_id=? and boot_num=? and pave_num=?", user_id, desk_id, boot_num, pave_num)
	Check(sqlerr)
	// 返回用户金额
	var order_sn string
	for rows.Next() {

		//查询并扫描用户金额 获取到扣款前的金额
		var Balance = 0
		_ = db.QueryRow("select balance from hq_user_account where user_id=?", user_id).Scan(&Balance)

		rows.Scan(&order_sn)
		var score int = 0
		sqler := db.QueryRow("SELECT score FROM "+Billflowtable+" WHERE order_sn=?", order_sn).Scan(&score)
		Check(sqler)

		// 插入流水表
		billflow, billflowerr := db.Prepare("insert  into " + Billflowtable + "(user_id,score,bet_before,bet_after,order_sn,status,remark,creatime)values(?,?,?,?,?,?)")
		Check(billflowerr)
		now := time.Now().Unix()
		_, er := billflow.Exec(user_id, score, Balance, Balance+score, order_sn, 2, "牛牛取消下注", now)
		Check(er)

		// 更新用户账变信息
		_, accounterr := db.Exec("update hq_user_account set balance=? where user_id=?", Balance+score, user_id)
		Check(accounterr)

		if er == nil && accounterr == nil && errs == nil && sqlerr == nil {
			tx.Commit() //执行成功 提交事务

		} else {
			tx.Rollback() //执行失败 事务回滚

		}

	}

	return errs

}

// 更新下注表输赢
func UpdOrderMoney(get_money int, user_id int, desk_info Desk_info) {

	db := Mysql()
	// 更新下注表输赢
	_, err := db.Exec("update "+Ordertable+" set get_money=?+get_money,status=? where user_id=? and desk_id=? and boot_num=? and pave_num=?", get_money, 2, user_id, desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num)
	Check(err)
}

/*
   存储游戏结果
*/
func GameRecord(desk_info Desk_info, resultMap map[string]string, resultnums map[string]string) {

	db := Mysql()

	// 插入结果表
	gamerecord, err := db.Prepare("insert into " + Gamerecordtable + "(bankernum,idle_one_num,idle_two_num,idle_three_num,desk_id,boot_num,pave_num,status,creatime)values(?,?,?,?,?,?,?,?,?)")
	Check(err)
	now := time.Now().Unix()
	_, er := gamerecord.Exec(resultnums["bankernum"], resultnums["x1num"], resultnums["x2num"], resultnums["x3num"], desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num, 1, now)
	Check(er)

	for _, v := range resultMap {

		if v == "x1win" {
			_, err := db.Exec("update "+Gamerecordtable+" set idle_one_result=? where  desk_id=? and boot_num=? and pave_num=?", "win", desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num)
			Check(err)
		} else if v == "x2win" {
			_, err := db.Exec("update "+Gamerecordtable+" set idle_two_result=? where  desk_id=? and boot_num=? and pave_num=?", "win", desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num)
			Check(err)
		} else if v == "x3win" {
			_, err := db.Exec("update "+Gamerecordtable+" set idle_three_result=? where  desk_id=? and boot_num=? and pave_num=?", "win", desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num)
			Check(err)
		}

	}

}

//用户结算
func Settlement(desk_info Desk_info, resultMap map[string]string, resultnums map[string]string) {
	// 查询下注用户
	db := Mysql()
	rows, sqlerr := db.Query("SELECT user_id,order_sn, bet_money FROM "+Ordertable+" WHERE status=? and desk_id=? and boot_num=? and pave_num=?", 0, desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num)
	Check(sqlerr)

	for rows.Next() {
		var user_id int
		var bet_money string
		var order_sn string
		rows.Scan(&user_id, &order_sn, &bet_money)

		fmt.Println("user_id：", user_id)

		// 结果转换为map
		var bet_moneymap map[string]int
		err := json.Unmarshal([]byte(bet_money), &bet_moneymap)
		Check(err)

		// 遍历结果
		for _, value := range resultMap {
			//遍历下注结果
			for k, v := range bet_moneymap {
				//获取倍数
				zbei := bei(resultnums["bankernum"])
				x1bei := bei(resultnums["x1num"])
				x2bei := bei(resultnums["x2num"])
				x3bei := bei(resultnums["x3num"])
				score := v
				if value == "x1win" {
					if k == "x1_double" { // 翻倍
						addUserMoney(user_id, score*Nnbei+(score*x1bei*Nnfee/100), order_sn, "牛牛盈利") //结算
						UpdOrderMoney(score*x1bei*Nnfee/100, user_id, desk_info)                     //更新下注记录 291 +

					} else if k == "x1_equal" { //平倍
						addUserMoney(user_id, score+(score*Nnfee/100), order_sn, "牛牛盈利")
						UpdOrderMoney(score*Nnfee/100, user_id, desk_info) //更新下注记录
					}

				} else if value == "x2win" {
					if k == "x2_double" { // 翻倍
						addUserMoney(user_id, score*Nnbei+(score*x2bei*Nnfee/100), order_sn, "牛牛盈利") //结算
						UpdOrderMoney(score*x2bei*Nnfee/100, user_id, desk_info)                     //更新下注记录
					} else if k == "x2_equal" { //平倍

						addUserMoney(user_id, score+(score*Nnfee/100), order_sn, "牛牛盈利")
						UpdOrderMoney(score*Nnfee/100, user_id, desk_info) //更新下注记录
					}
				} else if value == "x3win" {
					if k == "x3_double" { // 翻倍
						addUserMoney(user_id, score*Nnbei+(score*x3bei*Nnfee/100), order_sn, "牛牛盈利") //结算
						UpdOrderMoney(score*x3bei*Nnfee/100, user_id, desk_info)                     //更新下注记录
					} else if k == "x3_equal" { //平倍
						addUserMoney(user_id, score+(score*Nnfee/100), order_sn, "牛牛盈利")
						UpdOrderMoney(score*Nnfee/100, user_id, desk_info) //更新下注记录
					}
				} else {
					//无人中 庄家赢
					if score*3 > score*zbei {
						addUserMoney(user_id, score*3-score*zbei, order_sn, "牛牛退款")
						UpdOrderMoney(-(score * zbei), user_id, desk_info)
					}
				}
			}
		}
	}

	//所有用户结算完成  通知用户结算信息
	informSettlementInfo(desk_info)
}

// 用户加金额
func addUserMoney(user_id int, score int, order_sn string, remark string) {
	// 创建连接
	db := Mysql()
	// 开启事务
	tx, err := db.Begin()
	Check(err)

	//查询并扫描用户金额 获取到扣款前的金额
	var Balance int
	_ = db.QueryRow("select balance from hq_user_account where user_id=?", user_id).Scan(&Balance)

	// 插入流水表
	billflow, _ := db.Prepare("insert  into " + Billflowtable + "(user_id,score,bet_before,bet_after,order_sn,status,remark,creatime)values(?,?,?,?,?,?)")
	now := time.Now().Unix()
	_, billflowerr := billflow.Exec(user_id, score, Balance, Balance+score, order_sn, 2, remark, now)

	// 更新用户账变信息
	_, accounterr := db.Exec("update hq_user_account set balance=? where user_id=?", Balance+score, user_id)

	if billflowerr == nil && accounterr == nil {
		tx.Rollback() //提交事务
	} else {
		tx.Commit() //回滚
	}

}

//通知用户结算信息
func informSettlementInfo(desk_info Desk_info) {
	db := Mysql()
	Info, err := db.Query("select  DISTINCT user_id,sum(get_money) from "+Ordertable+" where desk_id=? and boot_num=? and pave_num=?", desk_info.Desk_id, desk_info.Boot_num, desk_info.Pave_num)
	Check(err)
	for Info.Next() {
		var user_id int
		var sumGetMoney int
		Info.Scan(&user_id, &sumGetMoney)
		// 通知信息
		SettlementInfo(user_id, sumGetMoney)

	}

}

// 通知用户结算
func SettlementInfo(user_id int, sumGetMoney int) {
	fmt.Println("通知用户 :", user_id, "输赢金额:", sumGetMoney)
}

// 根据点数获取倍数
func bei(num string) int {

	switch num {
	case "炸弹":
		return 3
	case "5公":
		return 3
	case "牛牛":
		return 3
	case "牛9":
		return 2
	case "牛8":
		return 2
	case "牛7":
		return 2
	default:
		return 1
	}
}

// 获取游戏开奖记录

func GetGameRecord(desk_id int, boot_num int, pave_num int) map[int]map[string]string {

	db := Mysql()
	gamerecord, err := db.Query("select id,bankernum,idle_one_num,idle_one_result,idle_two_num,idle_two_result,idle_three_num,idle_three_result from "+Gamerecordtable+" where desk_id=? and boot_num=? and pave_num=?", desk_id, boot_num, pave_num)
	Check(err)

	var bankernum string
	var idle_one_num string
	var idle_one_result string
	var idle_two_num string
	var idle_two_result string
	var idle_three_num string
	var idle_three_result string
	var id int

	//var mainMapA map[int]string
	mainMapA := map[int]map[string]string{}
	for gamerecord.Next() {

		GameRecordMap := map[string]string{}

		gamerecord.Scan(&id, &bankernum, &idle_one_num, &idle_one_result, &idle_two_num, &idle_two_result, &idle_three_num, &idle_three_result)

		GameRecordMap["bankernum"] = bankernum
		GameRecordMap["idle_one_num"] = idle_one_num
		GameRecordMap["idle_one_result"] = idle_one_result
		GameRecordMap["idle_two_num"] = idle_two_num
		GameRecordMap["idle_two_result"] = idle_two_result
		GameRecordMap["idle_three_num"] = idle_three_num
		GameRecordMap["idle_three_result"] = idle_three_result

		mainMapA[id] = GameRecordMap

	}

	return mainMapA

}

func GetUserBetsRecord(user_id int) map[int]map[string]string {

	db := Mysql()
	gamerecord, err := db.Query("select id,user_id,order_sn,desk_id,desk_name,boot_num,pave_num,bet_money,get_money,status,creatime from "+Ordertable+" where user_id=?", user_id)
	Check(err)

	var id int
	var userid string
	var order_sn string
	var desk_id string
	var desk_name string
	var boot_num string
	var pave_num string
	var bet_money string
	var get_money string
	var status string
	var creatime string

	//var mainMapA map[int]string
	mainMapA := map[int]map[string]string{}
	for gamerecord.Next() {

		BetsRecord := map[string]string{}

		gamerecord.Scan(&id, &userid, &order_sn, &desk_id, &desk_name, &boot_num, &pave_num, &bet_money, &get_money, &status, &creatime)

		BetsRecord["user_id"] = userid
		BetsRecord["order_sn"] = order_sn
		BetsRecord["desk_id"] = desk_id
		BetsRecord["desk_name"] = desk_name
		BetsRecord["boot_num"] = boot_num
		BetsRecord["pave_num"] = pave_num
		BetsRecord["bet_money"] = bet_money
		BetsRecord["get_money"] = get_money
		BetsRecord["status"] = status
		BetsRecord["creatime"] = creatime

		mainMapA[id] = BetsRecord

	}

	return mainMapA
}
