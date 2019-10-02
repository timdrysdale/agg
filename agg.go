package agg

import (
	"fmt"
	"strings"
	"sync"

	"github.com/jinzhu/copier"
	"github.com/timdrysdale/hub"
)

func New() *Hub {

	h := &Hub{
		Hub:        hub.New(),
		Broadcast:  make(chan hub.Message),
		Register:   make(chan *hub.Client),
		Unregister: make(chan *hub.Client),
		Streams:    make(map[string]map[*hub.Client]bool),
		SubClients: make(map[*hub.Client]map[*SubClient]bool),
		Rules:      make(map[string][]string),
		Add:        make(chan Rule),
		Delete:     make(chan Rule),
	}

	return h

}

func (h *Hub) Run(closed chan struct{}) {
	h.RunOptionalStats(closed, false)
}

func (h *Hub) RunWithStats(closed chan struct{}) {
	h.RunOptionalStats(closed, true)
}

func (h *Hub) RunOptionalStats(closed chan struct{}, withStats bool) {

	//start the hub
	if withStats {
		go h.Hub.RunWithStats(closed)
	} else {
		go h.Hub.Run(closed)
	}

	for {
		select {
		case <-closed:
			return
		case client := <-h.Register:
			fmt.Println("agg REGISTER")
			if strings.HasPrefix(client.Topic, "/stream/") {
				fmt.Println("agg registering for stream")
				// register the client to the stream
				if _, ok := h.Streams[client.Topic]; !ok {
					h.Streams[client.Topic] = make(map[*hub.Client]bool)
				}
				h.Streams[client.Topic][client] = true

				// register the client to any feeds currently set by stream rule
				if feeds, ok := h.Rules[client.Topic]; ok {
					h.SubClients[client] = make(map[*SubClient]bool)
					wg := &sync.WaitGroup{}
					for _, feed := range feeds {
						// create and store the subclients we will register with the hub
						subClient := &SubClient{Client: &hub.Client{}, Wg: wg}
						copier.Copy(&subClient.Client, client)
						subClient.Client.Topic = feed
						subClient.Client.Send = make(chan hub.Message)
						subClient.Stopped = make(chan struct{})
						h.SubClients[client][subClient] = true
						wg.Add(1)
						go subClient.RelayTo(client)
						h.Hub.Register <- subClient.Client
					}

				}
			} else {
				// register client directly
				h.Hub.Register <- client
				fmt.Println("agg registering for direct")
			}
		case client := <-h.Unregister:
			fmt.Println("agg unregistering")
			if strings.HasPrefix(client.Topic, "/stream/") {
				// unregister any subclients that are registered to feeds
				wg := &sync.WaitGroup{}
				for subClient := range h.SubClients[client] {
					h.Hub.Unregister <- subClient.Client
					close(subClient.Stopped)
					wg = subClient.Wg
				}

				wg.Wait() //same wg for all subclients

				// delete the client from the stream
				if _, ok := h.Streams[client.Topic]; ok {
					delete(h.Streams[client.Topic], client)
					close(client.Send)
				}
				delete(h.SubClients, client)

			} else {
				// unregister client directly
				h.Hub.Unregister <- client
			}
		case msg := <-h.Broadcast:
			fmt.Println("agg got message to broadcast")
			fmt.Printf("agg current subclients: %v\n", h.SubClients)
			// defer handling to hub
			// note that non-responsive clients will get deleted
			h.Hub.Broadcast <- msg
		case rule := <-h.Add:
			// unregister clients from old feeds, if any
			if _, ok := h.Rules[rule.Stream]; ok {
				for client, _ := range h.Streams[rule.Stream] {
					for subClient := range h.SubClients[client] {
						h.Hub.Unregister <- subClient.Client
						close(subClient.Stopped)
					}
				}
			}
			//set new rule
			h.Rules[rule.Stream] = rule.Feeds

			// register the clients to any feeds currently set by stream rule
			if feeds, ok := h.Rules[rule.Stream]; ok {
				for client, _ := range h.Streams[rule.Stream] {
					h.SubClients[client] = make(map[*SubClient]bool)
					wg := &sync.WaitGroup{}
					for _, feed := range feeds {
						// create and store the subclients we will register with the hub
						subClient := &SubClient{Client: &hub.Client{}, Wg: wg}
						copier.Copy(&subClient.Client, client)
						subClient.Client.Topic = feed
						subClient.Client.Send = make(chan hub.Message)
						subClient.Stopped = make(chan struct{})
						h.SubClients[client][subClient] = true
						wg.Add(1)
						go subClient.RelayTo(client)
						h.Hub.Register <- subClient.Client
					}
				}
			}

		case rule := <-h.Delete:
			// unregister clients from old feeds, if any
			if _, ok := h.Rules[rule.Stream]; ok {
				for client, _ := range h.Streams[rule.Stream] {
					for subClient := range h.SubClients[client] {
						h.Hub.Unregister <- subClient.Client
						close(subClient.Stopped)
					}
				}
			}

			// delete rule
			delete(h.Rules, rule.Stream)
		}
	}
}

// relay messages from subClient to Client
func (sc *SubClient) RelayTo(c *hub.Client) {
	defer sc.Wg.Done()
	for {
		select {
		case <-sc.Stopped:
			break
		case msg, ok := <-sc.Client.Send:
			if ok {
				fmt.Println("agg subclient RelayTo")
				c.Send <- msg
			}
		}
	}
}
