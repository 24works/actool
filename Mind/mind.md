# ACtool 开发草稿

---

## 首次開發

### 基礎功能

**啓動時** ： 讀取環境變量或從 `actool.env` 中讀取 `token` `deviceNo` `studentNameForAC`

1. ***使用GET獲取設備信息***
`GET` `/hatch-api/api/sdgongshang/device/getDeviceByNo`

2. ***確保正確設置token與deviceNo***
`deviceNo=` `000000000000`
`Token: ` `csioewnvoqw12396xc76`

3. ***測試請求通過*** (相關檔案 `GetdeviceNo.json` )
    - 解析json響應主體
    - 列印json中的部分鍵值内容
        - `campusTitle` : `校區`
        - `buildingTitle` : `宿舍樓號`
        - `floorTitle` : `樓層`
        - `roomNo` : `門牌號`
        - `balance` : `電費信息`

4. ***輸出内容模板***
```bash
==reponse==
回應狀態碼：
==回應訊息==
校   區：
宿舍樓號：
樓   層：
門牌號：
電費信息：
===========
```

### 空調開關
**步驟**
1. 讀取命令行 `/start` 為開 `/stop` 為關閉
    - 允許程式接收啓動參數，例如在Windows中使用powershell命令 `./actool.exe -start`來開啓
2. ***使用GET獲取設備信息***
3. 設置json相關内容並發出請求（相關檔案 `AirOpen.json` ）
4. 讀取回應内容
```json
{"code":0,"msg":"success","data":{"msgId":"0000000000000000000","deviceNo":"000000000000"}}
```
5. ***輸出内容模板***
```bash
==reponse==
回應狀態碼：
==回應訊息==
訊息： msg的内容
設備號： deviceNo的内容
===========
```


## 進階開發

### 進階功能一

- 若啓動時未接受到命令參數，先獲取基本設備信息保持程式運行，並允許用戶在命令行中輸入命令
    - 示例命令：
        - `/status` 輸出設備參數 `getDeviceInfo`
        - `/acon` 開啓空調
        - `/acoff` 關閉空調
        - `/help` 獲取使用幫助

### 進階功能二

#### ***定時功能***

***分鐘定時***
單位：`分鐘`

判斷`--acon`或`/acon`之後有無參數
例如儅啓動時為`--acon 5`或`/acon 5`則為開啓空調5分鐘
5分鐘后程式控制空調自動關閉
若啓動時為`--acon`或`/acon`，後面無參數則默認爲不啓用定時功能

***指定時間定時***
全程使用24小時制

新增`--timer`或`/timer`指令
啓動命令示例`--timer 01:30`或`/timer 01:30`
actool全程使用24小時制，判斷若當前時間，若位於參數設置的時間之後則參數為第二天
例如當前的時間為22:30，參數提供的時間為01:30，22>01，則定時器在第二日的01:30關閉空調
這裏使用22與1做比較只是我的例子，你可以根據你的思路使用最佳的方法進行判斷

在執行timer指令後向空調傳送開機指令，然後保持程式運行
同時可以使用status命令查看定時器設置的時間與實時剩餘時間（精確到秒）
若沒有啓用定時器則status返回的結果中
