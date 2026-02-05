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
	TargetRAMNudge = 400 // MB
	RequestTimeout = 60  // Seconds
)

// --- [🚀 CORE LOGIC: DARK-RELAY & GAS STATION] ---

func logIdentity() {
	fmt.Println("--- 🛡️ Protocol: Dark-Relay Fusion (Virginia Stable) ---")
	fmt.Println("⛑️ Status: Snake Nudge Memory Guard Active")
	fmt.Println("⛽ Strategy: Gas Station (08:00, 12:00, 20:00)")
}

func ThitNueaMonitor() {
	var m runtime.MemStats
	for {
		runtime.ReadMemStats(&m)
		allocMB := m.Alloc / 1024 / 1024
		if allocMB > TargetRAMNudge {
			runtime.GC()
		}
		time.Sleep(10 * time.Second)
	}
}

func runScheduler() {
	logIdentity()
	go ThitNueaMonitor()
	
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

		fmt.Printf("\n😴 [Gas Station]: Next Run at %s\n", nextRun.Format("15:04:05"))
		time.Sleep(time.Until(nextRun))

		ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout*time.Second)
		darkRelayExecution(ctx)
		cancel()
	}
}

func darkRelayExecution(ctx context.Context) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" { return fmt.Errorf("API KEY MISSING") }

	prompt := "Generate an elite tech insight for ThitNueaHub. Theme: Security & Scalability. Tone: Aggressive. Response in Thai."
	
	_, err := callGeminiDarkRelay(ctx, apiKey, prompt)
	if err != nil {
		log.Printf("❌ Snake Nudge Recall Triggered: %v", err)
		return err
	}
	fmt.Println("✅ [RELAY] Content Delivered Successfully.")
	return nil
}

func callGeminiDarkRelay(ctx context.Context, apiKey, prompt string) (string, error) {
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]string{{"text": prompt}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "Strictly Professional, Zero-Garbage, Elite Thai Language."}},
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil { return "", err }
	defer resp.Body.Close()

	var result struct {
		Candidates []struct {
			Content struct {
				Parts []struct { Text string `json:"text"` } `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	
	json.NewDecoder(resp.Body).Decode(&result)
	if len(result.Candidates) > 0 {
		return result.Candidates[0].Content.Parts[0].Text, nil
	}
	return "", fmt.Errorf("AI ERROR")
}

// --- [🎨 UI: THE COCKPIT DASHBOARD] ---

func main() {
	go runScheduler()

	port := os.Getenv("PORT")
	if port == "" { port = "10000" }

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
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
        @keyframes scan { 0%% { top: 0; } 100%% { top: 100%%; } }
        .scanner { animation: scan 3s linear infinite; background: linear-gradient(to bottom, transparent, rgba(204, 255, 0, 0.2), transparent); }
        .text-glow { text-shadow: 0 0 10px #ccff00; }
        @keyframes nudge-flash { 0%% { background: rgba(204, 255, 0, 0); } 50%% { background: rgba(204, 255, 0, 0.2); } 100%% { background: rgba(204, 255, 0, 0); } }
        .nudge-active { animation: nudge-flash 0.5s ease-in-out; }
    </style>
    <title>THITNUEA HUB | COCKPIT</title>
</head>
<body class="bg-[#050505] text-[#ccff00] font-mono overflow-hidden">
    <div class="fixed inset-0 pointer-events-none z-50">
        <div class="scanner absolute w-full h-20 opacity-20"></div>
    </div>
    <div class="h-screen flex flex-col p-4 md:p-8">
        <div class="flex justify-between items-start border-b border-[#ccff00]/30 pb-4 mb-6">
            <div>
                <h1 class="text-4xl font-black tracking-tighter italic text-glow uppercase">ThitNuea Hub</h1>
                <p class="text-[10px] tracking-[0.3em] opacity-50 uppercase">Virginia Station // Dark-Relay v3.2</p>
            </div>
            <div class="text-right">
                <div class="inline-block px-3 py-1 border border-[#ccff00] text-[10px] font-bold">MODE: 100%% AUTO</div>
                <p class="text-[10px] mt-1 opacity-50">SYSTEM TIME: <span id="clock">--:--:--</span></p>
            </div>
        </div>
        <div class="grid grid-cols-1 md:grid-cols-3 gap-6 flex-grow">
            <div class="border border-[#ccff00]/20 p-6 flex flex-col justify-between">
                <h3 class="text-xs font-bold mb-4 opacity-70 border-l-2 border-[#ccff00] pl-2">SYSTEM HEALTH</h3>
                <div class="space-y-6">
                    <div>
                        <div class="flex justify-between text-[10px] mb-1"><span>RAM USAGE</span><span>%dMB / 512MB</span></div>
                        <div class="h-1.5 bg-gray-900 rounded-full overflow-hidden border border-[#ccff00]/10">
                            <div class="bg-[#ccff00] h-full shadow-[0_0_10px_#ccff00]" style="width: %d%%"></div>
                        </div>
                    </div>
                </div>
                <div class="mt-4 p-2 bg-[#ccff00]/5 text-[9px] border-l-2 border-[#ccff00] animate-pulse uppercase">
                    ⛑️ Snake Nudge: Active
                </div>
            </div>
            <div class="md:col-span-2 border border-[#ccff00]/20 p-6 bg-gradient-to-br from-black to-[#0a0a0a]">
                <h3 class="text-xs font-bold mb-4 opacity-70 border-l-2 border-[#ccff00] pl-2 uppercase">Action Feed</h3>
                <div id="log-container" class="space-y-2 text-[11px] overflow-y-auto h-48 md:h-64">
                    <p class="opacity-30">>> Waiting for Gas Station (20:00:00)...</p>
                    <p class="text-white">>> [SYSTEM] Engine Live at Virginia Station Port %s</p>
                </div>
                <button onclick="triggerNudge()" class="mt-6 w-full py-4 border border-[#ccff00] hover:bg-[#ccff00] hover:text-black transition-all font-black text-sm uppercase tracking-widest">
                    FORCE RELAY EXECUTION (เคาะเลย)
                </button>
            </div>
        </div>
        <div class="mt-6 pt-4 border-t border-[#ccff00]/10 flex justify-between text-[9px] opacity-40">
            <p>© 2026 ARTHIT | THITNUEA HUB</p>
            <p>ZERO-GARBAGE ENGINE | VIRGINIA REGION</p>
        </div>
    </div>
    <script>
        function triggerNudge() {
            const log = document.getElementById('log-container');
            const p = document.createElement('p');
            p.className = "text-[#ccff00] font-bold nudge-active";
            p.innerText = ">> [" + new Date().toLocaleTimeString() + "] 🏍️ FORCE NUDGE: Manual Relay Triggered!";
            log.prepend(p);
            fetch('/nudge'); // ยิงไปฝั่ง Server
        }
        setInterval(() => { document.getElementById('clock').innerText = new Date().toLocaleTimeString(); }, 1000);
    </script>
</body>
</html>
		`, ramUse, (ramUse * 100 / 512), port)
	})

	// แถม Endpoint สำหรับปุ่ม "เคาะ" (Manual Nudge)
	http.HandleFunc("/nudge", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), RequestTimeout*time.Second)
		defer cancel()
		go darkRelayExecution(ctx)
		fmt.Fprintf(w, "Nudged")
	})

	fmt.Printf("🚪 Virginia Station Port: %s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
