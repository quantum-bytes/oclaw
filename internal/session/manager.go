package session

import "fmt"

// BuildKey constructs a session key for the given agent.
// Format: agent:<agentId>:<mainKey>
func BuildKey(agentID, mainKey string) string {
	if mainKey == "" {
		mainKey = "main"
	}
	return fmt.Sprintf("agent:%s:%s", agentID, mainKey)
}

// ParseKey extracts the agent ID and main key from a session key.
func ParseKey(key string) (agentID, mainKey string, ok bool) {
	// Expected format: agent:<agentId>:<mainKey>
	var prefix string
	n, _ := fmt.Sscanf(key, "%[^:]:%[^:]:%s", &prefix, &agentID, &mainKey)
	if n < 2 {
		return "", "", false
	}
	if mainKey == "" {
		mainKey = "main"
	}
	return agentID, mainKey, true
}
