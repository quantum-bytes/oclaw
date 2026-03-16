package ui

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/quantum-bytes/oclaw/internal/chat"
	"github.com/quantum-bytes/oclaw/internal/config"
	"github.com/quantum-bytes/oclaw/internal/gateway"
)

// View mode for the app.
type viewMode int

const (
	viewChat viewMode = iota
	viewAgentPicker
	viewSessionPicker
	viewHelp
)

// App is the root bubbletea model.
type App struct {
	cfg    *config.Config
	client *gateway.Client
	ctx    context.Context
	cancel context.CancelFunc

	// UI state
	width  int
	height int
	mode   viewMode

	// Chat
	viewport   viewport.Model
	input      textarea.Model
	messages   []chatMessage
	streaming  bool
	assembler  *chat.StreamAssembler
	currentRun string
	showThink  bool

	// Agent/Session
	currentAgent   string
	currentSession string
	agents         []gateway.AgentInfo
	sessions       []gateway.SessionInfo
	pickerCursor   int

	// Connection
	connected      bool
	reconnecting   bool
	statusMsg      string
	spinnerIdx        int
	thinkingMsgIdx    int
	thinkingTicks     int       // counts ticks to rotate message every ~30 ticks (3s)
	lastCtrlC         time.Time // for double ctrl+c to quit
	receivedChatEvent bool      // true if we got a chat delta/final for current run
	lastCompletedRun  string    // runId of last completed chat (dedup finals)
	debugLog          *os.File  // debug event log
}

type chatMessage struct {
	role     string // "user", "assistant", "system"
	text     string
	thinking string
	tools    []string
}

// Bubbletea messages
type gatewayConnectedMsg struct{}
type gatewayDisconnectedMsg struct{}
type gatewayEventMsg struct{ event *gateway.EventFrame }
type agentsLoadedMsg struct{ agents []gateway.AgentInfo }
type historyLoadedMsg struct {
	messages []gateway.HistoryMessage
	fullLoad bool // true = replace all messages, false = append latest response
}
type sessionsLoadedMsg struct{ sessions []gateway.SessionInfo }
type chatSentMsg struct{ err error }
type sessionResetMsg struct{ err error }
type errMsg struct{ err error }
type tickMsg struct{} // periodic UI refresh for spinner

// NewApp creates a new App model.
func NewApp(cfg *config.Config) *App {
	ctx, cancel := context.WithCancel(context.Background())

	ta := textarea.New()
	ta.Placeholder = "Type a message..."
	ta.Prompt = "│ "
	ta.CharLimit = 0
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.Focus()
	ta.ShowLineNumbers = false

	vp := viewport.New(80, 20)

	client := gateway.NewClient(cfg.GatewayURL, cfg.Token)

	app := &App{
		cfg:            cfg,
		client:         client,
		ctx:            ctx,
		cancel:         cancel,
		viewport:       vp,
		input:          ta,
		assembler:      chat.NewStreamAssembler(),
		showThink:      false,
		currentAgent:   strings.ToLower(cfg.AgentID),
		currentSession: fmt.Sprintf("agent:%s:main", strings.ToLower(cfg.AgentID)),
	}

	client.OnConnect(func() {})
	client.OnDisconnect(func() {})

	// Debug log
	if f, err := os.Create("/tmp/oclaw-debug.log"); err == nil {
		app.debugLog = f
	}

	return app
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

var thinkingMessages = []string{
	"Reasoning through the problem...",
	"Analyzing context...",
	"Forming a response...",
	"Processing your request...",
	"Exploring possibilities...",
	"Weighing different approaches...",
	"Connecting the dots...",
	"Synthesizing information...",
	"Crafting a thoughtful reply...",
	"Deep in thought...",
}

func (a *App) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		a.connectGateway(),
		a.tickSpinner(),
	)
}

func (a *App) tickSpinner() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg{}
	})
}

