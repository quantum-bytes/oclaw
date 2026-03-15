package gateway

import (
	"context"
	"fmt"
	"encoding/json"
	"testing"
	"time"
)

func TestClientConnectLive(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping live gateway test")
	}

	client := NewClient("ws://127.0.0.1:39421", "ollama")

	connectCalled := make(chan struct{}, 1)
	client.OnConnect(func() {
		t.Log("OnConnect callback fired")
		select {
		case connectCalled <- struct{}{}:
		default:
		}
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		err := client.Connect(ctx)
		t.Logf("Connect returned: %v", err)
		errCh <- err
	}()

	// Wait for connection
	select {
	case <-connectCalled:
		t.Log("connected via callback")
	case err := <-errCh:
		t.Fatalf("connect returned early: %v", err)
	case <-time.After(5 * time.Second):
		// Also check Connected()
		t.Logf("Connected() = %v", client.Connected())
		t.Fatal("timed out waiting for connection")
	}

	if !client.Connected() {
		t.Fatal("Connected() should be true after OnConnect callback")
	}

	// Test agents.list
	agents, err := client.ListAgents()
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("expected at least one agent")
	}
	for _, a := range agents {
		t.Logf("agent: %s (%s) model=%s", a.Name, a.ID, a.Model)
	}

	// Test sessions.list
	sessions, err := client.ListSessions(SessionsListParams{
		Limit:   5,
		AgentID: "aura",
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	t.Logf("sessions: %d", len(sessions))
	for _, s := range sessions {
		t.Logf("  session: %s title=%q", s.Key, s.Title)
	}

	// Test chat.history
	history, err := client.LoadHistory("agent:aura:main", 5)
	if err != nil {
		t.Logf("load history: %v (may be empty)", err)
	} else {
		t.Logf("history messages: %d", len(history))
	}

	// Test a chat message
	_, err = client.SendChat("agent:aura:main", "respond with just the word 'pong'", "", 30000)
	if err != nil {
		t.Fatalf("send chat: %v", err)
	}
	t.Log("chat sent, waiting for events...")

	// Drain events for response
	deadline := time.After(30 * time.Second)
	for {
		select {
		case evt := <-client.Events():
			if evt.Event == "chat" {
				t.Logf("chat event: %s", string(evt.Payload)[:minInt(200, len(evt.Payload))])
				// Check for final state
				var tmp struct {
					State string `json:"state"`
				}
				if err := json.Unmarshal(evt.Payload, &tmp); err == nil && (tmp.State == "final" || tmp.State == "error") {
					t.Logf("got final/error state: %s", tmp.State)
					goto done
				}
			} else {
				t.Logf("event: %s", evt.Event)
			}
		case <-deadline:
			t.Log("timed out waiting for chat response (OK for this test)")
			goto done
		}
	}
done:
	client.Close()
	cancel()
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestConnectParamsSerialization(t *testing.T) {
	params := ConnectParams{
		MinProtocol: 3,
		MaxProtocol: 3,
		Client: ClientInfo{
			ID:       "cli",
			Version:  "0.1.0",
			Platform: "darwin",
			Mode:     "cli",
		},
		Auth: map[string]interface{}{
			"token": "test",
		},
		Role:   "operator",
		Scopes: []string{"operator.admin"},
	}

	data, err := json.Marshal(params)
	if err != nil {
		t.Fatal(err)
	}

	s := string(data)
	if !contains(s, `"minProtocol":3`) {
		t.Errorf("missing minProtocol: %s", s)
	}
	if !contains(s, `"id":"cli"`) {
		t.Errorf("missing client.id: %s", s)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func init() {
	// Suppress unused import warning
	_ = fmt.Sprint
}
