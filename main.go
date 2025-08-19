package main

import (
	"bufio"         // 引入 bufio 套件用於帶緩衝的讀寫
	"bytes"         // 引入 bytes 套件用於處理字節緩衝區
	"encoding/json" // 引入 json 套件用於解析 JSON
	"fmt"
	"io"
	"net/http"
	"os"      // 引入 os 套件用於處理命令行參數和環境變數
	"strconv" // 引入 strconv 套件用於字串轉換
	"strings" // 引入 strings 套件用於字串操作
	"time"
)

// 全局變數用於儲存定時器的相關信息
var (
	timerActive      bool = false
	timerEndTime     time.Time
	timerDuration    time.Duration
	timerDescription string
	lastStatusTime   time.Time // 上次更新狀態的時間，用於精確計算剩餘時間
)

// loadEnvFile 函數用於從 .env 檔案中讀取環境變數
func loadEnvFile(filename string) (map[string]string, error) {
	envMap := make(map[string]string)
	file, err := os.Open(filename)
	if err != nil {
		if os.IsNotExist(err) {
			return envMap, nil // 檔案不存在不是錯誤，只是沒有環境變數
		}
		return nil, fmt.Errorf("無法打開環境變數檔案 %s: %w", filename, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue // 跳過空行或註釋行
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			envMap[key] = value
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("讀取環境變數檔案時出錯 %s: %w", filename, err)
	}
	return envMap, nil
}

// DeviceInfo 結構體用於解析 GET 響應中的 data 部分，以及構建 POST 請求的 payload
type DeviceInfo struct {
	ID                string      `json:"id"`
	ManufactorID      string      `json:"manufactorId"`
	ModelID           string      `json:"modelId"`
	GatewayID         string      `json:"gatewayId"`
	PortID            string      `json:"portId"`
	CampusID          string      `json:"campusId"`
	BuildingID        string      `json:"buildingId"`
	FloorID           string      `json:"floorId"`
	RoomID            string      `json:"roomId"`
	DeviceType        int         `json:"deviceType"`
	DeviceNo          string      `json:"deviceNo"`
	DeviceIdx         int         `json:"deviceIdx"`
	Status            int         `json:"status"`
	StatusReason      string      `json:"statusReason"`
	Creator           string      `json:"creator"`
	CreateDate        string      `json:"createDate"`
	CampusTitle       string      `json:"campusTitle"`
	BuildingTitle     string      `json:"buildingTitle"`
	FloorTitle        string      `json:"floorTitle"`
	RoomNo            string      `json:"roomNo"`
	ManufactorTitle   string      `json:"manufactorTitle"`
	ModelTitle        string      `json:"modelTitle"`
	GatewayNo         string      `json:"gatewayNo"`
	SNCode            string      `json:"snCode"`
	PortIdx           int         `json:"portIdx"`
	DeviceFan         *DeviceFan  `json:"deviceFan"`   // 使用指針，因為可能為 null
	DeviceMeter       interface{} `json:"deviceMeter"` // 可以是 null
	DeviceWater       interface{} `json:"deviceWater"` // 可以是 null
	IsInstallFinish   int         `json:"isInstallFinish"`
	Position          interface{} `json:"position"` // 可以是 null
	CommandKey        string      `json:"commandKey"`
	LastCommunication string      `json:"lastCommunication"`
	ProcessResult     interface{} `json:"processResult"` // 可以是 null
	ProcessMsg        interface{} `json:"processMsg"`    // 可以是 null
	CollectorNo       string      `json:"collectorNo"`
	Forbidden         int         `json:"forbidden"`
	Balance           float64     `json:"balance"`
	NickNames         string      `json:"nickNames"`
	UpdateDate        string      `json:"updateDate"`
	MeterUsePower     interface{} `json:"meterUsePower"`         // 可以是 null
	DeviceGroup       interface{} `json:"deviceGroup"`           // 可以是 null
	StudentName       string      `json:"studentName,omitempty"` // AirOpen.json 中有，GetdeviceNo.json 中沒有
}

// DeviceFan 結構體用於解析 DeviceInfo 中的 deviceFan 部分
type DeviceFan struct {
	ID             string  `json:"id"`
	DeviceID       string  `json:"deviceId"`
	FanType        int     `json:"fanType"`
	Password       string  `json:"password"`
	FanStatus      int     `json:"fanStatus"` // 0 為關閉，1 為開啟
	LockStatus     int     `json:"lockStatus"`
	TempSetting    float64 `json:"tempSetting"`
	FanModel       int     `json:"fanModel"`
	WindSpeed      int     `json:"windSpeed"`
	MaxTemp        float64 `json:"maxTemp"`
	MinTemp        float64 `json:"minTemp"`
	CompensateTemp float64 `json:"compensateTemp"`
	CompensateFalg int     `json:"compensateFalg"`
	ReturnTemp     float64 `json:"returnTemp"`
	CurrentTemp    float64 `json:"currentTemp"`
	FanStatusOld   int     `json:"fanStatusOld"`
}

// GetAPIResponse 結構體用於解析 GET 設備信息請求的整個 JSON 響應
type GetAPIResponse struct {
	Code int        `json:"code"`
	Msg  string     `json:"msg"`
	Data DeviceInfo `json:"data"`
}

// OperateAPIResponse 結構體用於解析空調開關請求的 JSON 響應
type OperateAPIResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data struct {
		MsgID    string `json:"msgId"`
		DeviceNo string `json:"deviceNo"`
	} `json:"data"`
}