func (a *App) connectGateway() tea.Cmd {
	return func() tea.Msg {
		go func() {
			_ = a.client.Connect(a.ctx)
		}()

		// Poll until connected or timeout
		for i := 0; i < 100; i++ {
			if a.client.Connected() {
				return gatewayConnectedMsg{}
			}
			select {
			case <-a.ctx.Done():
				return gatewayDisconnectedMsg{}
			default:
			}
			// Brief yield
			select {
			case <-a.ctx.Done():
				return gatewayDisconnectedMsg{}
			case <-timeAfter(50):
			}
		}
		return gatewayDisconnectedMsg{}
	}
}

func timeAfter(ms int) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		<-time.After(time.Duration(ms) * time.Millisecond)
		close(ch)
	}()
	return ch
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		SetMarkdownWidth(a.width)
		a.updateLayout()

	case tea.KeyMsg:
		cmd, handled := a.handleKey(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		if handled {
			return a, tea.Batch(cmds...)
		}

	case gatewayConnectedMsg:
		a.connected = true
		a.reconnecting = false
		a.statusMsg = ""
		cmds = append(cmds, a.loadAgents(), a.loadHistory(), a.listenEvents())

	case gatewayDisconnectedMsg:
		a.connected = false
		a.reconnecting = true
		a.statusMsg = "reconnecting..."

	case gatewayEventMsg:
		cmd := a.handleEvent(msg.event)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
		cmds = append(cmds, a.listenEvents())

	case agentsLoadedMsg:
		a.agents = msg.agents

	case historyLoadedMsg:
		if msg.fullLoad {
			// Full history load (initial connect or agent switch)
			a.messages = nil
			for _, m := range msg.messages {
				a.messages = append(a.messages, chatMessage{
					role:     m.Role,
					text:     chat.ExtractText(m.Content),
					thinking: chat.ExtractThinking(m.Content),
					tools:    chat.ExtractToolCalls(m.Content),
				})
			}
		} else {
			// Response fetch — append the latest assistant message
			for _, m := range msg.messages {
				if m.Role == "assistant" {
					a.messages = append(a.messages, chatMessage{
						role:     m.Role,
						text:     chat.ExtractText(m.Content),
						thinking: chat.ExtractThinking(m.Content),
						tools:    chat.ExtractToolCalls(m.Content),
					})
				}
			}
		}
		a.renderChat()
		a.viewport.GotoBottom()

	case sessionsLoadedMsg:
		a.sessions = msg.sessions

	case chatSentMsg:
		if msg.err != nil {
			a.statusMsg = fmt.Sprintf("send error: %v", msg.err)
			a.streaming = false
		}

	case sessionResetMsg:
		if msg.err != nil {
			a.statusMsg = fmt.Sprintf("reset error: %v", msg.err)
		} else {
			a.messages = nil
			a.renderChat()
			a.statusMsg = "session reset"
		}

	case errMsg:
		a.statusMsg = fmt.Sprintf("error: %v", msg.err)

	case tickMsg:
		if a.streaming {
			a.spinnerIdx = (a.spinnerIdx + 1) % len(spinnerFrames)
			a.thinkingTicks++
			if a.thinkingTicks%30 == 0 { // rotate message every 3s
				a.thinkingMsgIdx = (a.thinkingMsgIdx + 1) % len(thinkingMessages)
			}
			a.renderChat()
		}
		cmds = append(cmds, a.tickSpinner())
	}

	// Update textarea and viewport if in chat mode
	if a.mode == viewChat {
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Forward to viewport for scroll (mouse wheel, pgup/pgdn)
		a.viewport, cmd = a.viewport.Update(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}
	}

	return a, tea.Batch(cmds...)
}

