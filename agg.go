package agg

import (
	"strings"

	"github.com/timdrysdale/hub"
)

func New() *Hub {

	h := &Hub{
		Hub:        hub.New(),
		Broadcast:  make(chan hub.Message),
		Register:   make(chan *hub.Client),
		Unregister: make(chan *hub.Client),
		Clients:    make(map[string]map[*hub.Client]bool),
		Rules:      make(map[string][]string),
		Add:        make(chan Rule),
		Delete:     make(chan Rule),
	}

	return h

}

func (h *Hub) Run(closed chan struct{}) {
	go h.Hub.Run(closed) //start the hub
	for {
		select {
		case <-closed:
			return
		case client := <-h.Register:
			if strings.HasPrefix(client.Topic, "/stream/") {
				// register the client to the stream
				if _, ok := h.Clients[client.Topic]; !ok {
					h.Clients[client.Topic] = make(map[*hub.Client]bool)
				}
				h.Clients[client.Topic][client] = true

				// register the client to any feeds currently set by stream rule
				if feeds, ok := h.Rules[client.Topic]; ok {
					for client, _ := range h.Clients[client.Topic] {
						for _, feed := range feeds {
							client.Topic = feed
							h.Hub.Register <- client
						}
					}
				}
			} else {
				// register client normally
				h.Hub.Register <- client
			}
		case client := <-h.Unregister:
			if strings.HasPrefix(client.Topic, "/stream/") {
				// delete the client from the stream
				if _, ok := h.Clients[client.Topic]; ok {
					delete(h.Clients[client.Topic], client)
					close(client.Send)
				}
				// unregister the client from any feeds currently set by a rule
				if feeds, ok := h.Rules[client.Topic]; ok {
					for client, _ := range h.Clients[client.Topic] {
						for _, feed := range feeds {
							client.Topic = feed
							h.Hub.Unregister <- client
						}
					}
				}
			} else {
				// unregister client normally
				h.Hub.Unregister <- client
			}
		case msg := <-h.Broadcast:
			// defer handling to hub
			// note that non-responsive clients will get deleted
			h.Hub.Broadcast <- msg
		case rule := <-h.Add:
			// unregister clients from old feeds, if any
			if feeds, ok := h.Rules[rule.Stream]; ok {
				for client, _ := range h.Clients[rule.Stream] {
					for _, feed := range feeds {
						client.Topic = feed
						h.Hub.Unregister <- client
					}
				}
			}
			//set new rule
			h.Rules[rule.Stream] = rule.Feeds

			// register clients to new feeds
			if feeds, ok := h.Rules[rule.Stream]; ok {
				for client, _ := range h.Clients[rule.Stream] {
					for _, feed := range feeds {
						client.Topic = feed
						h.Hub.Register <- client
					}
				}
			}

		case rule := <-h.Delete:
			// unregister clients from old feeds, if any
			if feeds, ok := h.Rules[rule.Stream]; ok {
				for client, _ := range h.Clients[rule.Stream] {
					for _, feed := range feeds {
						client.Topic = feed
						h.Hub.Unregister <- client
					}
				}
			}
			// delete rule
			delete(h.Rules, rule.Stream)
		}
	}
}
