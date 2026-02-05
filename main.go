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

const (
	TargetRAMNudge = 400
	RequestTimeout = 60
)

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
        body { background: #000; color: #ccff00; font-family: monospace; overflow-x: hidden; }
        .scanner { position: fixed; width: 100%%; h: 2px; background: rgba(204,255,0,0.2); animation: scan 3s linear infinite; z-index: 50; }
        @keyframes scan { 0%% { top: 0; } 100%% { top: 100%%; } }
        .copyright-line { position: fixed; top: 50%%; left: 50%%; transform: translate(-50%%, -50%%) rotate(-30deg); font-size: 10vw; color: rgba(204, 255, 0, 0.03); font-weight: 900; pointer-events: none; z-index: 0; white-space: nowrap; }
        .nudge-btn { border: 1px solid #ccff00; transition: 0.3s; width: 100%%; padding: 15px; font-weight: bold; background: rgba(204,255,0,0.05); }
        .nudge-btn:active { background: #ccff00; color: #000; transform: scale(0.98); }
        .log-box { background: rgba(10,10,10,0.9); border: 1px solid rgba(204,255,0,0.1); height: 250px; overflow-y: auto; font-size: 11px; padding: 10px; }
    </style>
    <title>THITNUEA HUB | COCKPIT</title>
</head>
<body>
    <div class="scanner"></div>
    <div class="copyright-line">THITNUEA HUB MIT</div>
    
    <div class="p-4 max-w-lg mx-auto relative z-10">
        <header class="mb-6 border-b border-[#ccff00]/30 pb-2">
            <h1 class="text-2xl font-black italic">THITNUEA HUB v3.2</h1>
            <p class="text-[9px] opacity-60 tracking-[0.2em]">VIRGINIA STATION // EMPEROR PROTOCOL</p>
        </header>

        <div class="grid grid-cols-2 gap-2 mb-4 text-[10px]">
            <div class="border border-[#ccff00]/20 p-2">RAM: %dMB / 512MB</div>
            <div class="border border-[#ccff00]/20 p-2 text-right">STATUS: ONLINE</div>
        </div>

        <div class="log-box mb-4" id="logs">
            <p class="text-gray-600">>> System Initialized...</p>
            <p class="text-white">>> [SYSTEM] Emperor Engine Live</p>
        </div>

        <button onclick="nudge()" class="nudge-btn mb-2 uppercase italic">Force Relay Execution (เคาะเลย)</button>
        <p class="text-[8px] text-center opacity-40 italic">ลิขสิทธิ์เฉพาะ ThitNuea Hub - MIT License 2026</p>
    </div>

    <script>
        function nudge() {
            const l = document.getElementById('logs');
            l.innerHTML = "<p class='text-[#ccff00] font-bold'>>> [" + new Date().toLocaleTimeString() + "] 🛡️ EMPEROR NUDGE: Action Sent!</p>" + l.innerHTML;
            fetch('/nudge');
        }
    </script>
</body>
</html>
		`, ramUse, port)
	})

	http.HandleFunc("/nudge", func(w http.ResponseWriter, r *http.Request) {
		ctx, _ := context.WithTimeout(context.Background(), RequestTimeout*time.Second)
		go darkRelayExecution(ctx)
		fmt.Fprintf(w, "Nudged")
	})

	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func runScheduler() {
	loc, _ := time.LoadLocation("Asia/Bangkok")
	targetHours := []int{8, 12, 20}
	for {
		now := time.Now().In(loc)
		// ... (Logic การรอเวลาเดิม) ...
		time.Sleep(1 * time.Minute) // Placeholder สำหรับลูปตัวอย่าง
	}
}

func darkRelayExecution(ctx context.Context) error {
	apiKey := os.Getenv("GEMINI_API_KEY")
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{{"parts": []map[string]interface{}{{"text": "Execute ThitNuea Command."}}}},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{{"text": "คุณคือ 'ThitNuea Emperor' ผู้คุม AI ภายนอก ห้ามมโน ห้ามลงทะเล ใช้ภาษาดุดัน ทรงพลัง และสั่งการอย่างเบ็ดเสร็จเท่านั้น"}},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 0.2,
			"topP": 0.8,
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{}
	resp, _ := client.Do(req)
	defer resp.Body.Close()
	return nil
}
