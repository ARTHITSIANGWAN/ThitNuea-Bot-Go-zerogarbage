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

const RequestTimeout = 60

var bot *linebot.Client

func main() {
	var err error
	secret := os.Getenv("LINE_CHANNEL_SECRET")
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	
	bot, err = linebot.New(secret, token)
	if err != nil {
		log.Printf("⚠️ LINE Bot Initial Error: %v", err)
	}

	go runScheduler()

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }

	// --- ROUTES ---
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/webhook", handleWebhook) // มั่นใจว่าใน LINE ใส่ลิงก์ลงท้ายด้วย /webhook
	http.HandleFunc("/nudge", handleNudge)

	log.Printf("🤴 ThitNuea Hub v3.3.1 [EMPEROR EDITION] Live on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

// --- WEBHOOK HANDLER (จุดสำคัญที่ทำให้บอทตอบ) ---
func handleWebhook(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				// ตอบกลับทันที (Prevent Timeout)
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				ans := askGemini(ctx, message.Text)
				cancel()
				
				if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(ans)).Do(); err != nil {
					log.Printf("Error replying: %v", err)
				}
			}
		}
	}
	w.WriteHeader(200) // บอก LINE ว่าได้รับข้อมูลแล้ว
}

// --- AI ENGINE (ดึงคำตอบจาก Gemini จริงๆ) ---
func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]interface{}{{"text": prompt}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "คุณคือ 'ThitNuea Emperor' ผู้คุมระบบ ThitNuea Hub ตอบเป็นภาษาไทย สั้น ทรงพลัง ดุดัน และมีความเป็นผู้นำสูง"}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "🛡️ การเชื่อมต่อขัดข้อง: ติดต่อ Gemini ไม่ได้" }
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	
	// โครงสร้างการแกะ JSON ของ Gemini
	var data struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(body, &data); err != nil || len(data.Candidates) == 0 {
		return "🛡️ จักรพรรดิขัดข้อง: ไม่สามารถประมวลผลคำสั่ง"
	}

	return data.Candidates[0].Content.Parts[0].Text
}

// --- DASHBOARD UI & OTHERS (คงเดิมแต่ปรับปรุงเล็กน้อย) ---
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
        body { background: #000; color: #ccff00; font-family: monospace; }
        .copyright-line { position: fixed; top: 50%%; left: 50%%; transform: translate(-50%%, -50%%) rotate(-30deg); font-size: 8vw; color: rgba(204, 255, 0, 0.05); pointer-events: none; z-index: 0; white-space: nowrap; }
    </style>
    <title>THITNUEA HUB | COCKPIT</title>
</head>
<body class="p-6">
    <div class="copyright-line">THITNUEA HUB MIT</div>
    <div class="max-w-md mx-auto relative z-10 border border-[#ccff00] p-6 bg-black/80">
        <h1 class="text-3xl font-bold italic text-[#ccff00] mb-4">THITNUEA HUB v3.3.1</h1>
        <div class="text-sm space-y-2 mb-6">
            <p>STATUS: <span class="text-white animate-pulse">● ONLINE</span></p>
            <p>RAM USAGE: %d MB</p>
            <p>REGION: VIRGINIA (RENDER)</p>
        </div>
        <button onclick="nudge()" class="w-full border border-[#ccff00] py-4 hover:bg-[#ccff00] hover:text-black transition">FORCE NUDGE</button>
        <div id="logs" class="mt-4 text-[10px] text-gray-500 h-20 overflow-auto border-t border-[#ccff00]/20 pt-2"></div>
    </div>
    <script>
        function nudge() {
            fetch('/nudge').then(() => {
                document.getElementById('logs').innerHTML += ">> Nudge Sent Successfully<br>";
            });
        }
    </script>
</body>
</html>
	`, ramUse)
}

func handleNudge(w http.ResponseWriter, r *http.Request) {
	targetID := os.Getenv("LINE_USER_ID")
	if targetID == "" {
		fmt.Fprintf(w, "Error: No User ID")
		return
	}
	ctx, _ := context.WithTimeout(context.Background(), 30*time.Second)
	ans := askGemini(ctx, "รายงานสถานะสั้นๆ ในฐานะจักรพรรดิ")
	bot.PushMessage(targetID, linebot.NewTextMessage("🛡️ [SYSTEM REPORT]:\n"+ans)).Do()
	fmt.Fprintf(w, "Nudge Sent")
}

func runScheduler() {
	// ใส่ Logic เดิมของเจ้านายตรงนี้
}
