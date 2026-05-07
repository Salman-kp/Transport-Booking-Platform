package bot

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
)

// SessionState holds the current booking flow data for a user
type SessionState struct {
	Step       string
	Type       string // "flight" or "bus"
	Origin     string
	Dest       string
	Date       string
	Passengers string
}

// UserSessions stores active chats (In production, replace this with Redis!)
var UserSessions sync.Map

type MetaWebhook struct {
	Object string      `json:"object"`
	Entry  []MetaEntry `json:"entry"`
}

type MetaEntry struct {
	ID      string       `json:"id"`
	Changes []MetaChange `json:"changes"`
}

type MetaChange struct {
	Value MetaValue `json:"value"`
	Field string    `json:"field"`
}

type MetaValue struct {
	MessagingProduct string        `json:"messaging_product"`
	Metadata         MetaMetadata  `json:"metadata"`
	Contacts         []MetaContact `json:"contacts"`
	Messages         []MetaMessage `json:"messages"`
}

type MetaMetadata struct {
	DisplayPhoneNumber string `json:"display_phone_number"`
	PhoneNumberID      string `json:"phone_number_id"`
}

type MetaContact struct {
	Profile MetaProfile `json:"profile"`
	WaID    string      `json:"wa_id"`
}

type MetaProfile struct {
	Name string `json:"name"`
}

type MetaMessage struct {
	From      string   `json:"from"`
	ID        string   `json:"id"`
	Timestamp string   `json:"timestamp"`
	Type      string   `json:"type"`
	Text      MetaText `json:"text"`
}

type MetaText struct {
	Body string `json:"body"`
}


// HandleMetaWebhookVerification handles the GET request from Meta to verify the webhook URL
func HandleMetaWebhookVerification(c fiber.Ctx) error {
	// The token you configure in the Meta dashboard
	verifyToken := "my_super_secret_verify_token"

	mode := c.Query("hub.mode")
	token := c.Query("hub.verify_token")
	challenge := c.Query("hub.challenge")

	if mode == "subscribe" && token == verifyToken {
		fmt.Println("Webhook verified successfully!")
		// Meta expects the raw challenge string back
		return c.SendString(challenge)
	}

	return c.Status(fiber.StatusForbidden).SendString("Verification failed")
}

// HandleMetaMessage receives the incoming POST message from WhatsApp
func HandleMetaMessage(c fiber.Ctx) error {
	// Log raw body to debug what Meta is actually sending
	fmt.Printf("=== RAW META PAYLOAD ===\n%s\n========================\n", string(c.Body()))

	var body MetaWebhook
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).SendString("Bad Request")
	}

	if body.Object == "whatsapp_business_account" {
		for _, entry := range body.Entry {
			for _, change := range entry.Changes {
				// Ensure there is actually a message (Meta sometimes sends status updates like "delivered")
				if len(change.Value.Messages) > 0 {
					msg := change.Value.Messages[0]
					
						// We only care about text messages for now
					if msg.Type == "text" {
						senderPhone := msg.From
						messageText := strings.TrimSpace(msg.Text.Body)

						// 1. Get or create the user's session
						val, _ := UserSessions.LoadOrStore(senderPhone, &SessionState{Step: "STATE_IDLE"})
						session := val.(*SessionState)

						// 2. Process their message through the State Machine
						handleConversation(senderPhone, messageText, session)
					}
				}
			}
		}
	}

	// Always return 200 OK quickly, otherwise Meta will retry sending the message
	return c.Status(fiber.StatusOK).SendString("EVENT_RECEIVED")
}

