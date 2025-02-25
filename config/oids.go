package config

// SNMPOids OIDs for SNMP system data
var SNMPOids = map[string]string{
	"1.3.6.1.2.1.1.5.0": "system.name",
	"1.3.6.1.2.1.1.1.0": "system.description",
	"1.3.6.1.2.1.1.6.0": "system.location",
	"1.3.6.1.2.1.1.2.0": "system.objectId",
	"1.3.6.1.2.1.1.3.0": "system.uptime",
	"1.3.6.1.2.1.2.1.0": "system.interfaces",
}

// InterfaceOids OIDs for SNMP interface data
var InterfaceOids = map[string]string{
	"1.3.6.1.2.1.2.2.1.1":     "Index",                // Interface Index
	"1.3.6.1.2.1.31.1.1.1.1":  "Name",                 // Interface Name
	"1.3.6.1.2.1.31.1.1.1.18": "Alias",                // Interface Alias
	"1.3.6.1.2.1.2.2.1.8":     "OperationalStatus",    // Operational Status (up/down)
	"1.3.6.1.2.1.2.2.1.7":     "AdminStatus",          // Administrative Status
	"1.3.6.1.2.1.2.2.1.2":     "Description",          // Interface Description
	"1.3.6.1.2.1.2.2.1.20":    "SentErrorPackets",     // Sent Error Packets
	"1.3.6.1.2.1.2.2.1.14":    "ReceivedErrorPackets", // Received Error Packets
	"1.3.6.1.2.1.2.2.1.16":    "SentOctets",           // Sent Octets (bytes sent)
	"1.3.6.1.2.1.2.2.1.10":    "ReceivedOctets",       // Received Octets (bytes received)
	"1.3.6.1.2.1.2.2.1.5":     "Speed",                // Interface Speed (in bps)
	"1.3.6.1.2.1.2.2.1.6":     "PhysicalAddress",      // Physical Address (MAC Address)
	"1.3.6.1.2.1.2.2.1.13":    "DiscardPackets",       // Discarded Packets
	"1.3.6.1.2.1.2.2.1.11":    "InPackets",            // Incoming Packets
	"1.3.6.1.2.1.2.2.1.17":    "OutPackets",           // Outgoing Packets
}