// handleKey processes a key event. Returns (cmd, handled).
// When handled is true, the key should NOT be forwarded to the textarea.
func (a *App) handleKey(msg tea.KeyMsg) (tea.Cmd, bool) {
	// Global keys — always handled (not forwarded to textarea)
	switch {
	case msg.String() == "ctrl+d":
		a.cancel()
		a.client.Close()
		return tea.Quit, true

	case msg.String() == "ctrl+c":
		now := time.Now()

		// If streaming, first ctrl+c aborts the stream
		if a.streaming {
			if a.currentRun != "" {
				go a.client.AbortChat(a.currentSession, a.currentRun)
			}
			a.streaming = false
			a.assembler.Reset()
			a.statusMsg = "aborted"
			a.lastCtrlC = now
			return nil, true
		}

		// Double ctrl+c within 1.5s — quit
		if !a.lastCtrlC.IsZero() && now.Sub(a.lastCtrlC) < 1500*time.Millisecond {
			a.cancel()
			a.client.Close()
			return tea.Quit, true
		}

		// First ctrl+c — clear the input box
		if strings.TrimSpace(a.input.Value()) != "" {
			a.input.Reset()
			a.statusMsg = ""
			a.lastCtrlC = now
			return nil, true
		}

		// Input already empty — record and show hint
		a.lastCtrlC = now
		a.statusMsg = "press ctrl+c again to quit"
		return nil, true

	case msg.String() == "esc":
		if a.mode != viewChat {
			a.mode = viewChat
			a.pickerCursor = 0
			return nil, true
		}

	case msg.String() == "ctrl+a":
		if a.mode == viewAgentPicker {
			a.mode = viewChat
		} else {
			a.mode = viewAgentPicker
			a.pickerCursor = 0
		}
		return nil, true

	case msg.String() == "ctrl+s":
		if a.mode == viewSessionPicker {
			a.mode = viewChat
		} else {
			a.mode = viewSessionPicker
			a.pickerCursor = 0
			return a.loadSessions(), true
		}
		return nil, true

	case msg.String() == "ctrl+n":
		return a.resetSession(), true

	case msg.String() == "ctrl+t":
		a.showThink = !a.showThink
		a.renderChat()
		return nil, true

	case msg.String() == "ctrl+/":
		if a.mode == viewHelp {
			a.mode = viewChat
		} else {
			a.mode = viewHelp
		}
		return nil, true
	}

	// Overlay-specific keys — always handled
	if a.mode == viewAgentPicker {
		return a.handleAgentPickerKey(msg), true
	}
	if a.mode == viewSessionPicker {
		return a.handleSessionPickerKey(msg), true
	}
	if a.mode == viewHelp {
		if msg.String() == "q" || msg.String() == "esc" {
			a.mode = viewChat
		}
		return nil, true
	}

	// Chat mode keys
	if a.mode == viewChat {
		switch msg.String() {
		case "pgup":
			a.viewport.ViewUp()
			return nil, true
		case "pgdown":
			a.viewport.ViewDown()
			return nil, true
		case "home":
			a.viewport.GotoTop()
			return nil, true
		case "end":
			a.viewport.GotoBottom()
			return nil, true
		case "enter":
			if a.streaming {
				return nil, true
			}
			text := strings.TrimSpace(a.input.Value())
			if text == "" {
				return nil, true
			}

			// Handle slash commands
			if strings.HasPrefix(text, "/") {
				cmd := a.handleSlashCommand(text)
				a.input.Reset()
				return cmd, true
			}

			a.input.Reset()
			a.messages = append(a.messages, chatMessage{role: "user", text: text})
			a.renderChat()
			a.viewport.GotoBottom()
			a.streaming = true
			a.assembler.Reset()
			a.statusMsg = ""
			a.thinkingMsgIdx = 0
			a.thinkingTicks = 0
			a.receivedChatEvent = false
			return a.sendMessage(text), true
		}
	}

	// Not handled — let textarea process it
	return nil, false
}

