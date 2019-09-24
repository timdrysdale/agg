package agg

import (
	"bytes"
	"reflect"
	"testing"
	"time"

	"github.com/timdrysdale/hub"
)

func TestInstantiateHub(t *testing.T) {

	h := New()

	if reflect.TypeOf(h.Broadcast) != reflect.TypeOf(make(chan hub.Message)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}
	if reflect.TypeOf(h.Register) != reflect.TypeOf(make(chan *hub.Client)) {
		t.Error("Hub.Register channel of wrong type")
	}
	if reflect.TypeOf(h.Unregister) != reflect.TypeOf(make(chan *hub.Client)) {
		t.Error("Hub.Unregister channel of wrong type")
	}

	if reflect.TypeOf(h.Streams) != reflect.TypeOf(make(map[string]map[*hub.Client]bool)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}

	if reflect.TypeOf(h.Rules) != reflect.TypeOf(make(map[string][]string)) {
		t.Error("Hub.Broadcast channel of wrong type")
	}

}

func TestRegisterClient(t *testing.T) {

	topic := "/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Hub.Clients[topic][c]; !ok {
		t.Error("Client not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}
	close(closed)
}

func TestUnRegisterClient(t *testing.T) {

	topic := "/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Hub.Clients[topic][c]; !ok {
		t.Error("Client not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}

	time.Sleep(time.Millisecond)
	h.Unregister <- c
	time.Sleep(time.Millisecond)
	if val, ok := h.Hub.Clients[topic][c]; ok {
		if val {
			t.Error("Client still registered")
		}
	}
	close(closed)
}

func TestSendMessage(t *testing.T) {

	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	topicA := "/videoA"
	c1 := &hub.Client{Hub: h.Hub, Name: "1", Topic: topicA, Send: make(chan hub.Message), Stats: hub.NewClientStats()}
	c2 := &hub.Client{Hub: h.Hub, Name: "2", Topic: topicA, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	topicB := "/videoB"
	c3 := &hub.Client{Hub: h.Hub, Name: "2", Topic: topicB, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c1
	h.Register <- c2
	h.Register <- c3

	content := []byte{'t', 'e', 's', 't'}

	m := &hub.Message{Data: content, Sender: *c1, Sent: time.Now(), Type: 0}

	var start time.Time

	rxCount := 0

	go func() {
		timer := time.NewTimer(5 * time.Millisecond)
	COLLECT:
		for {
			select {
			case <-c1.Send:
				t.Error("Sender received echo")
			case msg := <-c2.Send:
				elapsed := time.Since(start)
				if elapsed > (time.Millisecond) {
					t.Error("Message took longer than 1 millisecond, ", elapsed)
				}
				rxCount++
				if bytes.Compare(msg.Data, content) != 0 {
					t.Error("Wrong data in message")
				}
			case <-c3.Send:
				t.Error("Wrong client received message")
			case <-timer.C:
				break COLLECT
			}
		}
	}()

	time.Sleep(time.Millisecond)
	start = time.Now()
	h.Broadcast <- *m
	time.Sleep(time.Millisecond)
	if rxCount != 1 {
		t.Error("Receiver did not receive message in correct quantity, wanted 1 got ", rxCount)
	}
	close(closed)
}

func TestRegisterStreamNoRule(t *testing.T) {

	topic := "/stream/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[topic][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}
	close(closed)
}

func TestUnRegisterStreamNoRule(t *testing.T) {

	topic := "/stream/video0"
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: topic, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[topic][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Client registered but not made true in map")
	}

	time.Sleep(time.Millisecond)
	h.Unregister <- c
	time.Sleep(time.Millisecond)
	if val, ok := h.Streams[topic][c]; ok {
		if val {
			t.Error("Stream still registered")
		}
	}
	close(closed)
}

func TestAddRuleNoStream(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	stream := "/stream/large"
	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}
	close(closed)

}

func TestDeleteRuleNoStream(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	stream := "/stream/large"
	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}

	h.Delete <- *r

	time.Sleep(time.Millisecond)

	if _, ok := h.Rules[stream]; ok {
		t.Error("Rule still registered in Rules")

	}

	close(closed)

}

func TestAddRuleAddStream(t *testing.T) {
	h := New()
	closed := make(chan struct{})
	go h.Run(closed)

	// add rule
	stream := "/stream/large"

	feeds := []string{"video0", "audio"}
	r := &Rule{Stream: stream, Feeds: feeds}

	h.Add <- *r

	time.Sleep(time.Millisecond)

	if val, ok := h.Rules[stream]; !ok {
		t.Error("Rule not registered in Rules")

	} else if len(val) != len(feeds) {
		t.Error("Rule has incorrect number of feeds")
	}

	// register client to stream
	c := &hub.Client{Hub: h.Hub, Name: "aa", Topic: stream, Send: make(chan hub.Message), Stats: hub.NewClientStats()}

	h.Register <- c

	time.Sleep(time.Millisecond)

	if val, ok := h.Streams[stream][c]; !ok {
		t.Error("Stream not registered in topic")
	} else if val == false {
		t.Error("Stream registered but not made true in map")
	}

	time.Sleep(time.Millisecond)

	//Check client is registered to feeds

	isFound := make([]bool, len(feeds))

	for i, feed := range feeds {

		for subclient := range h.SubClients[c] {

			if _, ok := h.Hub.Clients[feed][subclient]; ok {
				isFound[i] = true
			}
		}
	}

	for i, val := range isFound {
		if !val {
			t.Error("did not find subclient for", feeds[i])
		}
	}
	close(closed)

}