// getDeviceInfo 函數用於獲取設備信息
func getDeviceInfo(deviceNo, token string) (*DeviceInfo, int, error) {
	url := fmt.Sprintf("https://es.sdtbu.edu.cn/hatch-api/api/sdgongshang/device/getDeviceByNo?deviceNo=%s", deviceNo)

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, 0, fmt.Errorf("創建 GET 請求失敗: %w", err)
	}

	req.Header.Set("Host", "es.sdtbu.edu.cn")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 NetType/WIFI MicroMessenger/7.0.20.1781(0x6700143B) WindowsWechat(0x63090c33) XWEB/13639 Flue")
	req.Header.Set("Token", token)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", "https://es.sdtbu.edu.cn/?code=081yCw2w33V3553Rxc4w3olDyB0yCw2G&state=wx")
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return nil, 0, fmt.Errorf("發送 GET 請求失敗: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, fmt.Errorf("讀取 GET 響應體失敗: %w", err)
	}

	var getResponse GetAPIResponse
	err = json.Unmarshal(body, &getResponse)
	if err != nil {
		return nil, resp.StatusCode, fmt.Errorf("解析 GET JSON 失敗: %w, 原始響應體:\n%s", err, string(body))
	}

	if getResponse.Code != 0 {
		return nil, resp.StatusCode, fmt.Errorf("獲取設備信息 API 返回錯誤代碼: %d, 訊息: %s", getResponse.Code, getResponse.Msg)
	}

	return &getResponse.Data, resp.StatusCode, nil
}

