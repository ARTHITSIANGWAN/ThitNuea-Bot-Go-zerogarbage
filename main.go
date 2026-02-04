package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

// --- [🛡️ IDENTITY: THITNUEA HUB DARK-RELAY] ---
func logIdentity() {
	fmt.Println("--- 🛡️ Protocol: Dark-Relay Fusion (Final Build v3.2) ---")
	fmt.Println("⛑️ Agents: Sprinter | Strategist | Refiner | Finisher")
	fmt.Println("⛽ Logic: Gas Station Scheduler (TH Timezone Active)")
	fmt.Println("🐍 System: Snake Nudge Error Recall Ready")
}

// runScheduler: ระบบ Gas Station ตั้งเวลาพ่นคอนเทนต์ (08:00, 12:00, 20:00)
func runScheduler() {
	logIdentity()
	targetHours := []int{8, 12, 20}

	for {
		// ตั้งค่า Timezone เป็นไทย (GMT+7)
		loc := time.FixedZone("Asia/Bangkok", 7*60*60)
		now := time.Now().In(loc)
		
		nextRun := time.Time{}
		found := false

		for _, hour := range targetHours {
			targetTime := time.Date(now.Year(), now.Month(), now.Day(), hour, 0, 0, 0, loc)
			if targetTime.After(now) {
				nextRun = targetTime
				found = true
				break
			}
		}

		if !found {
			nextRun = time.Date(now.Year(), now.Month(), now.Day()+1, targetHours[0], 0, 0, 0, loc)
		}

		fmt.Printf("😴 [Gas Station]: พักเครื่อง.. รอบถัดไปคือ %s (รออีก %v)\n", 
			nextRun.Format("15:04:05"), time.Until(nextRun).Round(time.Second))

		time.Sleep(time.Until(nextRun))

		// --- [🏁 START 4x100 RELAY SEQUENCE] ---
		fmt.Printf("⏰ [%s] 🏁 Gas Station ปล่อยตัว: Stage 1 Initiating...\n", time.Now().In(loc).Format("15:04:05"))
		err := darkRelayExecution()
		if err != nil {
			log.Printf("❌ Snake Nudge Recall triggered: %v", err)
			time.Sleep(5 * time.Minute) // พัก 5 นาทีแล้วลองใหม่ (Retry)
			darkRelayExecution()
		} else {
			fmt.Println("✅ Stage 4: Finisher Delivered Successfully.")
		}
	}
}

func darkRelayExecution() error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	tgToken := os.Getenv("TELEGRAM_TOKEN")
	chatID := os.Getenv("CHAT_ID")
	if apiKey == "" { return fmt.Errorf("secure credentials missing") }

	prompt := "Task: Generate an elite tech insight for ThitNueaHub. Style: Aggressive, Professional, Zero-Garbage."
	
	rawOutput, err := callGeminiDarkRelay(apiKey, prompt)
	if err != nil { return err }

	if tgToken != "" && chatID != "" {
		finalReport := fmt.Sprintf("🛡️ **DARK-RELAY FUSION REPORT**\n\n%s\n\n#ThitNueaHub #DarkRelay #Gemini3Flash", rawOutput)
		sendTelegram(tgToken, chatID, finalReport)
	}
	return nil
}

func callGeminiDarkRelay(apiKey, prompt string) (string, error) {
	// ใช้ Gemini 1.5 Flash เป็นฐานที่เสถียรที่สุดสำหรับ Free Tier
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": "You are the Dark-Relay Finisher. STRICT POLICY: Zero-Garbage, No conversational fillers, Elite professional output ONLY. Response in Thai."},
			},
		},
		"safetySettings": []map[string]interface{}{
			{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "BLOCK_NONE"},
			{"category": "HARM_CATEGORY_HATE_SPEECH", "threshold": "BLOCK_NONE"},
			{"category": "HARM_CATEGORY_DANGEROUS_CONTENT", "threshold": "BLOCK_NONE"},
			{"category": "HARM_CATEGORY_SEXUALLY_EXPLICIT", "threshold": "BLOCK_NONE"},
		},
	})

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(payload))
	if err != nil { return "", err }
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API Error %d: %s", resp.StatusCode, string(body))
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	candidates := result["candidates"].([]interface{})
	if len(candidates) == 0 { return "", fmt.Errorf("AI Refused") }
	content := candidates[0].(map[string]interface{})["content"].(map[string]interface{})
	parts := content["parts"].([]interface{})
	return parts[0].(map[string]interface{})["text"].(string), nil
}

func sendTelegram(token, chatID, text string) {
	url := fmt.Sprintf("https://api.telegram.org/bot%s/sendMessage", token)
	payload, _ := json.Marshal(map[string]string{"chat_id": chatID, "text": text, "parse_mode": "Markdown"})
	http.Post(url, "application/json", bytes.NewBuffer(payload))
}

func main() {
	go runScheduler()

	port := os.Getenv("PORT")
	if port == "" { port = "8080" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🛡️ ThitNuea Hub: Dark-Relay Protocol ONLINE ✅\nStatus: Gas Station Active (Asia/Bangkok Time)")
	})

	fmt.Printf("🚪 Virginia Station Port: %s\n", port)
	http.ListenAndServe(":"+port, nil)
}