func (a *App) handleSlashCommand(text string) tea.Cmd {
	parts := strings.Fields(text)
	cmd := parts[0]

	switch cmd {
	case "/agent":
		if len(parts) > 1 {
			return a.switchAgent(parts[1])
		}
		a.mode = viewAgentPicker
		a.pickerCursor = 0
		return nil

	case "/session", "/sessions":
		a.mode = viewSessionPicker
		a.pickerCursor = 0
		return a.loadSessions()

	case "/new", "/reset":
		return a.resetSession()

	case "/think":
		if len(parts) > 1 {
			a.statusMsg = "thinking: " + parts[1]
		}
		return nil

	case "/save":
		a.statusMsg = "saving memory..."
		return a.saveMemory()

	case "/quit", "/exit":
		a.cancel()
		a.client.Close()
		return tea.Quit

	case "/help":
		a.mode = viewHelp
		return nil

	default:
		a.statusMsg = fmt.Sprintf("unknown command: %s", cmd)
		return nil
	}
}

func (a *App) handleAgentPickerKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if a.pickerCursor > 0 {
			a.pickerCursor--
		}
	case "down", "j":
		if a.pickerCursor < len(a.agents)-1 {
			a.pickerCursor++
		}
	case "enter":
		if a.pickerCursor < len(a.agents) {
			agent := a.agents[a.pickerCursor]
			a.mode = viewChat
			return a.switchAgent(agent.ID)
		}
	}
	return nil
}

func (a *App) handleSessionPickerKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if a.pickerCursor > 0 {
			a.pickerCursor--
		}
	case "down", "j":
		if a.pickerCursor < len(a.sessions)-1 {
			a.pickerCursor++
		}
	case "enter":
		if a.pickerCursor < len(a.sessions) {
			session := a.sessions[a.pickerCursor]
			a.currentSession = session.Key
			a.mode = viewChat
			return a.loadHistory()
		}
	}
	return nil
}

func (a *App) handleEvent(evt *gateway.EventFrame) tea.Cmd {
	// Log all events to debug file
	if a.debugLog != nil {
		fmt.Fprintf(a.debugLog, "[%s] event=%s payload=%s\n",
			time.Now().Format("15:04:05.000"), evt.Event,
			truncate(string(evt.Payload), 300))
	}

	switch evt.Event {
	case "agent":
		return a.handleAgentEvent(evt)
	case "chat":
		return a.handleChatEvent(evt)
	}
	return nil
}

// agentEventPayload represents the agent lifecycle event.
type agentEventPayload struct {
	RunID      string `json:"runId"`
	Stream     string `json:"stream"`
	SessionKey string `json:"sessionKey"`
	Data       struct {
		Phase     string `json:"phase"` // "start", "end"
		StartedAt int64  `json:"startedAt,omitempty"`
		EndedAt   int64  `json:"endedAt,omitempty"`
	} `json:"data"`
}

func (a *App) handleAgentEvent(evt *gateway.EventFrame) tea.Cmd {
	var payload agentEventPayload
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return nil
	}

	if payload.SessionKey != a.currentSession {
		return nil
	}

	switch payload.Stream {
	case "lifecycle":
		switch payload.Data.Phase {
		case "start":
			a.currentRun = payload.RunID
			a.streaming = true
			a.receivedChatEvent = false
			a.statusMsg = "thinking..."
			a.renderChat()
			a.viewport.GotoBottom()
		case "end":
			// Only fetch history if we never got chat events (Gemini/models
			// that don't stream). If we DID get chat events, the chat "final"
			// handler will complete the response.
			if a.streaming && !a.receivedChatEvent {
				a.streaming = false
				a.currentRun = ""
				a.statusMsg = ""
				a.receivedChatEvent = true // prevent chat final from also fetching
				a.lastCompletedRun = payload.RunID
				return a.fetchLatestResponse()
			}
		}
	case "assistant":
		if a.streaming {
			a.statusMsg = "responding..."
		}
	case "thinking":
		if a.streaming {
			a.statusMsg = "reasoning..."
		}
	case "tool":
		if a.streaming {
			// Extract tool name if available
			var toolData struct {
				Data struct {
					Name string `json:"name"`
				} `json:"data"`
			}
			if json.Unmarshal(evt.Payload, &toolData) == nil && toolData.Data.Name != "" {
				a.statusMsg = "using " + toolData.Data.Name + "..."
			} else {
				a.statusMsg = "using tools..."
			}
		}
	}

	return nil
}

