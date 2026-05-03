package audit

import (
	"encoding/json"
	"time"
)

const (
	EventAuthSuccess    = "auth_success"
	EventAuthFailure    = "auth_failure"
	EventNodeRegister   = "node_register"
	EventNodeUnregister = "node_unregister"
	EventPluginLoad     = "plugin_load"
	EventPluginUnload   = "plugin_unload"
)

const (
	ResultSuccess = "success"
	ResultFailure = "failure"
)

type AuditLog struct {
	ID        int64     `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`
	ClientID  string    `json:"client_id,omitempty"`
	Role      string    `json:"role,omitempty"`
	Result    string    `json:"result"`
	Details   string    `json:"details,omitempty"`
}

type AuthDetails struct {
	ClientID string `json:"client_id"`
	Reason   string `json:"reason,omitempty"`
}

type NodeDetails struct {
	NodeID   int64  `json:"node_id,omitempty"`
	NodeName string `json:"node_name,omitempty"`
	Host     string `json:"host,omitempty"`
	Port     int    `json:"port,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type PluginDetails struct {
	PluginName string `json:"plugin_name"`
	Reason     string `json:"reason,omitempty"`
}

func LogAuth(clientID, role, result, reason string) error {
	details := AuthDetails{ClientID: clientID, Reason: reason}
	detailsJSON, _ := json.Marshal(details)
	return InsertLog(EventAuthSuccess, clientID, role, result, string(detailsJSON))
}

func LogNodeRegister(nodeID int64, nodeName, host string, port int, role string) error {
	details := NodeDetails{NodeID: nodeID, NodeName: nodeName, Host: host, Port: port}
	detailsJSON, _ := json.Marshal(details)
	return InsertLog(EventNodeRegister, "", role, ResultSuccess, string(detailsJSON))
}

func LogNodeUnregister(nodeID int64, nodeName, reason, role string) error {
	details := NodeDetails{NodeID: nodeID, NodeName: nodeName, Reason: reason}
	detailsJSON, _ := json.Marshal(details)
	return InsertLog(EventNodeUnregister, "", role, ResultSuccess, string(detailsJSON))
}

func LogPluginLoad(pluginName, role string) error {
	details := PluginDetails{PluginName: pluginName}
	detailsJSON, _ := json.Marshal(details)
	return InsertLog(EventPluginLoad, "", role, ResultSuccess, string(detailsJSON))
}

func LogPluginUnload(pluginName, role, reason string) error {
	details := PluginDetails{PluginName: pluginName, Reason: reason}
	detailsJSON, _ := json.Marshal(details)
	return InsertLog(EventPluginUnload, "", role, ResultSuccess, string(detailsJSON))
}