// SendWhatsAppMessage sends a text message back to the user via Meta Graph API
func SendWhatsAppMessage(toPhone string, message string) error {
	accessToken := strings.TrimSpace(os.Getenv("META_ACCESS_TOKEN"))
	phoneID := strings.TrimSpace(os.Getenv("META_PHONE_ID"))

	fmt.Printf("DEBUG - PhoneID: '%s', TokenLength: %d\n", phoneID, len(accessToken))

	url := fmt.Sprintf("https://graph.facebook.com/v25.0/%s/messages", phoneID)

	payload := map[string]interface{}{
		"messaging_product": "whatsapp",
		"to":                toPhone,
		"type":              "text",
		"text": map[string]string{
			"body": message,
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Read the body to see exactly why Meta rejected it
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		return fmt.Errorf("status code %d, error details: %s", resp.StatusCode, buf.String())
	}

	return nil
}

// handleConversation walks the user through the booking flow step-by-step
func handleConversation(phone, text string, session *SessionState) {
	lowerText := strings.ToLower(text)

	// Allow the user to cancel at any time
	if lowerText == "cancel" || lowerText == "reset" {
		session.Step = "STATE_IDLE"
		SendWhatsAppMessage(phone, "Session reset. Say 'Hi' whenever you are ready to book again!")
		return
	}

	switch session.Step {
	case "STATE_IDLE":
		reply := "👋 Welcome to *Tripneo*!\n\nWhat would you like to book today?\nReply with *1* for Flight ✈️\nReply with *2* for Bus 🚌"
		SendWhatsAppMessage(phone, reply)
		session.Step = "STATE_ASK_TYPE"

	case "STATE_ASK_TYPE":
		if lowerText == "1" || lowerText == "flight" {
			session.Type = "flight"
		} else if lowerText == "2" || lowerText == "bus" {
			session.Type = "bus"
		} else {
			SendWhatsAppMessage(phone, "Please reply with *1* for Flight or *2* for Bus.")
			return
		}
		SendWhatsAppMessage(phone, "Awesome! Where are you departing from? (e.g., NYC, London)")
		session.Step = "STATE_ASK_ORIGIN"

	case "STATE_ASK_ORIGIN":
		// Capitalize the city name (e.g., "new delhi" -> "New Delhi")
		session.Origin = strings.Title(strings.ToLower(text))
		SendWhatsAppMessage(phone, "Got it! Where are you traveling to?")
		session.Step = "STATE_ASK_DEST"

	case "STATE_ASK_DEST":
		session.Dest = strings.Title(strings.ToLower(text))
		
		// Calculate dynamic dates
		tomorrow := time.Now().Add(24 * time.Hour).Format("2006-01-02")
		dayAfter := time.Now().Add(48 * time.Hour).Format("2006-01-02")
		
		reply := fmt.Sprintf("When do you want to travel?\n\nReply with *1* for Tomorrow (%s)\nReply with *2* for Day After (%s)\n\nOr simply type any date (e.g., 2026-10-25)", tomorrow, dayAfter)
		SendWhatsAppMessage(phone, reply)
		session.Step = "STATE_ASK_DATE"

	case "STATE_ASK_DATE":
		if text == "1" {
			session.Date = time.Now().Add(24 * time.Hour).Format("2006-01-02")
		} else if text == "2" {
			session.Date = time.Now().Add(48 * time.Hour).Format("2006-01-02")
		} else {
			// They typed a custom date
			session.Date = text
		}
		
		SendWhatsAppMessage(phone, "Almost done! How many passengers?")
		session.Step = "STATE_ASK_PASSENGERS"

	case "STATE_ASK_PASSENGERS":
		session.Passengers = text
		
		originCode := getCityCode(session.Origin)
		destCode := getCityCode(session.Dest)
		dateQuery := url.QueryEscape(session.Date)
		paxQuery := url.QueryEscape(session.Passengers)
		
		var checkoutURL string
		if session.Type == "flight" {
			// Real Flight URL Format
			checkoutURL = fmt.Sprintf("https://tripneo.in/flights/results?origin=%s&destination=%s&adults=%s&children=0&infants=0&class=ECONOMY&date=%s", 
				originCode, destCode, paxQuery, dateQuery)
		} else {
			// Real Bus URL Format (Assuming a standard format, you can adjust if needed)
			checkoutURL = fmt.Sprintf("https://tripneo.in/buses/results?origin=%s&destination=%s&date=%s&passengers=%s", 
				url.QueryEscape(session.Origin), url.QueryEscape(session.Dest), dateQuery, paxQuery)
		}
		
		reply := fmt.Sprintf("✅ *Search Complete!*\n\nLooking for %s tickets:\n📍 From: %s (%s)\n📍 To: %s (%s)\n📅 Date: %s\n🧑‍🤝‍🧑 Passengers: %s\n\nTap the secure link below to see live prices and book your tickets:\n%s",
			session.Type, session.Origin, originCode, session.Dest, destCode, session.Date, session.Passengers, checkoutURL)
		
		SendWhatsAppMessage(phone, reply)
		
		// Reset the session so they can book again!
		session.Step = "STATE_IDLE"
	}
}

// getCityCode maps full Indian city names to their 3-letter IATA codes
func getCityCode(city string) string {
	lowerCity := strings.ToLower(strings.TrimSpace(city))
	codes := map[string]string{
		"delhi":     "DEL",
		"new delhi": "DEL",
		"mumbai":    "BOM",
		"bombay":    "BOM",
		"bangalore": "BLR",
		"bengaluru": "BLR",
		"chennai":   "MAA",
		"kolkata":   "CCU",
		"hyderabad": "HYD",
		"pune":      "PNQ",
		"goa":       "GOI",
	}

	if code, exists := codes[lowerCity]; exists {
		return code
	}
	
	// Fallback: Just uppercase the first 3 letters if not found
	if len(lowerCity) >= 3 {
		return strings.ToUpper(lowerCity[:3])
	}
	return strings.ToUpper(lowerCity)
}
