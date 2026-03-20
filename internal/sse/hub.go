package sse

import (
	"sync"
)

type NotificationHub interface {
	SubscribeGlobal() chan bool
	SubscribeTablet(id int64) chan bool
	Unsubscribe(ch chan bool, id int64)
	Notify(id int64)
}

type Hub struct {
	mu            sync.Mutex
	globalClients map[chan bool]bool
	tabletClients map[int64]map[chan bool]bool
}

var Instance = &Hub{
	globalClients: make(map[chan bool]bool),
	tabletClients: make(map[int64]map[chan bool]bool),
}

// WsHub is the global WebSocket hub for real-time communication
var WsHub *WebSocketHub

// InitWsHub initializes the WebSocket hub
func InitWsHub() {
	WsHub = NewWebSocketHub()
	go WsHub.Run()
}

func (h *Hub) SubscribeGlobal() chan bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	ch := make(chan bool, 1)
	h.globalClients[ch] = true
	return ch
}

func (h *Hub) SubscribeTablet(id int64) chan bool {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.tabletClients[id] == nil {
		h.tabletClients[id] = make(map[chan bool]bool)
	}
	ch := make(chan bool, 1)
	h.tabletClients[id][ch] = true
	return ch
}

func (h *Hub) Unsubscribe(ch chan bool, id int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.globalClients, ch)
	if id != 0 && h.tabletClients[id] != nil {
		delete(h.tabletClients[id], ch)
	}
}

func (h *Hub) NotifyNewReport(tabId int64) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.globalClients {
		select {
		case ch <- true:
		default:

		}
	}

	if clients, ok := h.tabletClients[tabId]; ok {
		for ch := range clients {
			select {
			case ch <- true:
			default:

			}
		}
	}
}