func (a *App) handleChatEvent(evt *gateway.EventFrame) tea.Cmd {
	var payload struct {
		SessionKey string          `json:"sessionKey"`
		State      string          `json:"state"`
		RunID      string          `json:"runId"`
		Message    json.RawMessage `json:"message,omitempty"`
	}
	if err := json.Unmarshal(evt.Payload, &payload); err != nil {
		return nil
	}

	if a.debugLog != nil {
		fmt.Fprintf(a.debugLog, "  CHAT state=%s session=%s (current=%s) streaming=%v receivedChat=%v\n",
			payload.State, payload.SessionKey, a.currentSession, a.streaming, a.receivedChatEvent)
	}

	if payload.SessionKey != a.currentSession {
		if a.debugLog != nil {
			fmt.Fprintf(a.debugLog, "  SKIPPED: session mismatch\n")
		}
		return nil
	}

	switch payload.State {
	case "delta":
		// Streaming content — extract and display it live
		a.receivedChatEvent = true
		if payload.Message != nil {
			var msg struct {
				Content json.RawMessage `json:"content"`
			}
			if json.Unmarshal(payload.Message, &msg) == nil {
				text := chat.ExtractText(msg.Content)
				if text != "" {
					a.ensureAssistantMessage()
					a.messages[len(a.messages)-1].text = text
					a.renderChat()
					a.viewport.GotoBottom()
				}
			}
		}

	case "final":
		// Deduplicate: gateway can send multiple finals for the same run
		if payload.RunID != "" && payload.RunID == a.lastCompletedRun {
			return nil
		}
		a.lastCompletedRun = payload.RunID

		// Response complete
		a.streaming = false
		a.currentRun = ""
		a.statusMsg = ""
		a.receivedChatEvent = true

		if payload.Message != nil {
			var msg struct {
				Content json.RawMessage `json:"content"`
			}
			if json.Unmarshal(payload.Message, &msg) == nil {
				text := chat.ExtractText(msg.Content)
				thinking := chat.ExtractThinking(msg.Content)
				if text != "" {
					a.ensureAssistantMessage()
					last := &a.messages[len(a.messages)-1]
					last.text = text
					last.thinking = thinking
					last.tools = chat.ExtractToolCalls(msg.Content)
					a.renderChat()
					a.viewport.GotoBottom()
					return nil
				}
			}
		}

		// If final had no message content, fetch from history
		return a.fetchLatestResponse()

	case "error":
		a.streaming = false
		a.currentRun = ""
		var errPayload struct {
			ErrorMessage string `json:"errorMessage"`
		}
		json.Unmarshal(evt.Payload, &errPayload)
		if errPayload.ErrorMessage != "" {
			a.statusMsg = "error: " + errPayload.ErrorMessage
		} else {
			a.statusMsg = "agent error"
		}

	case "aborted":
		a.streaming = false
		a.currentRun = ""
		a.statusMsg = "aborted"
	}

	return nil
}

// fetchLatestResponse loads history and appends the latest assistant message.
func (a *App) fetchLatestResponse() tea.Cmd {
	return func() tea.Msg {
		messages, err := a.client.LoadHistory(a.currentSession, 2)
		if err != nil {
			return errMsg{err: fmt.Errorf("fetch response: %w", err)}
		}
		return historyLoadedMsg{messages: messages}
	}
}

// saveMemory asks the agent to persist conversation context to MEMORY.md.
func (a *App) saveMemory() tea.Cmd {
	savePrompt := `Review our conversation history and update your MEMORY.md file with any important context from this session. Include: key topics discussed, decisions made, user preferences learned, files or resources referenced, and any ongoing work context. Keep entries as concise bullet points grouped by topic. Remove stale or outdated entries. Reply with a brief summary of what you saved.`

	a.messages = append(a.messages, chatMessage{role: "system", text: "Saving memory..."})
	a.renderChat()
	a.viewport.GotoBottom()
	a.streaming = true
	a.assembler.Reset()
	a.thinkingMsgIdx = 0
	a.thinkingTicks = 0

	return func() tea.Msg {
		_, err := a.client.SendChat(a.currentSession, savePrompt, "", 0)
		return chatSentMsg{err: err}
	}
}

