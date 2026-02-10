package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/line/line-bot-sdk-go/v7/linebot"
	_ "github.com/mattn/go-sqlite3"
)

// --- 1. ตั้งค่าจักรวรรดิ ---
const (
	PaypalLink      = "https://paypal.me/arthitsiangwan" // 💎 ท่อลำเลียงทรัพย์
)

var (
	bot *linebot.Client
	db  *sql.DB
)

func main() {
	// เริ่มต้นคลังปัญญา & คลังสมบัติ
	initEmpireVault()

	var err error
	bot, err = linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil { log.Println("⚠️ LINE Bot Warning:", err) }

	// Route สำหรับ Webhook และ Dashboard
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/webhook/line", handleLineWebhook) // ท่อหลัก LINE
	
	// หมายเหตุ: ตัด /command ออกชั่วคราวเพื่อให้ผ่าน Build ก่อน
	// http.HandleFunc("/command", handleEmperorCommand)   

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }
	
	fmt.Printf("👑 THITNUEA EMPIRE | 💰 MONEY MODE: ON | Port: %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// --- 2. สมองส่วนจัดการ LINE (Dispatcher Logic) ---
func handleLineWebhook(w http.ResponseWriter, r *http.Request) {
	if bot == nil { w.WriteHeader(500); return }
	events, err := bot.ParseRequest(r)
	if err != nil { w.WriteHeader(400); return }

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			if message, ok := event.Message.(*linebot.TextMessage); ok {
				userMsg := strings.ToLower(message.Text)

				// 💎 MONEY TRAP: ดักจับคีย์เวิร์ดทำเงินก่อนเสมอ!
				if isMoneyKeyword(userMsg) {
					go logToVault("Money_Opportunity", "User สนใจเปย์: "+userMsg)
					replyFlexPayment(event.ReplyToken)
				} else {
					// ตอบกลับปกติ
					replyText(event.ReplyToken, "💎 แก้วตา: รับทราบค่ะ! ขอบคุณที่ทักทายจักรวรรดิ ThitNueaHub นะคะ")
				}
			}
		}
	}
	w.WriteHeader(200)
}

// --- 3. ฟังก์ชันและ Logic เสริม ---

func isMoneyKeyword(text string) bool {
	keywords := []string{"สมัคร", "vip", "donate", "เปย์", "สนับสนุน", "เลขบัญชี", "พร้อมเพย์", "money"}
	for _, k := range keywords {
		if strings.Contains(text, k) { return true }
	}
	return false
}

// ส่ง Flex Message แบบสวยงามดูแพง
func replyFlexPayment(replyToken string) {
	// JSON Flex Message: การ์ดเชิญชวนแบบ Premium
	flexJSON := fmt.Sprintf(`{
		"type": "bubble",
		"hero": {
			"type": "image",
			"url": "https://cdn-icons-png.flaticon.com/512/2454/2454269.png", 
			"size": "full",
			"aspectRatio": "20:13",
			"aspectMode": "cover"
		},
		"body": {
			"type": "box",
			"layout": "vertical",
			"contents": [
				{"type": "text", "text": "💎 ThitNuea Premium", "weight": "bold", "size": "xl", "color": "#1DB446"},
				{"type": "text", "text": "ปลดล็อกพลัง AI ระดับเทพ!", "size": "md", "weight": "bold"},
				{"type": "text", "text": "ร่วมเป็นผู้สนับสนุนทีมงานเพื่อพัฒนาเทคโนโลยีเพื่อสังคม", "wrap": true, "size": "sm", "color": "#666666", "margin": "md"}
			]
		},
		"footer": {
			"type": "box",
			"layout": "vertical",
			"spacing": "sm",
			"contents": [
				{
					"type": "button",
					"style": "primary",
					"height": "sm",
					"color": "#00308F",
					"action": {
						"type": "uri",
						"label": "👉 เปย์เลย (PayPal)",
						"uri": "%s"
					}
				},
				{"type": "text", "text": "ขอบคุณที่สนับสนุนความฝันครับ ❤️", "size": "xs", "align": "center", "color": "#aaaaaa", "margin": "md"}
			]
		}
	}`, PaypalLink)

	container, err := linebot.UnmarshalFlexMessageJSON([]byte(flexJSON))
	if err != nil {
		// ถ้า Flex พัง ให้ส่ง Text สำรอง
		replyText(replyToken, "💎 สนับสนุนได้ที่: "+PaypalLink)
		return
	}
	bot.ReplyMessage(replyToken, linebot.NewFlexMessage("💎 สารจากจักรวรรดิ: โอกาสสนับสนุน", container)).Do()
}

func replyText(token, text string) {
	bot.ReplyMessage(token, linebot.NewTextMessage(text)).Do()
}

// --- 4. ระบบหลังบ้าน (Dashboard & DB) ---

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "<h1>💎 THITNUEA MONEY HUB IS ACTIVE</h1><h3>Status: Ready to Receive Wealth</h3>")
}

func initEmpireVault() {
	var err error
	db, err = sql.Open("sqlite3", "./thitnuea_empire.db")
	if err != nil { log.Println("⚠️ DB Error (Ignore if using ephemeral fs):", err) }
	// สร้างตารางเก็บ Log
	db.Exec("CREATE TABLE IF NOT EXISTS empire_logs (id INTEGER PRIMARY KEY, event TEXT, details TEXT, timestamp DATETIME)")
}

func logToVault(event, details string) {
	if db != nil {
		db.Exec("INSERT INTO empire_logs (event, details, timestamp) VALUES (?, ?, ?)", event, details, time.Now())
	}
	fmt.Printf("💰 [Money Log]: %s - %s\n", event, details)
}