// operateDevice 函數用於空調開關操作
// 接收 studentName 參數
func operateDevice(deviceInfo *DeviceInfo, token string, action string, studentName string) (int, string, string, error) {
	url := "https://es.sdtbu.edu.cn/hatch-api/api/sdgongshang/device/operateDevice"

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// 根據操作類型設置 commandKey 和 fanStatus
	if action == "start" || action == "acon" { // 添加 /acon 命令
		deviceInfo.CommandKey = "AirOpen"
		if deviceInfo.DeviceFan != nil {
			deviceInfo.DeviceFan.FanStatus = 1 // 開啟
		}
		deviceInfo.StudentName = studentName // 使用傳入的 studentName 參數
	} else if action == "stop" || action == "acoff" { // 添加 /acoff 命令
		deviceInfo.CommandKey = "AirClose"
		if deviceInfo.DeviceFan != nil {
			deviceInfo.DeviceFan.FanStatus = 0 // 關閉
		}
		deviceInfo.StudentName = studentName // 關閉時也設置 StudentName
	} else {
		return 0, "", "", fmt.Errorf("無效的操作：%s，請使用 -start/-acon 或 -stop/-acoff", action)
	}

	// 將更新後的 deviceInfo 序列化為 JSON
	payloadBytes, err := json.Marshal(deviceInfo)
	if err != nil {
		return 0, "", "", fmt.Errorf("序列化請求 payload 失敗: %w", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return 0, "", "", fmt.Errorf("創建 POST 請求失敗: %w", err)
	}

	// 設定請求頭
	req.Header.Set("Host", "es.sdtbu.edu.cn")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/126.0.0.0 Safari/537.36 NetType/WIFI MicroMessenger/7.0.20.1781(0x6700143B) WindowsWechat(0x63090c33) XWEB/13639 Flue")
	req.Header.Set("Token", token)
	req.Header.Set("Content-Type", "application/json") // POST 請求需要設置 Content-Type
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Origin", "https://es.sdtbu.edu.cn") // 新增 Origin
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Referer", "https://es.sdtbu.edu.cn/") // 更新 Referer
	req.Header.Set("Accept-Encoding", "gzip, deflate")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Connection", "close")

	resp, err := client.Do(req)
	if err != nil {
		return 0, "", "", fmt.Errorf("發送 POST 請求失敗: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return resp.StatusCode, "", "", fmt.Errorf("讀取 POST 響應體失敗: %w", err)
	}

	var operateResponse OperateAPIResponse
	err = json.Unmarshal(body, &operateResponse)
	if err != nil {
		return resp.StatusCode, "", "", fmt.Errorf("解析 POST JSON 失敗: %w, 原始響應體:\n%s", err, string(body))
	}

	if operateResponse.Code != 0 {
		return resp.StatusCode, "", "", fmt.Errorf("空調操作 API 返回錯誤代碼: %d, 訊息: %s", operateResponse.Code, operateResponse.Msg)
	}

	return resp.StatusCode, operateResponse.Data.MsgID, operateResponse.Data.DeviceNo, nil
}

// printDeviceInfo 函數用於輸出設備信息
func printDeviceInfo(deviceInfo *DeviceInfo, statusCode int) {
	fmt.Println("==reponse==")
	fmt.Printf("回應狀態碼：%d\n", statusCode)
	fmt.Println("==回應訊息==")
	fmt.Printf("校   區：%s\n", deviceInfo.CampusTitle)
	fmt.Printf("宿舍樓號：%s\n", deviceInfo.BuildingTitle)
	fmt.Printf("樓   層：%s\n", deviceInfo.FloorTitle)
	fmt.Printf("門牌號：%s\n", deviceInfo.RoomNo)
	fmt.Printf("電費信息：%.2f\n", deviceInfo.Balance)

	// 顯示定時器狀態
	if timerActive {
		remaining := timerEndTime.Sub(time.Now())
		if remaining <= 0 {
			fmt.Println("定時器狀態：已過期，等待關閉空調。")
		} else {
			hours := int(remaining.Hours())
			minutes := int(remaining.Minutes()) % 60
			seconds := int(remaining.Seconds()) % 60
			fmt.Printf("定時器狀態：啟用中，將於 %s 後關閉空調 (%s 後，在 %s)。\n",
				timerDescription,
				fmt.Sprintf("%02d時%02d分%02d秒", hours, minutes, seconds),
				timerEndTime.Format("15:04:05"))
		}
	} else {
		fmt.Println("定時器狀態：未啟用。")
	}
	fmt.Println("===========")
}

// printInteractiveHelpMessage 函數用於輸出互動模式下的使用幫助
func printInteractiveHelpMessage() {
	fmt.Println("===================================")
	fmt.Println("         ACtool 使用幫助           ")
	fmt.Println("===================================")
	fmt.Println("輸入以下命令進行操作：")
	fmt.Println("  /status  - 獲取設備的詳細資訊 (包括定時器狀態)")
	fmt.Println("  /acon    - 開啟空調 (可選: /acon <分鐘>，設定分鐘定時)")
	fmt.Println("  /acoff   - 關閉空調")
	fmt.Println("  /timer <HH:MM> - 設定指定時間關閉空調 (24小時制)")
	fmt.Println("  /help    - 顯示此幫助訊息")
	fmt.Println("  /exit    - 退出程式")
	fmt.Println("===================================")
}

// printCommandLineHelpMessage 函數用於輸出命令行參數的使用幫助
func printCommandLineHelpMessage() {
	fmt.Println("===================================")
	fmt.Println("         ACtool 命令行使用幫助       ")
	fmt.Println("===================================")
	fmt.Println("使用以下參數啟動程式：")
	fmt.Println("  --status  - 獲取設備的詳細資訊 (包括定時器狀態)")
	fmt.Println("  --acon [分鐘] - 開啟空調 (可選: 帶分鐘參數，設定分鐘定時)")
	fmt.Println("  --acoff   - 關閉空調")
	fmt.Println("  --timer <HH:MM> - 設定指定時間關閉空調 (24小時制)")
	fmt.Println("  --help    - 顯示此幫助訊息")
	fmt.Println("===================================")
}

// handleTimer 處理定時器邏輯
func handleTimer(deviceInfo *DeviceInfo, token, studentName string) {
	// 如果定時器已經過期，則執行關閉操作
	if timerActive && time.Now().After(timerEndTime) {
		fmt.Println("\n定時器已到期，正在自動關閉空調...")
		_, _, _, err := operateDevice(deviceInfo, token, "acoff", studentName)
		if err != nil {
			fmt.Printf("自動關閉空調失敗: %v\n", err)
		} else {
			fmt.Println("空調已自動關閉。")
		}
		timerActive = false // 定時器完成
	}
}

func main() {
	var token, deviceNo, studentName string

	// 1. 嘗試從環境變數讀取
	token = os.Getenv("TOKEN")
	deviceNo = os.Getenv("DEVICENO")
	studentName = os.Getenv("STUDENTNAME")

	// 2. 如果環境變數未設定，嘗試從 actool.env 檔案讀取
	envFromFile, err := loadEnvFile("actool.env")
	if err != nil {
		fmt.Printf("警告: 無法讀取 actool.env 檔案: %v\n", err)
	} else {
		if token == "" {
			token = envFromFile["TOKEN"]
		}
		if deviceNo == "" {
			deviceNo = envFromFile["DEVICENO"]
		}
		if studentName == "" {
			studentName = envFromFile["STUDENTNAME"]
		}
	}

	// 3. 檢查所有必要變數是否已設置
	if token == "" {
		fmt.Println("錯誤: TOKEN 環境變數或 actool.env 中的 TOKEN 未設定。請設定。")
		os.Exit(1) // 退出程式
	}
	if deviceNo == "" {
		fmt.Println("錯誤: DEVICENO 環境變數或 actool.env 中的 DEVICENO 未設定。請設定。")
		os.Exit(1) // 退出程式
	}
	if studentName == "" {
		fmt.Println("錯誤: STUDENTNAME 環境變數或 actool.env 中的 STUDENTNAME 未設定。請設定。")
		os.Exit(1) // 退出程式
	}

	// 判斷是否帶有命令行參數啟動
	if len(os.Args) >= 2 {
		arg := os.Args[1]
		// 移除命令參數前的雙連字符 "--"
		commandArg := strings.TrimPrefix(strings.ToLower(arg), "--") // 確保參數也是小寫

		// 處理帶有時間參數的acon
		if commandArg == "acon" && len(os.Args) >= 3 {
			minutesStr := os.Args[2]
			minutes, err := strconv.Atoi(minutesStr)
			if err != nil || minutes <= 0 {
				fmt.Println("錯誤: --acon 後的定時分鐘數無效。請輸入正整數。")
				return
			}
			// 設置定時器
			timerActive = true
			timerEndTime = time.Now().Add(time.Duration(minutes) * time.Minute)
			timerDuration = time.Duration(minutes) * time.Minute
			timerDescription = fmt.Sprintf("%d分鐘", minutes)

			fmt.Printf("\n空調將在 %d 分鐘後自動關閉。\n", minutes)
			// 命令行啟動時，acon 後有時間參數，只執行開啟動作，不進入互動模式
			deviceInfo, statusCode, err := getDeviceInfo(deviceNo, token)
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				return
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acon", studentName)
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")
				// 在單次命令行模式下設置了定時器，但程式會立即退出，因此定時器不會生效。
				// 如果需要定時器生效，需要讓程式保持運行，這通常是互動模式或作為服務運行。
				// 根據 mind.md，命令行模式下設置定時器應保持程式運行。
				// 因此，對於帶有定時功能的命令行模式，程式不應立即退出，而應進入監聽模式。
				fmt.Println("定時任務已設定。程式將保持運行以監聽定時器。")
				runInteractiveMode(token, deviceNo, studentName) // 進入互動模式，監聽定時器
			}
			return // 處理完畢，退出命令行模式
		} else if commandArg == "timer" && len(os.Args) >= 3 {
			timeStr := os.Args[2] // HH:MM
			now := time.Now()
			targetTime, err := time.Parse("15:04", timeStr)
			if err != nil {
				fmt.Println("錯誤: --timer 後的時間格式無效。請使用 HH:MM 格式。")
				return
			}

			// 構建目標時間的日期部分，考慮是否為第二天
			targetDateTime := time.Date(now.Year(), now.Month(), now.Day(), targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())
			if targetDateTime.Before(now) {
				targetDateTime = targetDateTime.Add(24 * time.Hour) // 如果目標時間已過，則為第二天
			}

			timerActive = true
			timerEndTime = targetDateTime
			timerDuration = targetDateTime.Sub(now)
			timerDescription = fmt.Sprintf("指定時間 %s", timeStr)

			fmt.Printf("\n空調將在 %s 自動關閉。\n", timerEndTime.Format("2006-01-02 15:04:05"))
			fmt.Println("定時任務已設定。程式將保持運行以監聽定時器。")

			// 在設定定時後，開啟空調
			deviceInfo, statusCode, err := getDeviceInfo(deviceNo, token)
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				return
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acon", studentName)
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")
			}
			runInteractiveMode(token, deviceNo, studentName) // 進入互動模式，監聽定時器
			return                                           // 處理完畢，退出命令行模式
		}

		switch commandArg {
		case "status":
			fmt.Println("\n正在獲取設備信息...")
			deviceInfo, statusCode, err := getDeviceInfo(deviceNo, token)
			if err != nil {
				fmt.Printf("錯誤: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
			} else {
				printDeviceInfo(deviceInfo, statusCode)
			}
		case "acon": // 無定時參數的acon
			fmt.Println("\n正在開啟空調...")
			deviceInfo, statusCode, err := getDeviceInfo(deviceNo, token) // 重新獲取最新設備狀態
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				break // break from switch, then return from main
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acon", studentName) // Pass "acon" as action
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")
			}
		case "acoff":
			fmt.Println("\n正在關閉空調...")
			deviceInfo, statusCode, err := getDeviceInfo(deviceNo, token) // 重新獲取最新設備狀態
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				break // break from switch, then return from main
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acoff", studentName) // Pass "acoff" as action
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")
			}
		case "help":
			printCommandLineHelpMessage() // 呼叫新的命令行幫助函數
		default:
			fmt.Printf("無效的啓動參數：\"%s\"。\n", arg)
			fmt.Println("用法：./actool [--status | --acon [分鐘] | --acoff | --timer <HH:MM> | --help]")
			fmt.Println("例如：./actool --acon 30 開啟空調30分鐘")
			fmt.Println("例如：./actool --timer 23:30 在23:30關閉空調")
		}
		return // 帶有命令行參數時，執行完畢後直接退出
	}

	// 若未接受到命令參數，進入互動模式
	runInteractiveMode(token, deviceNo, studentName)
}