// Commands

func (a *App) sendMessage(text string) tea.Cmd {
	return func() tea.Msg {
		_, err := a.client.SendChat(a.currentSession, text, "", 0)
		return chatSentMsg{err: err}
	}
}

func (a *App) loadAgents() tea.Cmd {
	return func() tea.Msg {
		agents, err := a.client.ListAgents()
		if err != nil {
			return errMsg{err: err}
		}
		return agentsLoadedMsg{agents: agents}
	}
}

func (a *App) loadHistory() tea.Cmd {
	return func() tea.Msg {
		messages, err := a.client.LoadHistory(a.currentSession, 50)
		if err != nil {
			return errMsg{err: err}
		}
		return historyLoadedMsg{messages: messages, fullLoad: true}
	}
}

func (a *App) loadSessions() tea.Cmd {
	return func() tea.Msg {
		sessions, err := a.client.ListSessions(gateway.SessionsListParams{
			Limit:               20,
			AgentID:             a.currentAgent,
			IncludeDerivedTitle: true,
			IncludeLastMessage:  true,
		})
		if err != nil {
			return errMsg{err: err}
		}
		return sessionsLoadedMsg{sessions: sessions}
	}
}

func (a *App) switchAgent(agentID string) tea.Cmd {
	a.currentAgent = strings.ToLower(agentID)
	a.currentSession = fmt.Sprintf("agent:%s:main", strings.ToLower(agentID))
	a.messages = nil
	a.renderChat()
	return a.loadHistory()
}

func (a *App) resetSession() tea.Cmd {
	return func() tea.Msg {
		err := a.client.ResetSession(a.currentSession, "new")
		return sessionResetMsg{err: err}
	}
}

func (a *App) listenEvents() tea.Cmd {
	return func() tea.Msg {
		select {
		case evt, ok := <-a.client.Events():
			if !ok {
				return gatewayDisconnectedMsg{}
			}
			return gatewayEventMsg{event: evt}
		case <-a.ctx.Done():
			return nil
		}
	}
}

// Layout

func (a *App) updateLayout() {
	headerHeight := 1
	statusHeight := 1
	inputHeight := 3
	borderLines := 2 // borders between sections

	chatHeight := a.height - headerHeight - statusHeight - inputHeight - borderLines
	if chatHeight < 1 {
		chatHeight = 1
	}

	a.viewport.Width = a.width
	a.viewport.Height = chatHeight
	a.input.SetWidth(a.width - 2)

	a.renderChat()
}

func (a *App) renderChat() {
	var sb strings.Builder

	for _, msg := range a.messages {
		switch msg.role {
		case "user":
			sb.WriteString(userLabelStyle.Render("You") + "\n")
			sb.WriteString(userTextStyle.Render(WrapText(msg.text)) + "\n")

		case "assistant":
			agentName := a.currentAgent
			for _, ag := range a.agents {
				if ag.ID == a.currentAgent {
					agentName = ag.Name
					break
				}
			}

			if a.showThink && msg.thinking != "" {
				sb.WriteString(thinkingLabelStyle.Render(agentName+" (thinking)") + "\n")
				sb.WriteString(thinkingTextStyle.Render(WrapText(msg.thinking)) + "\n")
			}

			if len(msg.tools) > 0 {
				for _, t := range msg.tools {
					sb.WriteString(dimStyle().Render("  ⚙ "+t) + "\n")
				}
			}

			if msg.text != "" {
				sb.WriteString(assistantLabelStyle.Render(agentName) + "\n")
				rendered := RenderMarkdown(msg.text)
				sb.WriteString(rendered + "\n")
			}

		case "system":
			sb.WriteString(dimStyle().Render(msg.text) + "\n")
		}
	}

	if a.streaming {
		agentName := a.currentAgent
		for _, ag := range a.agents {
			if ag.ID == a.currentAgent {
				agentName = ag.Name
				break
			}
		}
		frame := spinnerFrames[a.spinnerIdx%len(spinnerFrames)]
		statusText := thinkingMessages[a.thinkingMsgIdx%len(thinkingMessages)]

		// Shimmer effect: color each character with a gradient that shifts over time
		shimmerLine := renderShimmer(statusText, a.spinnerIdx)
		sb.WriteString("\n" + spinnerStyle.Render(frame+" ") +
			lipgloss.NewStyle().Bold(true).Foreground(assistantColor).Render(agentName) +
			"  " + shimmerLine + "\n")
	}

	a.viewport.SetContent(sb.String())
}

