package core

import "testing"

func TestNetwork_Delivery(t *testing.T) {
	net := NewNetwork(100, 1, 1, 0, 0, 16, nil, 0, 5)
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
	net := NewNetwork(200, 1, 1, 1, 0, 16, nil, 0, 5)
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
	net := NewNetwork(300, 1, 1, 0, 1, 16, nil, 0, 5)
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

func TestNetwork_RateLimitBlocksAndUnblocks(t *testing.T) {
	net := NewNetwork(400, 1, 1, 0, 0, 16, nil, 1, 2)
	n1 := &Node{id: NodeID("node00"), kp: GenKeypair()}
	n2 := &Node{id: NodeID("node01"), kp: GenKeypair()}
	RegisterNode(net, n1)
	RegisterNode(net, n2)

	// exceed rate limit
	net.Send(n1.id, n2.id, "first")
	net.Send(n1.id, n2.id, "second_should_block")
	for i := 0; i < 5; i++ {
		net.Tick()
	}
	first, ok := <-n2.inbox
	if !ok {
		t.Fatalf("expected inbox message")
	}
	if first.Body != "first" {
		t.Fatalf("expected first message, got %v", first.Body)
	}
	select {
	case <-n2.inbox:
		t.Fatalf("second message should have been blocked")
	default:
	}
	// advance ticks to unblock and send again
	for i := 0; i < 5; i++ {
		net.Tick()
	}
	net.Send(n1.id, n2.id, "after_unblock")
	for i := 0; i < 5; i++ {
		net.Tick()
	}
	select {
	case m := <-n2.inbox:
		if m.Body != "after_unblock" {
			t.Fatalf("expected after_unblock, got %v", m.Body)
		}
	default:
		t.Fatalf("third message should deliver after unblock")
	}
}
