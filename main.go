package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"time"
)

// --- [🛡️ THITNUEA CONSTANTS] ---
const (
	TargetRAMNudge = 400 // MB (จุดเริ่มสะกิด)
	RequestTimeout = 45  // Seconds
)

func logIdentity() {
	fmt.Println("--- 🛡️ Protocol: Dark-Relay Fusion (Virginia Stable) ---")
	fmt.Println("⛑️ Status: Snake Nudge Memory Guard Active")
	fmt.Println("⛽ Strategy: Gas Station (08:00, 12:00, 20:00)")
}

// ThitNueaMonitor: คอยกวาดขยะและรายงาน RAM ตลอดการทำงาน
func ThitNueaMonitor() {
	var m runtime.MemStats
	for {
		runtime.ReadMemStats(&m)
		allocMB := m.Alloc / 1024 / 1024
		if allocMB > TargetRAMNudge {
			fmt.Printf("\n⚠️ [Nudge] RAM %dMB: Force GC Initiated...", allocMB)
			runtime.GC()
		}
		time.Sleep(10 * time.Second)
	}
}

func runScheduler() {
	logIdentity()
	go ThitNueaMonitor() // สตาร์ทตัวตรวจจับ RAM ใน Background
	
	targetHours := []int{8, 12, 20}
	loc := time.FixedZone("Asia/Bangkok", 7*60*60)

	for {
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

		fmt.Printf("\n😴 [Gas Station]: พักเครื่อง.. รอบถัดไปคือ %s\n", nextRun.Format("15:04:05"))
		time.Sleep(time.Until(nextRun))

		// --- [🏁 START RELAY] ---
		fmt.Printf("⏰ [%s] 🏁 Start Action!\n", time.Now().In(loc).Format("15:04:05"))
		
		// ป้องกันการค้างด้วย Context Timeout
		ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout*time.Second)
		err := darkRelayExecution(ctx)
		cancel()

		if err != nil {
			log.Printf("❌ Snake Nudge Recall: %v", err)
			time.Sleep(5 * time.Minute) 
		} else {
			fmt.Println("✅ Sequence Success.")
		}
	}
}

func darkRelayExecution(ctx context.Context) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" { return fmt.Errorf("secure credentials missing") }

	prompt := "Task: Elite tech insight for ThitNueaHub. Theme: Security & Scalability. Tone: Aggressive."
	
	// ใช้ Decoder เพื่อประหยัด RAM แทน ReadAll
	rawOutput, err := callGeminiDarkRelay(ctx, apiKey, prompt)
	if err != nil { return err }

	// ส่ง Telegram (ใส่ Logic Telegram เดิมของเจ้านายได้เลย)
	fmt.Println("📡 AI Response Received: ", len(rawOutput), " characters.")
	return nil
}

func callGeminiDarkRelay(ctx context.Context, apiKey, prompt string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]string{{"text": prompt}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "Zero-Garbage, No fillers, Thai Language."}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	if resp.StatusCode != 200 { return "", fmt.Errorf("API Error %d", resp.StatusCode) }

	// ใช้ JSON Decoder เพื่อดึงข้อมูลแบบ Stream (เซฟ RAM 512MB สุดๆ)
	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Candidates) > 0 && len(result.Candidates[0].Content.Parts) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("AI Refused")
}

func main() {
	go runScheduler()
	
	// รัน Web Server เบาๆ ไว้เช็กสถานะที่ Virginia
	port := os.Getenv("PORT")
	if port == "" { port = "8080" }
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "🛡️ ThitNuea Hub: Dark-Relay Online ✅")
	})
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
