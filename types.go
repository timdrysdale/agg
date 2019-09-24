package agg

import "github.com/timdrysdale/hub"

type Hub struct {
	Hub        *hub.Hub
	Broadcast  chan hub.Message
	Register   chan *hub.Client
	Unregister chan *hub.Client
	Add        chan Rule
	Delete     chan Rule
	Rules      map[string][]string
	Streams    map[string]map[*hub.Client]bool
}

type Rule struct {
	Stream string
	Feeds  []string
}
