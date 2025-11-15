package core

import "testing"

func TestNetwork_Delivery(t *testing.T) {
	net := NewNetwork(100, 1, 1)
	// create two lightweight nodes (minimal fields needed for delivery)
	n1 := &Node{id: NodeID("node00")}
	n2 := &Node{id: NodeID("node01")}
	RegisterNode(net, n1)
	RegisterNode(net, n2)

	// send message from n1 to n2
	net.Send(n1.id, n2.id, "ping")
	// advance ticks enough to deliver
	for i := 0; i < 5; i++ {
		net.Tick()
	}

	// message should be delivered in n2.inbox channel (non-blocking)
	select {
	case m := <-n2.inbox:
		if m.Body != "ping" {
			t.Fatalf("unexpected body: %v", m.Body)
		}
	default:
		t.Fatalf("message not delivered to n2")
	}
}