// ensureAssistantMessage makes sure the last message is an assistant message.
// If it already is, reuse it. Otherwise append a new one.
func (a *App) ensureAssistantMessage() {
	if len(a.messages) > 0 && a.messages[len(a.messages)-1].role == "assistant" {
		return
	}
	a.messages = append(a.messages, chatMessage{role: "assistant"})
}

func dimStyle() lipgloss.Style {
	return lipgloss.NewStyle().Foreground(dimColor)
}

// renderShimmer renders text with a gradient shimmer that shifts each tick.
func renderShimmer(text string, offset int) string {
	runes := []rune(text)
	var sb strings.Builder
	numColors := len(shimmerColors)
	for i, r := range runes {
		colorIdx := (i + offset) % numColors
		style := lipgloss.NewStyle().Foreground(shimmerColors[colorIdx])
		sb.WriteString(style.Render(string(r)))
	}
	return sb.String()
}

// View

func (a *App) View() string {
	if a.width == 0 {
		return "Loading..."
	}

	header := a.renderHeader()
	status := a.renderStatusBar()

	// Main content depends on mode
	var content string
	switch a.mode {
	case viewAgentPicker:
		content = a.renderAgentPicker()
	case viewSessionPicker:
		content = a.renderSessionPicker()
	case viewHelp:
		content = a.renderHelp()
	default:
		content = a.viewport.View()
	}

	inputBorder := lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false).
		BorderForeground(dimColor).
		Width(a.width)

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		content,
		inputBorder.Render(a.input.View()),
		status,
	)
}

func (a *App) renderHeader() string {
	title := headerStyle.Render(" oclaw ")

	agentName := a.currentAgent
	agentModel := ""
	for _, ag := range a.agents {
		if ag.ID == a.currentAgent {
			agentName = ag.Name
			agentModel = ag.Model
			break
		}
	}
	// Fall back to config agents if gateway agents not loaded
	if agentModel == "" {
		if ac := a.cfg.FindAgent(a.currentAgent); ac != nil {
			agentName = ac.Name
			agentModel = ac.Model
		}
	}

	agentInfo := headerAgentStyle.Render(fmt.Sprintf(" %s (%s) ", agentName, agentModel))

	sessionLabel := headerAgentStyle.Render(fmt.Sprintf(" session: %s ", a.sessionLabel()))

	padding := a.width - lipgloss.Width(title) - lipgloss.Width(agentInfo) - lipgloss.Width(sessionLabel)
	if padding < 0 {
		padding = 0
	}

	return lipgloss.NewStyle().
		Background(accentColor).
		Width(a.width).
		Render(title + agentInfo + strings.Repeat(" ", padding) + sessionLabel)
}

func (a *App) sessionLabel() string {
	parts := strings.Split(a.currentSession, ":")
	if len(parts) >= 3 {
		return parts[2]
	}
	return a.currentSession
}

