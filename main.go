package main

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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
)

// --- CONFIGURATION ---
const (
	RequestTimeout = 60
	ServerPort     = "10000" // Render Default
)

// --- GLOBAL VARIABLES (Thread-Safe) ---
var (
	bot          *linebot.Client
	audioCache   = make(map[string][]byte) // เก็บไฟล์เสียงใน RAM ชั่วคราว
	cacheMutex   sync.RWMutex              // กุญแจล็อคป้องกัน RAM ตีกัน
	myServerURL  string                    // URL ของ Server เรา (ไว้ทำลิงก์เสียง)
)

func main() {
	// 1. Setup Environment
	var err error
	myServerURL = os.Getenv("SERVER_URL") // ต้องตั้งใน Render Env (เช่น https://thitnuea-app.onrender.com)
	if myServerURL == "" { log.Fatal("⚠️ SETUP ERROR: กรุณาตั้งค่า SERVER_URL ใน Environment Variables") }

	// 2. Initialize LINE Bot
	bot, err = linebot.New(
		os.Getenv("LINE_CHANNEL_SECRET"),
		os.Getenv("LINE_CHANNEL_ACCESS_TOKEN"),
	)
	if err != nil { log.Fatalf("⚠️ LINE Init Failed: %v", err) }

	// 3. Start Scheduler (Gas Station Logic)
	go runScheduler()

	// 4. Setup Routes (Emperor Gateway)
	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/webhook/line", handleLineWebhook)       // ทางด่วน LINE
	http.HandleFunc("/webhook/messenger", handleFBWebhook)    // ทางด่วน Messenger
	http.HandleFunc("/audio/", handleAudioServe)              // 🔊 ช่องทางเสิร์ฟไฟล์เสียง (แก้บอทใบ้)
	http.HandleFunc("/nudge", handleNudge)

	// 5. Start Server
	port := os.Getenv("PORT")
	if port == "" { port = ServerPort }
	log.Printf("🤴 THITNUEA EMPEROR v3.5 READY | Port: %s | Server: %s", port, myServerURL)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// ==========================================
// 🛡️ LINE WEBHOOK HANDLER
// ==========================================
func handleLineWebhook(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature { w.WriteHeader(400) } else { w.WriteHeader(500) }
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				go processLineMessage(event.ReplyToken, message.Text) // รันแยก Thread (Go Routine) เพื่อความไว
			}
		}
	}
	w.WriteHeader(200)
}

func processLineMessage(replyToken, userText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	// 1. ถาม Gemini
	aiText := askGemini(ctx, userText)

	// 2. สร้างเสียง (ElevenLabs)
	audioID, err := generateVoice(aiText)
	
	// 3. ตอบกลับ (ถ้าสร้างเสียงได้ ส่งเสียงด้วย / ถ้าไม่ได้ ส่งแค่ข้อความ)
	if err == nil {
		audioURL := fmt.Sprintf("%s/audio/%s.mp3", myServerURL, audioID)
		duration := 10000 // สมมติ 10 วิ (ElevenLabs ไม่บอก duration ต้องกะเอา หรือใช้ ffmpeg เช็ค)
		if _, err := bot.ReplyMessage(
			replyToken,
			linebot.NewTextMessage(aiText),
			linebot.NewAudioMessage(audioURL, duration),
		).Do(); err != nil {
			log.Printf("❌ Reply Error: %v", err)
		}
	} else {
		bot.ReplyMessage(replyToken, linebot.NewTextMessage(aiText)).Do()
	}
}

