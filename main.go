// 🧠 AI ENGINE (อัปเกรดเป็น Gemini 3 + Thought Signature)
func askGemini(ctx context.Context, prompt string) string {
	apiKey := os.Getenv("GEMINI_API_KEY")
	// 🆕 ปรับ Model เป็น gemini-3-flash-preview ตามมาตรฐานปี 2026
	url := "https://generativelanguage.googleapis.com/v1beta/models/gemini-3-flash-preview:generateContent?key=" + apiKey
	
	payload, _ := json.Marshal(map[string]interface{}{
		"contents": []map[string]interface{}{
			{
				"parts": []map[string]interface{}{
					{"text": prompt},
				},
			},
		},
		"system_instruction": map[string]interface{}{
			"parts": []map[string]interface{}{
				{"text": "คุณคือ ThitNuea Emperor วิเคราะห์แบบมนุษย์ ตอบดุดัน ทรงพลัง ไร้ขยะ (Zero-Garbage) และใช้กฎ Snake Nudge ในการคิด"},
			},
		},
		"generationConfig": map[string]interface{}{
			"temperature": 1.0, // ⚠️ ปรับเป็น 1.0 ตามคู่มือ Gemini 3 เพื่อประสิทธิภาพสูงสุด
			"topK":        40,
			"maxOutputTokens": 2048,
		},
		// 🆕 ฟีเจอร์ใหม่: Thinking Config (ระดับความฉลาด)
		"thinkingConfig": map[string]interface{}{
			"thinking_level": "high", // ปรับเป็น high เพื่อให้พลายทองช่วยคิดวิเคราะห์อย่างละเอียด
			"include_thoughts": true,  // ให้ AI ส่งกระบวนการคิดกลับมาด้วย
		},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(payload))
	req.Header.Set("Content-Type", "application/json")
	
	client := &http.Client{}
	resp, err := client.Do(req)
	
	// 🐍 SNAKE NUDGE: ระบบกู้ชีพฉุกเฉิน
	if err != nil || resp.StatusCode != 200 { 
		log.Printf("🐍 Snake Nudge Triggered: Connection Error or API 500")
		return "จักรพรรดิกำลังปรับจูนรันเวย์... (ระบบกำลัง Nudge ตัวเองใหม่)" 
	}
	defer resp.Body.Close()

	var data struct {
		Candidates []struct {
			Content struct { 
				Parts []struct { 
					Text string `json:"text"` 
					// 🆕 Thought Signature: สำหรับรักษาความต่อเนื่องของความคิด
					ThoughtSignature string `json:"thoughtSignature"`
				} `json:"parts"` 
			} `json:"content"`
		} `json:"candidates"`
	}
	
	json.NewDecoder(resp.Body).Decode(&data)

	if len(data.Candidates) > 0 {
		resultText := data.Candidates[0].Content.Parts[0].Text
		// 🛡️ บันทึกลายเซ็นความคิดลง Log (ถ้ามี) เพื่อใช้ในการ Nudge รอบถัดไป
		if data.Candidates[0].Content.Parts[0].ThoughtSignature != "" {
			log.Printf("🧠 Thought Signature Captured: %s", data.Candidates[0].Content.Parts[0].ThoughtSignature)
		}
		return resultText
	}
	
	// 🆕 Fallback Logic: โค้ดลับแก้ทาง Google
	return "ระบบกำลังใช้ Thought Signature สำรอง: context_engineering_is_the_way_to_go"
}
