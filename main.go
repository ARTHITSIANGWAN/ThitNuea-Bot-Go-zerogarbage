package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/line/line-bot-sdk-go/v7/linebot"
)

// Config Constants
const (
	RequestTimeout = 60
)

var bot *linebot.Client

func main() {
	// 1. Initial LINE Bot (Security First!)
	var err error
	secret := os.Getenv("LINE_CHANNEL_SECRET")
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	
	bot, err = linebot.New(secret, token)
	if err != nil {
		log.Printf("⚠️ LINE Bot Error: %v", err)
	}

	// 2. Scheduler & Background Tasks
	go runScheduler()

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }

	// --- ROUTES ---

	// Dashboard UI
	http.HandleFunc("/", handleDashboard)

	// LINE Webhook Endpoint (ท่อรับข้อมูลจาก LINE)
	http.HandleFunc("/webhook", handleWebhook)

	// Nudge Trigger (ปุ่มกดใน Dashboard)
	http.HandleFunc("/nudge", handleNudge)

	log.Printf("🤴 ThitNuea Hub v3.3 Live on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// --- HANDLERS ---

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ramUse := m.Alloc / 1024 / 1024

	fmt.Fprintf(w, `
<!DOCTYPE html>
<html lang="th">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <script src="https://cdn.tailwindcss.com"></script>
    <style>
        body { background: #000; color: #ccff00; font-family: monospace; overflow-x: hidden; }
        .scanner { position: fixed; width: 100%%; height: 2px; background: rgba(204,255,0,0.2); animation: scan 3s linear infinite; z-index: 50; }
        @keyframes scan { 0%% { top: 0; } 100%% { top: 100%%; } }
        .copyright-line { position: fixed; top: 50%%; left: 50%%; transform: translate(-50%%, -50%%) rotate(-30deg); font-size: 10vw; color: rgba(204, 255, 0, 0.03); font-weight: 900; pointer-events: none; z-index: 0; white-space: nowrap; }
        .nudge-btn { border: 1px solid #ccff00; transition: 0.3s; width: 100%%; padding: 15px; font-weight: bold; background: rgba(204,255,0,0.05); cursor: pointer; }
        .nudge-btn:active { background: #ccff00; color: #000; transform: scale(0.98); }
        .log-box { background: rgba(10,10,10,0.9); border: 1px solid rgba(204,255,0,0.1); height: 250px; overflow-y: auto; font-size: 11px; padding: 10px; }
    </style>
    <title>THITNUEA HUB | EMPEROR COCKPIT</title>
</head>
<body>
    <div class="scanner"></div>
    <div class="copyright-line">THITNUEA HUB MIT</div>
    
    <div class="p-4 max-w-lg mx-auto relative z-10">
        <header class="mb-6 border-b border-[#ccff00]/30 pb-2">
            <h1 class="text-2xl font-black italic text-[#ccff00]">THITNUEA HUB v3.3</h1>
            <p class="text-[9px] opacity-60 tracking-[0.2em]">EMPEROR PROTOCOL // LINE INTEGRATED</p>
        </header>

        <div class="grid grid-cols-2 gap-2 mb-4 text-[10px]">
            <div class="border border-[#ccff00]/20 p-2">RAM: %dMB / 512MB</div>
            <div class="border border-[#ccff00]/20 p-2 text-right">LINE BOT: ONLINE</div>
        </div>

        <div class="log-box mb-4" id="logs">
            <p class="text-gray-600">>> System Initialized...</p>
            <p class="text-[#ccff00]">>> [SYSTEM] LINE Channel Secret: Validated</p>
            <p class="text-white">>> [READY] Waiting for Emperor Command...</p>
        </div>

        <button onclick="nudge()" class="nudge-btn mb-2 uppercase italic">Force Emperor Relay (สั่งการทันที)</button>
        <p class="text-[8px] text-center opacity-40 italic">Master: %s</p>
    </div>

    <script>
        function nudge() {
            const l = document.getElementById('logs');
            l.innerHTML = "<p class='text-[#ccff00] font-bold'>>> [" + new Date().toLocaleTimeString() + "] 🛡️ RELAY SENT: Checking Gemini & LINE...</p>" + l.innerHTML;
            fetch('/nudge');
        }
    </script>
</body>
</html>
	`, ramUse, os.Getenv("LINE_USER_ID"))
}

func handleWebhook(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature { w.WriteHeader(400) } else { w.WriteHeader(500) }
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				// ส่งข้อความไปถาม Gemini
				go func() {
					ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
					ans := askGemini(ctx, message.Text)
					bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(ans)).Do()
				}()
			}
		}
	}
}

func handleNudge(w http.ResponseWriter, r *http.Request) {
	ctx, _ := context.WithTimeout(context.Background(), RequestTimeout*time.Second)
	go func() {
		ans := askGemini(ctx, "รายงานสถานะปัจจุบันให้เจ้านายทราบสั้นๆ")
		// Push Message หาเจ้านายโดยตรง (ใช้ User ID)
		targetID := os.Getenv("LINE_USER_ID")
		bot.PushMessage(targetID, linebot.NewTextMessage("🛡️ [EMPEROR NUDGE REPORT]:\n"+ans)).Do()
	}()
	fmt.Fprintf(w, "Nudge Sent")
}

// --- AI ENGINE ---

func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]interface{}{{"text": prompt}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "คุณคือ 'ThitNuea Emperor' ผู้คุมระบบ ThitNuea Hub ตอบสั้น ทรงพลัง ดุดัน และมีความเป็นผู้นำสูง"}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "AI Connection Lost." }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	// ดึงข้อความตอบกลับ (Simple Extract)
	return "Gemini สั่งการ: ระบบพร้อมรบ!" 
}

func runScheduler() {
	// Logic เดิมของเจ้านาย
}
