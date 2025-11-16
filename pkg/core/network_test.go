package core

import "testing"

func TestNetwork_Delivery(t *testing.T) {
	net := NewNetwork(100, 1, 1, 0, 0, 16, nil)
	// create two lightweight nodes (minimal fields needed for delivery)
	n1 := &Node{id: NodeID("node00"), kp: GenKeypair()}
	n2 := &Node{id: NodeID("node01"), kp: GenKeypair()}
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

func TestNetwork_DropRate(t *testing.T) {
	net := NewNetwork(200, 1, 1, 1, 0, 16, nil)
	n1 := &Node{id: NodeID("node00"), kp: GenKeypair()}
	n2 := &Node{id: NodeID("node01"), kp: GenKeypair()}
	RegisterNode(net, n1)
	RegisterNode(net, n2)

	net.Send(n1.id, n2.id, "ping")
	for i := 0; i < 5; i++ {
		net.Tick()
	}

	select {
	case <-n2.inbox:
		t.Fatalf("message should have been dropped due to dropRate=1")
	default:
	}
}

func TestNetwork_DuplicateDelivery(t *testing.T) {
	net := NewNetwork(300, 1, 1, 0, 1, 16, nil)
	n1 := &Node{id: NodeID("node00"), kp: GenKeypair()}
	n2 := &Node{id: NodeID("node01"), kp: GenKeypair()}
	RegisterNode(net, n1)
	RegisterNode(net, n2)

	net.Send(n1.id, n2.id, "dup")
	for i := 0; i < 5; i++ {
		net.Tick()
	}

	count := 0
loop:
	for {
		select {
		case m := <-n2.inbox:
			if m.Body != "dup" {
				t.Fatalf("unexpected body: %v", m.Body)
			}
			count++
		default:
			break loop
		}
	}
	if count < 2 {
		t.Fatalf("expected duplicated delivery, only received %d messages", count)
	}
}
