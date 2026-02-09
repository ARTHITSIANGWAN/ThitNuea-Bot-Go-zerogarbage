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
	_ "github.com/mattn/go-sqlite3"
)

var (
	bot          *linebot.Client
	db           *sql.DB
	audioCache   = make(map[string][]byte)
	cacheMutex   sync.RWMutex
	myServerURL  string
)

func init() {
	var err error
	db, err = sql.Open("sqlite3", "./thitnuea_empire.db")
	if err != nil { log.Fatal(err) }
	db.Exec("CREATE TABLE IF NOT EXISTS knowledge (id INTEGER PRIMARY KEY, topic TEXT, insight TEXT, created_at DATETIME)")
}

func main() {
	myServerURL = os.Getenv("SERVER_URL")
	bot, _ = linebot.New(os.Getenv("LINE_CHANNEL_SECRET"), os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"))

	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/webhook/line", handleLineWebhook)
	http.HandleFunc("/audio/", handleAudioServe)

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }
	log.Printf("🤴 THITNUEA EMPEROR v4.0 [GEMINI 3 READY] | Port: %s", port)
	http.ListenAndServe(":"+port, nil)
}

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
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second) // เพิ่มเวลาให้ Gemini 3 คิด
	defer cancel()

	aiText := askGemini(ctx, userText)
	db.Exec("INSERT INTO knowledge (topic, insight, created_at) VALUES (?, ?, ?)", "User_Talk", userText+" -> "+aiText, time.Now())

	audioID, err := generateVoice(aiText)
	if err == nil {
		audioURL := fmt.Sprintf("%s/audio/%s.mp3", myServerURL, audioID)
		bot.ReplyMessage(replyToken, 
			linebot.NewTextMessage(aiText),
			linebot.NewAudioMessage(audioURL, 15000),
		).Do()
	} else {
		bot.ReplyMessage(replyToken, linebot.NewTextMessage(aiText)).Do()
	}
}

// 🧠 AI ENGINE: UPGRADED TO GEMINI 3 (No More Red Lines!)
func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	// ใช้ gemini-3-flash-preview ตามคู่มือล่าสุด 2026
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=" + apiKey
	
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]interface{}{{"text": prompt}}},
		},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "คุณคือ ThitNuea Emperor วิเคราะห์แบบมนุษย์ ตอบดุดัน ทรงพลัง ไร้ขยะ"}},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 1.0, // มาตรฐาน Gemini 3
			"topK":        40,
		},
		"thinkingConfig": map[string]interface{}{
			"thinking_level": "high", // ปล่อยให้พลายทองโชว์กึ๋น
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { 
		log.Printf("🐍 Snake Nudge: Connection Error")
		return "จักรพรรดิกำลังเดินทางผ่านมิติคอนเนคชั่น..." 
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	
	var res struct {
		Candidates []struct {
			Content struct {
				Parts []map[string]interface{} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "พลายแก้วกำลังแกะรหัสคำตอบ... (Unmarshal Error)"
	}

	if len(res.Candidates) > 0 && len(res.Candidates[0].Content.Parts) > 0 {
		// ดึง Text ออกมาอย่างปลอดภัย
		if text, ok := res.Candidates[0].Content.Parts[0]["text"].(string); ok {
			return text
		}
	}
	
	return "จักรพรรดิกำลังใช้ลายเซ็นความคิดสำรอง..."
}

func generateVoice(text string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	voiceID := "ErXw6udqS8tO90962vF"
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

	go func() {
		time.Sleep(5 * time.Minute)
		cacheMutex.Lock()
		delete(audioCache, audioID)
		cacheMutex.Unlock()
	}()
	return audioID, nil
}

func handleAudioServe(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path[len("/audio/"):]
	if len(path) < 4 { return }
	id := path[:len(path)-4]
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
	fmt.Fprintf(w, "<html><body style='background:black;color:#0f0'><h1>🛡️ THITNUEA EMPIRE v4.0 ACTIVE [3.0 READY]</h1><p>RAM Usage: %d MB</p></body></html>", m.Alloc/1024/1024)
}