// runInteractiveMode 運行互動模式的主循環
func runInteractiveMode(token, deviceNo, studentName string) {
	// 先獲取基本設備信息並顯示
	fmt.Println("\n執行獲取設備信息功能...")
	deviceInfo, statusCode, err := getDeviceInfo(deviceNo, token)
	if err != nil {
		fmt.Printf("錯誤: %v\n", err)
		fmt.Println("請檢查您的配置或稍後再試。")
	} else {
		printDeviceInfo(deviceInfo, statusCode)
	}
	// 在顯示設備資訊後再顯示進入互動模式的提示
	fmt.Println("\n未檢測到命令行參數，進入互動模式。輸入 /help 獲取使用幫助。")

	// 進入互動模式的無限循環
	scanner := bufio.NewScanner(os.Stdin)
	for {
		// 在每次循環開始時處理定時器
		if timerActive {
			// 為了避免在每次循環中都進行網絡請求，只在必要時觸發關閉
			// 或者在一個單獨的 goroutine 中進行定時監聽和操作
			// 這裡為了簡化，仍然在主循環中檢查
			remaining := timerEndTime.Sub(time.Now())
			if remaining <= 0 {
				fmt.Println("\n定時器已到期，正在自動關閉空調...")
				deviceInfo, statusCode, err = getDeviceInfo(deviceNo, token) // 確保有最新的設備信息
				if err != nil {
					fmt.Printf("獲取設備信息失敗以執行自動關閉: %v\n", err)
					fmt.Printf("回應狀態碼：%d\n", statusCode)
				} else {
					_, _, _, opErr := operateDevice(deviceInfo, token, "acoff", studentName)
					if opErr != nil {
						fmt.Printf("自動關閉空調失敗: %v\n", opErr)
					} else {
						fmt.Println("空調已自動關閉。")
					}
				}
				timerActive = false // 定時器完成
			}
		}

		fmt.Print("> ") // 將提示符改為 "> "
		scanner.Scan()
		input := strings.TrimSpace(scanner.Text())
		commandParts := strings.Fields(strings.ToLower(input)) // 將輸入分割為命令和參數

		if len(commandParts) == 0 {
			continue // 忽略空輸入
		}

		command := commandParts[0]
		args := commandParts[1:]

		switch command {
		case "/status":
			fmt.Println("\n正在獲取設備信息...")
			deviceInfo, statusCode, err = getDeviceInfo(deviceNo, token)
			if err != nil {
				fmt.Printf("錯誤: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
			} else {
				printDeviceInfo(deviceInfo, statusCode)
			}
		case "/acon":
			var minutes int
			if len(args) > 0 {
				minutesStr := args[0]
				var parseErr error
				minutes, parseErr = strconv.Atoi(minutesStr)
				if parseErr != nil || minutes <= 0 {
					fmt.Println("錯誤: /acon 後的定時分鐘數無效。請輸入正整數。")
					break
				}
			}

			fmt.Println("\n正在開啟空調...")
			deviceInfo, statusCode, err = getDeviceInfo(deviceNo, token) // 重新獲取最新設備狀態
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				break
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acon", studentName)
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")
				if minutes > 0 {
					timerActive = true
					timerEndTime = time.Now().Add(time.Duration(minutes) * time.Minute)
					timerDuration = time.Duration(minutes) * time.Minute
					timerDescription = fmt.Sprintf("%d分鐘", minutes)
					fmt.Printf("已設定空調在 %d 分鐘後自動關閉。\n", minutes)
				} else {
					timerActive = false // 無定時
				}
			}
		case "/acoff":
			fmt.Println("\n正在關閉空調...")
			deviceInfo, statusCode, err = getDeviceInfo(deviceNo, token) // 重新獲取最新設備狀態
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				break
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acoff", studentName)
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")
				timerActive = false // 關閉空調時取消所有定時
				fmt.Println("定時器已取消。")
			}
		case "/timer":
			if len(args) == 0 {
				fmt.Println("錯誤: /timer 需要時間參數，例如 /timer 01:30。")
				break
			}
			timeStr := args[0] // HH:MM
			now := time.Now()
			targetTime, parseErr := time.Parse("15:04", timeStr)
			if parseErr != nil {
				fmt.Println("錯誤: /timer 後的時間格式無效。請使用 HH:MM 格式。")
				break
			}

			// 構建目標時間的日期部分，考慮是否為第二天
			targetDateTime := time.Date(now.Year(), now.Month(), now.Day(), targetTime.Hour(), targetTime.Minute(), 0, 0, now.Location())
			if targetDateTime.Before(now) {
				targetDateTime = targetDateTime.Add(24 * time.Hour) // 如果目標時間已過，則為第二天
			}

			// 在設定定時後，開啟空調
			fmt.Println("\n正在開啟空調並設定定時...")
			deviceInfo, statusCode, err = getDeviceInfo(deviceNo, token) // 重新獲取最新設備狀態
			if err != nil {
				fmt.Printf("獲取設備信息失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", statusCode)
				break
			}
			operateStatusCode, msgID, operateDeviceNo, err := operateDevice(deviceInfo, token, "acon", studentName)
			if err != nil {
				fmt.Printf("空調操作失敗: %v\n", err)
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				break
			} else {
				fmt.Println("==reponse==")
				fmt.Printf("回應狀態碼：%d\n", operateStatusCode)
				fmt.Println("==回應訊息==")
				fmt.Printf("訊息：%s\n", msgID)
				fmt.Printf("設備號：%s\n", operateDeviceNo)
				fmt.Println("===========")

				timerActive = true
				timerEndTime = targetDateTime
				timerDuration = targetDateTime.Sub(now)
				timerDescription = fmt.Sprintf("指定時間 %s", timeStr)
				fmt.Printf("已設定空調將在 %s (%s 後) 自動關閉。\n", timerEndTime.Format("15:04:05"), timerDuration.String())
			}
		case "/help":
			printInteractiveHelpMessage() // 呼叫原有的互動模式幫助函數
		case "/exit", "/quit": // 允許 /exit 或 /quit 退出
			fmt.Println("程式已退出。")
			return // 退出 main 函數，結束程式
		default:
			fmt.Println("無效的命令。請輸入 /help 查看可用命令。")
		}
	}
}