func (a *App) renderStatusBar() string {
	var connStatus string
	if a.connected {
		connStatus = statusConnectedStyle.Render("● connected")
	} else if a.reconnecting {
		connStatus = statusReconnectingStyle.Render("○ reconnecting...")
	} else {
		connStatus = statusDisconnectedStyle.Render("● disconnected")
	}

	agent := a.currentAgent
	hints := helpKeyStyle.Render("ctrl+/") + helpDescStyle.Render(" help")

	status := connStatus + "  │  " + agent + "  │  " + hints

	if a.statusMsg != "" {
		status += "  │  " + dimStyle().Render(a.statusMsg)
	}

	return statusBarStyle.Width(a.width).Render(status)
}

func (a *App) renderAgentPicker() string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Switch Agent") + "\n\n")

	for i, agent := range a.agents {
		label := fmt.Sprintf("%s (%s) — %s", agent.Name, agent.ID, agent.Model)
		if i == a.pickerCursor {
			sb.WriteString(overlaySelectedStyle.Render("▸ "+label) + "\n")
		} else {
			sb.WriteString(overlayItemStyle.Render("  "+label) + "\n")
		}
	}

	// If no agents loaded from gateway, show config agents
	if len(a.agents) == 0 {
		for i, agent := range a.cfg.Agents {
			label := fmt.Sprintf("%s (%s) — %s", agent.Name, agent.ID, agent.Model)
			if i == a.pickerCursor {
				sb.WriteString(overlaySelectedStyle.Render("▸ "+label) + "\n")
			} else {
				sb.WriteString(overlayItemStyle.Render("  "+label) + "\n")
			}
		}
	}

	sb.WriteString("\n" + dimStyle().Render("↑/↓ navigate  enter select  esc cancel"))
	return sb.String()
}

func (a *App) renderSessionPicker() string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Sessions") + "\n\n")

	if len(a.sessions) == 0 {
		sb.WriteString(dimStyle().Render("  No sessions found") + "\n")
	}

	for i, session := range a.sessions {
		title := session.Title
		if title == "" {
			title = session.Key
		}
		label := title
		if ts := session.UpdatedAtString(); ts != "" {
			label += "  " + dimStyle().Render(ts)
		}

		if i == a.pickerCursor {
			sb.WriteString(overlaySelectedStyle.Render("▸ "+label) + "\n")
		} else {
			sb.WriteString(overlayItemStyle.Render("  "+label) + "\n")
		}
	}

	sb.WriteString("\n" + dimStyle().Render("↑/↓ navigate  enter select  esc cancel"))
	return sb.String()
}

func (a *App) renderHelp() string {
	var sb strings.Builder
	sb.WriteString(overlayTitleStyle.Render("Keyboard Shortcuts") + "\n\n")

	bindings := []struct {
		key  string
		desc string
	}{
		{"enter", "Send message"},
		{"shift+enter", "New line in input"},
		{"ctrl+c", "Abort response / quit"},
		{"ctrl+d", "Quit"},
		{"ctrl+a", "Switch agent"},
		{"ctrl+s", "Browse sessions"},
		{"ctrl+n", "New session"},
		{"ctrl+t", "Toggle thinking text"},
		{"ctrl+/", "This help"},
		{"esc", "Close overlay"},
	}

	for _, b := range bindings {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Width(14).Render(b.key),
			helpDescStyle.Render(b.desc),
		))
	}

	sb.WriteString("\n" + overlayTitleStyle.Render("Slash Commands") + "\n\n")

	commands := []struct {
		cmd  string
		desc string
	}{
		{"/agent <id>", "Switch to agent"},
		{"/session", "Browse sessions"},
		{"/new", "Reset session"},
		{"/save", "Save memory to disk"},
		{"/think <level>", "Set thinking level"},
		{"/help", "Show help"},
		{"/quit", "Quit"},
	}

	for _, c := range commands {
		sb.WriteString(fmt.Sprintf("  %s  %s\n",
			helpKeyStyle.Width(16).Render(c.cmd),
			helpDescStyle.Render(c.desc),
		))
	}

	sb.WriteString("\n" + dimStyle().Render("Press esc or q to close"))
	return sb.String()
}
