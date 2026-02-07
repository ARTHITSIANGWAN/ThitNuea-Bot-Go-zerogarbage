package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/line/line-bot-sdk-go/v7/linebot"
	_ "github.com/mattn/go-sqlite3" // คลังความจำระยะยาว
)

// --- GLOBAL VARIABLES ---
var (
	bot          *linebot.Client
	db           *sql.DB
	audioCache   = make(map[string][]byte)
	cacheMutex   sync.RWMutex
	myServerURL  string
)

func init() {
	// 📖 1. เตรียมคลังสมอง SQLite
	var err error
	db, err = sql.Open("sqlite3", "./thitnuea_empire.db")
	if err != nil { log.Fatal(err) }
	db.Exec("CREATE TABLE IF NOT EXISTS knowledge (id INTEGER PRIMARY KEY, topic TEXT, insight TEXT, created_at DATETIME)")
}

func main() {
	// 2. Setup Environment
	myServerURL = os.Getenv("SERVER_URL")
	bot, _ = linebot.New(os.Getenv("LINE_CHANNEL_SECRET"), os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))

	// 3. Routes (The Emperor Gateway)
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/webhook/line", handleLineWebhook)
	http.HandleFunc("/audio/", handleAudioServe)

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }
	log.Printf("🤴 THITNUEA EMPEROR v4.0 [FINAL] | Port: %s", port)
	http.ListenAndServe(":"+port, nil)
}

// 🛡️ LINE WEBHOOK (พลายแก้ว 🏍️ รับงาน)
func handleLineWebhook(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil { return }

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			if message, ok := event.Message.(*linebot.TextMessage); ok {
				go processEmperorLogic(event.ReplyToken, message.Text)
			}
		}
	}
	w.WriteHeader(200)
}

func processEmperorLogic(replyToken, userText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// --- 🧠 STEP 1: ถามสมอง AI (Gemini) ---
	aiText := askGemini(ctx, userText)

	// --- 📖 STEP 2: บันทึกลงคลังความจำ SQLite ---
	db.Exec("INSERT INTO knowledge (topic, insight, created_at) VALUES (?, ?, ?)", "User_Talk", userText+" -> "+aiText, time.Now())

	// --- 🔊 STEP 3: สร้างเสียง (ElevenLabs) ---
	audioID, err := generateVoice(aiText)

	// --- 🏁 STEP 4: ตอบกลับแบบคอมโบ (ข้อความ + เสียง) ---
	if err == nil {
		audioURL := fmt.Sprintf("%s/audio/%s.mp3", myServerURL, audioID)
		bot.ReplyMessage(replyToken, 
			linebot.NewTextMessage(aiText),
			linebot.NewAudioMessage(audioURL, 15000), // 15 วินาที
		).Do()
	} else {
		bot.ReplyMessage(replyToken, linebot.NewTextMessage(aiText)).Do()
	}
}

// 🧠 AI ENGINE
func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]interface{}{{"text": prompt}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "คุณคือ ThitNuea Emperor วิเคราะห์แบบมนุษย์ มีรอยยิ้ม ตอบดุดัน ทรงพลัง"}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "ระบบสื่อสารขัดข้อง" }
	defer resp.Body.Close()

	var data struct {
		Candidates []struct {
			Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"`
		} `json:"candidates"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if len(data.Candidates) > 0 { return data.Candidates[0].Content.Parts[0].Text }
	return "จักรพรรดิกำลังใช้ความคิด..."
}

// 🔊 VOICE ENGINE (ElevenLabs + RAM Cache)
func generateVoice(text string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	voiceID := "ErXw6udqS8tO90962vF" // ใส่ ID เสียงของทีมงาน
	url := "https://api.elevenlabs.io/v1/text-to-speech/" + voiceID
	
	payload, _ := json.Marshal(map[string]interface{}{
		"text": text,
		"model_id": "eleven_multilingual_v2",
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 { return "", fmt.Errorf("voice failed") }
	defer resp.Body.Close()

	voiceData, _ := io.ReadAll(resp.Body)
	audioID := uuid.New().String()

	cacheMutex.Lock()
	audioCache[audioID] = voiceData
	cacheMutex.Unlock()

	// Zero-Garbage: ลบไฟล์เสียงใน 5 นาที
	go func() {
		time.Sleep(5 * time.Minute)
		cacheMutex.Lock()
		delete(audioCache, audioID)
		cacheMutex.Unlock()
	}()

	return audioID, nil
}

func handleAudioServe(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/audio/"):]
	id = id[:len(id)-4] // ลบ .mp3

	cacheMutex.RLock()
	data, exists := audioCache[id]
	cacheMutex.RUnlock()

	if !exists { w.WriteHeader(404); return }
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Write(data)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "<html><body style='background:black;color:#0f0'><h1>🛡️ THITNUEA EMPIRE v4.0 ACTIVE</h1><p>RAM Usage: %d MB</p></body></html>", m.Alloc/1024/1024)
}
