package main // แก้เป็นตัวพิมพ์เล็กแล้วครับเจ้านาย!

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
	// สร้างโฟลเดอร์ data เพื่อเก็บ DB ให้ปลอดภัยขึ้น (ถ้ามี)
	db, err = sql.Open("sqlite3", "./thitnuea_empire.db")
	if err != nil {
		log.Fatal("❌ DB Error:", err)
	}
	db.Exec("CREATE TABLE IF NOT EXISTS knowledge (id INTEGER PRIMARY KEY, topic TEXT, insight TEXT, created_at DATETIME)")
}

func main() {
	myServerURL = os.Getenv("SERVER_URL")
	
	// ป้องกันบอท Panic ถ้าลืมใส่คีย์
	secret := os.Getenv("LINE_CHANNEL_SECRET")
	token := os.Getenv("LINE_CHANNEL_ACCESS_TOKEN")
	
	var err error
	bot, err = linebot.New(secret, token)
	if err != nil {
		log.Printf("⚠️ LINE Bot Init Warning: %v", err)
	}

	http.HandleFunc("/", handleDashboard)
	http.HandleFunc("/webhook/line", handleLineWebhook)
	http.HandleFunc("/audio/", handleAudioServe)

	port := os.Getenv("PORT")
	if port == "" {
		port = "10000"
	}
	log.Printf("🤴 THITNUEA EMPEROR v4.1 [FIXED & READY] | Port: %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleLineWebhook(w http.ResponseWriter, r *http.Request) {
	if bot == nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	events, err := bot.ParseRequest(r)
	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(http.StatusBadRequest)
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
		return
	}
	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			if message, ok := event.Message.(*linebot.TextMessage); ok {
				// ส่งเข้าประมวลผลแยก Thread เพื่อความเร็ว
				go processEmperorLogic(event.ReplyToken, message.Text)
			}
		}
	}
	w.WriteHeader(http.StatusOK)
}

func processEmperorLogic(replyToken, userText string) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	defer cancel()

	aiText := askGemini(ctx, userText)
	
	// บันทึกลงคลังความรู้จักรวรรดิ
	_, _ = db.Exec("INSERT INTO knowledge (topic, insight, created_at) VALUES (?, ?, ?)", "User_Talk", userText+" -> "+aiText, time.Now())

	audioID, err := generateVoice(aiText)
	if err == nil && myServerURL != "" {
		audioURL := fmt.Sprintf("%s/audio/%s.mp3", myServerURL, audioID)
		_, _ = bot.ReplyMessage(replyToken, 
			linebot.NewTextMessage(aiText),
			linebot.NewAudioMessage(audioURL, 15000),
		).Do()
	} else {
		_, _ = bot.ReplyMessage(replyToken, linebot.NewTextMessage(aiText)).Do()
	}
}

func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" { return "เจ้านายลืมวางกุญแจ GEMINI_API_KEY ครับ!" }

	// ใช้โมเดลล่าสุดตามที่เจ้านายเลือก
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=" + apiKey
	
	payload := map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": "คุณคือ ThitNuea Emperor AI ผู้ปกครองจักรวรรดิ คุยแบบมนุษย์ ทรงพลัง ดุดัน และประหยัดถ้อยคำ"},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.9,
		},
	}

	jsonData, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { 
		return "มิติเชื่อมต่อขัดข้อง: " + err.Error()
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	
	// แกะโครงสร้าง JSON ของ Gemini 3 แบบละเอียด
	var res struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error struct {
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "พลายแก้วอ่านลายมือ Gemini 3 ไม่ค่อยออกครับ..."
	}

	if res.Error.Message != "" {
		return "Gemini แจ้งว่า: " + res.Error.Message
	}

	if len(res.Candidates) > 0 && len(res.Candidates[0].Content.Parts) > 0 {
		return res.Candidates[0].Content.Parts[0].Text
	}
	
	return "จักรพรรดิใช้ความเงียบสยบความเคลื่อนไหว... (ไม่มีคำตอบ)"
}

func generateVoice(text string) (string, error) {
	apiKey := os.Getenv("ELEVENLABS_API_KEY")
	if apiKey == "" { return "", fmt.Errorf("no key") }
	
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

	// ลบไฟล์เสียงหลังผ่านไป 5 นาทีเพื่อประหยัด RAM
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
	id := path[:len(path)-4] // ตัด .mp3 ออก
	cacheMutex.RLock()
	data, exists := audioCache[id]
	cacheMutex.RUnlock()
	if !exists { 
		w.WriteHeader(404)
		return 
	}
	w.Header().Set("Content-Type", "audio/mpeg")
	w.Write(data)
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body style='background:#0a0a0a;color:#00ff00;font-family:monospace;padding:50px;'>"+
		"<h1>🛡️ THITNUEA EMPIRE v4.1</h1>"+
		"<h3>STATUS: <span style='color:white'>ACTIVE [GEMINI 3.0]</span></h3>"+
		"<p>RAM USAGE: %d MB</p>"+
		"<p>SERVER TIME: %s</p>"+
		"<hr style='border:1px solid #333'>"+
		"<p>เพราะความสำเร็จคือรอยยิ้มของทีมงาน ThitNueaHub</p>"+
		"</body></html>", m.Alloc/1024/1024, time.Now().Format(time.RFC822))
}