// ==========================================
// 🛡️ FACEBOOK MESSENGER WEBHOOK HANDLER
// ==========================================
func handleFBWebhook(w http.ResponseWriter, r *http.Request) {
	// A. Verification Request (ตอนเชื่อมต่อครั้งแรก)
	if r.Method == "GET" {
		verifyToken := os.Getenv("FB_VERIFY_TOKEN")
		if r.URL.Query().Get("hub.verify_token") == verifyToken {
			fmt.Fprintf(w, r.URL.Query().Get("hub.challenge"))
			return
		}
		w.WriteHeader(http.StatusForbidden)
		return
	}

	// B. Message Handling (ตอนมีข้อความเข้า)
	var payload struct {
		Entry []struct {
			Messaging []struct {
				Sender struct{ ID string `json:"id"` } `json:"sender"`
				Message struct{ Text string `json:"text"` } `json:"message"`
			} `json:"messaging"`
		} `json:"entry"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
		if len(payload.Entry) > 0 && len(payload.Entry[0].Messaging) > 0 {
			event := payload.Entry[0].Messaging[0]
			go processFBMessage(event.Sender.ID, event.Message.Text)
		}
	}
	w.WriteHeader(200)
}

func processFBMessage(senderID, text string) {
	if text == "" { return }
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	ans := askGemini(ctx, text)
	sendFBMessage(senderID, ans)
}

func sendFBMessage(recipientID, text string) {
	url := "https://graph.facebook.com/v18.0/me/messages?access_token=" + os.Getenv("FB_PAGE_ACCESS_TOKEN")
	body, _ := json.Marshal(map[string]interface{}{
		"recipient": map[string]string{"id": recipientID},
		"message":   map[string]string{"text": text},
	})
	http.Post(url, "application/json", bytes.NewBuffer(body))
}

// ==========================================
// 🧠 AI & VOICE ENGINE (THE CORE)
// ==========================================
func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]interface{}{{"text": prompt}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "คุณคือ ThitNuea Emperor ตอบสั้น กระชับ ดุดัน และจริงใจ"}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "ระบบสื่อสารผิดพลาด" }
	defer resp.Body.Close()

	var data struct {
		Candidates []struct {
			Content struct { Parts []struct { Text string `json:"text"` } `json:"parts"` } `json:"content"`
		} `json:"candidates"`
	}
	json.NewDecoder(resp.Body).Decode(&data)
	if len(data.Candidates) > 0 { return data.Candidates[0].Content.Parts[0].Text }
	return "ขัดข้องทางเทคนิค"
}

func generateVoice(text string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" { return "", fmt.Errorf("no key") }
	
	voiceID := "ErXw6udqS8tO90962vF" // ใส่ ID เสียงที่ต้องการ
	url := "https://api.elevenlabs.io/v1/text-to-speech/" + voiceID
	
	payload, _ := json.Marshal(map[string]interface{}{
		"text": text,
		"model_id": "eleven_multilingual_v2",
		"voice_settings": map[string]float64{"stability": 0.5, "similarity_boost": 0.8},
	})

	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(payload))
	req.Header.Set("xi-api-key", apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != 200 { return "", fmt.Errorf("voice failed") }
	defer resp.Body.Close()

	voiceData, _ := io.ReadAll(resp.Body)
	
	// เก็บลง RAM Cache
	audioID := uuid.New().String()
	cacheMutex.Lock()
	audioCache[audioID] = voiceData
	cacheMutex.Unlock()

	// ตั้งเวลาลบไฟล์ทิ้งใน 5 นาที (Zero-Garbage)
	go func() {
		time.Sleep(5 * time.Minute)
		cacheMutex.Lock()
		delete(audioCache, audioID)
		cacheMutex.Unlock()
	}()

	return audioID, nil
}

// เสิร์ฟไฟล์เสียงจาก RAM
func handleAudioServe(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/audio/"):] // ตัดคำว่า /audio/ ออกเหลือแค่ ID
	id = id[:len(id)-4] // ตัด .mp3 ออก

	cacheMutex.RLock()
	data, exists := audioCache[id]
	cacheMutex.RUnlock()

	if !exists {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Write(data)
}

// ==========================================
// 📟 DASHBOARD & UTILS
// ==========================================
func handleDashboard(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	fmt.Fprintf(w, "<html><body style='background:black;color:#0f0'><h1>THITNUEA EMPEROR ACTIVE</h1><p>RAM: %d MB</p><p>Audio Cache: %d files</p></body></html>", m.Alloc/1024/1024, len(audioCache))
}

func handleNudge(w http.ResponseWriter, r *http.Request) { /* ... Logic เดิม ... */ }
func runScheduler() { /* ... Logic เดิม ... */ }
