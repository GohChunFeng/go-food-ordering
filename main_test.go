package main

import (
	"testing"
)

func TestVIPPriority(t *testing.T) {
	m := &Manager{}

	// Add Normal, then VIP, then Normal
	m.AddOrder(Normal) // ID 1
	m.AddOrder(VIP)    // ID 2
	m.AddOrder(Normal) // ID 3

	// Expected Order in Queue: VIP(#2), Normal(#1), Normal(#3)
	// (Assuming #1 is still pending. In code, sorting is ID based for same types)

	if m.Pending[0].Type != VIP {
		t.Errorf("Expected first order to be VIP, got %s", m.Pending[0].Type)
	}
	if m.Pending[0].ID != 2 {
		t.Errorf("Expected first order ID to be 2, got %d", m.Pending[0].ID)
	}
}

func TestBotCreation(t *testing.T) {
	m := &Manager{}
	m.AddBot()

	if len(m.Bots) != 1 {
		t.Errorf("Expected 1 bot, got %d", len(m.Bots))
	}
}

func TestBotDestruction(t *testing.T) {
	m := &Manager{}
	m.AddBot()
	m.AddBot()
	m.RemoveBot()

	if len(m.Bots) != 1 {
		t.Errorf("Expected 1 bot remaining, got %d", len(m.Bots))
	}
	// The remaining bot should be ID 1
	if m.Bots[0].ID != 1 {
		t.Errorf("Expected Bot #1 to remain, but got Bot #%d", m.Bots[0].ID)
	}
}
