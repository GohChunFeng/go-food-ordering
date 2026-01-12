package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"sort"
	"sync"
	"time"
)

// --- CONFIGURATION ---
const ProcessTime = 10 * time.Second

// --- MODELS ---
type OrderType string

const (
	Normal OrderType = "Normal"
	VIP    OrderType = "VIP"
)

type Order struct {
	ID     int
	Type   OrderType
	Status string // PENDING, PROCESSING, COMPLETE
}

type Bot struct {
	ID     int
	Cancel context.CancelFunc // The "Remote Detonator" to kill the bot
}

// --- STORE (The Manager) ---
type Manager struct {
	mu           sync.Mutex
	Pending      []*Order
	Completed    []*Order
	Bots         []*Bot
	OrderIdCount int
	BotIdCount   int
	LogFile      *os.File
}

// --- LOGGING HELPER ---
// writeLog prints to both console and result.txt with timestamp
func (m *Manager) writeLog(format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	// Print to console
	fmt.Println(msg)
	// Print to file (log package handles timestamp automatically)
	if m.LogFile != nil {
		log.Println(msg)
	}
}

// --- CORE LOGIC ---

func (m *Manager) AddOrder(oType OrderType) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.OrderIdCount++
	order := &Order{ID: m.OrderIdCount, Type: oType, Status: "PENDING"}
	m.Pending = append(m.Pending, order)

	// SORTING LOGIC: VIP first, then ID (FIFO)
	sort.SliceStable(m.Pending, func(i, j int) bool {
		if m.Pending[i].Type == VIP && m.Pending[j].Type == Normal {
			return true
		}
		if m.Pending[i].Type == Normal && m.Pending[j].Type == VIP {
			return false
		}
		return m.Pending[i].ID < m.Pending[j].ID
	})

	m.writeLog("[ORDER] New %s Order #%d added to PENDING", oType, order.ID)
}

func (m *Manager) AddBot() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.BotIdCount++
	botID := m.BotIdCount

	// Create a Context with Cancel. This allows us to kill the bot later.
	ctx, cancel := context.WithCancel(context.Background())

	bot := &Bot{ID: botID, Cancel: cancel}
	m.Bots = append(m.Bots, bot)

	m.writeLog("[BOT]   Bot #%d Created and IDLE", botID)

	// Launch the bot in a separate thread (Goroutine)
	go m.botLoop(ctx, botID)
}

func (m *Manager) RemoveBot() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if len(m.Bots) == 0 {
		return
	}

	// Remove the newest bot (Last in Last out)
	lastIndex := len(m.Bots) - 1
	bot := m.Bots[lastIndex]
	m.Bots = m.Bots[:lastIndex]

	m.writeLog("[BOT]   Bot #%d Destroyed", bot.ID)

	// This triggers the ctx.Done() channel in the botLoop
	bot.Cancel()
}

// The Worker Process
func (m *Manager) botLoop(ctx context.Context, botID int) {
	for {
		// Check if we have been killed before looking for work
		select {
		case <-ctx.Done():
			return
		default:
		}

		// 1. Try to get an order
		var myOrder *Order

		// Lock to safely check queue
		m.mu.Lock()
		if len(m.Pending) > 0 {
			myOrder = m.Pending[0]
			m.Pending = m.Pending[1:] // Pop from queue
			myOrder.Status = "PROCESSING"
		}
		m.mu.Unlock()

		// 2. If no order, wait a bit and try again (Polling)
		if myOrder == nil {
			time.Sleep(500 * time.Millisecond)
			continue
		}

		// 3. Process the order
		m.writeLog("[BOT]   Bot #%d picked up Order #%d (%s)", botID, myOrder.ID, myOrder.Type)

		// We use a select statement to handle "Working" OR "Getting Destroyed"
		select {
		case <-time.After(ProcessTime):
			// SUCCESS: 10 seconds passed
			m.mu.Lock()
			myOrder.Status = "COMPLETE"
			m.Completed = append(m.Completed, myOrder)
			m.writeLog("[DONE]  Order #%d COMPLETE by Bot #%d", myOrder.ID, botID)
			m.mu.Unlock()

		case <-ctx.Done():
			// INTERRUPTED: Bot destroyed while working
			m.mu.Lock()
			myOrder.Status = "PENDING"
			// Put back at the front (or re-sort, usually re-appending and sorting is safest)
			m.Pending = append(m.Pending, myOrder)
			// Re-apply sorting logic to ensure VIPs stay top
			sort.SliceStable(m.Pending, func(i, j int) bool {
				if m.Pending[i].Type == VIP && m.Pending[j].Type == Normal {
					return true
				}
				if m.Pending[i].Type == Normal && m.Pending[j].Type == VIP {
					return false
				}
				return m.Pending[i].ID < m.Pending[j].ID
			})
			m.writeLog("[WARN]  Bot #%d destroyed mid-process! Order #%d returned to PENDING", botID, myOrder.ID)
			m.mu.Unlock()
			return // Exit goroutine
		}
	}
}

// --- MAIN SIMULATION ---
func main() {
	// 1. Setup Logging
	f, err := os.Create("scripts/result.txt")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// Set log format to include HH:MM:SS
	log.SetOutput(f)
	log.SetFlags(log.Ltime)

	manager := &Manager{LogFile: f}

	fmt.Println("--- STARTING MCDONALDS SIMULATION ---")

	// 2. Scenario: Normal Order
	manager.AddOrder(Normal)

	// 3. Scenario: Add Bot to process it
	manager.AddBot() // Bot 1

	// 4. Scenario: VIP Priority Test
	// Bot 1 is busy (10s). We add orders now.
	time.Sleep(1 * time.Second)
	manager.AddOrder(Normal) // #2
	manager.AddOrder(VIP)    // #3 (Should jump ahead of #2)

	// 5. Scenario: Bot Destruction
	time.Sleep(2 * time.Second)
	manager.AddBot() // Bot 2 (Newest)

	// Bot 2 should pick up #3 (VIP) immediately
	time.Sleep(1 * time.Second)

	// Now remove Bot 2 while it is processing #3
	manager.RemoveBot() // Should return #3 to queue

	// 6. Wait for Bot 1 to finish #1, then it should pick up #3 (VIP), then #2
	// We wait enough time for everything to finish
	time.Sleep(30 * time.Second)

	fmt.Println("--- SIMULATION FINISHED. CHECK result.txt ---")
}
