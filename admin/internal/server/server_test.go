package server

import (
	"crypto/tls"
	"net"
	"testing"
	"time"

	"anthill/admin/internal/model"
)

type mockConn struct {
	net.Conn
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func TestConnManager_AutoModeRetry(t *testing.T) {
	cm := NewConnManager(nil)

	nodeID := "123"

	if cm.GetAutoModeRetryCount(nodeID) != 0 {
		t.Errorf("expected 0 retries, got %d", cm.GetAutoModeRetryCount(nodeID))
	}

	cm.IncrementAutoModeRetry(nodeID)
	if cm.GetAutoModeRetryCount(nodeID) != 1 {
		t.Errorf("expected 1 retry, got %d", cm.GetAutoModeRetryCount(nodeID))
	}

	cm.ResetAutoModeRetry(nodeID)
	if cm.GetAutoModeRetryCount(nodeID) != 0 {
		t.Errorf("expected 0 retries after reset, got %d", cm.GetAutoModeRetryCount(nodeID))
	}
}

func TestConnManager_ShouldSwitchToPassive(t *testing.T) {
	cm := NewConnManager(nil)
	cm.maxAutoRetries = 3

	nodeID := "123"

	if cm.ShouldSwitchToPassive(nodeID) {
		t.Error("expected false when no retries recorded")
	}

	for i := 0; i < 3; i++ {
		cm.IncrementAutoModeRetry(nodeID)
	}

	if !cm.ShouldSwitchToPassive(nodeID) {
		t.Error("expected true after max retries")
	}
}

func TestConnManager_AddConnection(t *testing.T) {
	cm := NewConnManager(nil)

	conn, err := net.Dial("tcp", "127.0.0.1:9999")
	if err != nil {
		t.Skip("skipping test - no listener available")
		return
	}
	defer conn.Close()

	nodeID := "123"
	err = cm.AddConnection(nodeID, conn, "tls", "active_tls")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !cm.ShouldSwitchToPassive(nodeID) {
		t.Error("expected false for active_tls mode")
	}

	conn2, _ := net.Dial("tcp", "127.0.0.1:9999")
	if conn2 != nil {
		defer conn2.Close()
		err = cm.AddConnection(nodeID, conn2, "tls", "passive_tls")
		if err != nil {
			t.Errorf("unexpected error: %v", err)
		}

		if cm.GetAutoModeRetryCount(nodeID) != 0 {
			t.Errorf("expected 0 retries after passive connection, got %d", cm.GetAutoModeRetryCount(nodeID))
		}
	}
}

func TestConnManager_GetSetCACert(t *testing.T) {
	cm := NewConnManager(nil)

	if cm.GetCACert() != nil {
		t.Error("expected nil cert before setting")
	}

	cert := &tls.Certificate{}
	cm.SetCACert(cert)

	if cm.GetCACert() != cert {
		t.Error("cert not set correctly")
	}
}

func TestConnectModeConstants(t *testing.T) {
	if model.ConnectModeActiveTLS != "active_tls" {
		t.Errorf("ConnectModeActiveTLS = %v, want active_tls", model.ConnectModeActiveTLS)
	}
	if model.ConnectModeActiveWSS != "active_wss" {
		t.Errorf("ConnectModeActiveWSS = %v, want active_wss", model.ConnectModeActiveWSS)
	}
	if model.ConnectModePassiveTLS != "passive_tls" {
		t.Errorf("ConnectModePassiveTLS = %v, want passive_tls", model.ConnectModePassiveTLS)
	}
	if model.ConnectModePassiveWSS != "passive_wss" {
		t.Errorf("ConnectModePassiveWSS = %v, want passive_wss", model.ConnectModePassiveWSS)
	}
	if model.ConnectModeAuto != "auto" {
		t.Errorf("ConnectModeAuto = %v, want auto", model.ConnectModeAuto)
	}
}

func TestValidConnectMode(t *testing.T) {
	validModes := []string{"active_tls", "active_wss", "passive_tls", "passive_wss", "auto"}
	for _, mode := range validModes {
		if !model.ValidConnectMode(mode) {
			t.Errorf("ValidConnectMode(%s) = false, want true", mode)
		}
	}

	invalidModes := []string{"", "invalid", "ACTIVE_TLS", "passive", "tls"}
	for _, mode := range invalidModes {
		if model.ValidConnectMode(mode) {
			t.Errorf("ValidConnectMode(%s) = true, want false", mode)
		}
	}
}

func TestNodeConnection(t *testing.T) {
	conn, err := net.Dial("tcp", "127.0.0.1:9999")
	if err != nil {
		t.Skip("skipping test - no listener available")
		return
	}
	defer conn.Close()

	nc := &NodeConnection{
		NodeID:        "123",
		Conn:          conn,
		Protocol:      "tls",
		LastHeartbeat: time.Now(),
		Mode:          "active_tls",
	}

	if nc.NodeID != "123" {
		t.Errorf("NodeID = %v, want 123", nc.NodeID)
	}
	if nc.Protocol != "tls" {
		t.Errorf("Protocol = %v, want tls", nc.Protocol)
	}
	if nc.Mode != "active_tls" {
		t.Errorf("Mode = %v, want active_tls", nc.Mode)
	}
}

func TestConnManager_Close(t *testing.T) {
	cm := NewConnManager(nil)

	conn1 := &mockConn{}
	conn2 := &mockConn{}

	cm.AddConnection("1", conn1, "tls", "active")
	cm.AddConnection("2", conn2, "wss", "passive")

	err := cm.Close()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !conn1.closed {
		t.Error("conn1 should be closed")
	}
	if !conn2.closed {
		t.Error("conn2 should be closed")
	}

	if len(cm.nodes) != 0 {
		t.Errorf("nodes map should be empty, got %d", len(cm.nodes))
	}
}

func TestConnManager_ActiveClients(t *testing.T) {
	cm := NewConnManager(nil)

	if _, ok := cm.GetActiveClient("nonexistent"); ok {
		t.Error("expected not found for nonexistent client")
	}

	cm.AddActiveClient("123", nil)

	if _, ok := cm.GetActiveClient("123"); !ok {
		t.Error("expected found after adding client")
	}

	cm.RemoveActiveClient("123")

	if _, ok := cm.GetActiveClient("123"); ok {
		t.Error("expected not found after removing client")
	}
}