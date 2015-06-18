package models

import (
	"reflect"

	cf_tcp_router "github.com/cloudfoundry-incubator/cf-tcp-router"
)

type RoutingKey struct {
	Port uint16
}

type BackendServerInfo struct {
	Address string
	Port    uint16
}

type BackendServerInfos []BackendServerInfo

type RoutingTableEntry struct {
	Backends map[BackendServerInfo]struct{}
}

func NewRoutingTableEntry(backends BackendServerInfos) RoutingTableEntry {
	routingTableEntry := RoutingTableEntry{
		Backends: make(map[BackendServerInfo]struct{}),
	}
	for _, backend := range backends {
		routingTableEntry.Backends[backend] = struct{}{}
	}
	return routingTableEntry
}

type RoutingTable struct {
	Entries map[RoutingKey]RoutingTableEntry
}

func NewRoutingTable() RoutingTable {
	return RoutingTable{
		Entries: make(map[RoutingKey]RoutingTableEntry),
	}
}

func (table RoutingTable) Set(key RoutingKey, newEntry RoutingTableEntry) bool {
	existingEntry, ok := table.Entries[key]
	if ok == true && reflect.DeepEqual(existingEntry, newEntry) {
		return false
	}
	table.Entries[key] = newEntry
	return true
}

func (table RoutingTable) Get(key RoutingKey) RoutingTableEntry {
	return table.Entries[key]
}

func ToRoutingTableEntry(mappingRequest cf_tcp_router.MappingRequest) (RoutingKey, RoutingTableEntry) {
	routingKey := RoutingKey{mappingRequest.ExternalPort}
	routingTableEntry := RoutingTableEntry{
		Backends: make(map[BackendServerInfo]struct{}),
	}
	for _, backend := range mappingRequest.Backends {
		backendServerInfo := BackendServerInfo{
			Address: backend.Address,
			Port:    backend.Port,
		}
		routingTableEntry.Backends[backendServerInfo] = struct{}{}
	}
	return routingKey, routingTableEntry
}
