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
	"1.3.6.1.2.1.2.2.1.1":     "Index",
	"1.3.6.1.2.1.31.1.1.1.1":  "Name",
	"1.3.6.1.2.1.31.1.1.1.18": "Alias",
	"1.3.6.1.2.1.2.2.1.8":     "OperationalStatus",
	"1.3.6.1.2.1.2.2.1.7":     "AdminStatus",
	"1.3.6.1.2.1.2.2.1.2":     "Description",
	"1.3.6.1.2.1.2.2.1.20":    "SentErrorPackets",
	"1.3.6.1.2.1.2.2.1.14":    "ReceivedErrorPackets",
	"1.3.6.1.2.1.2.2.1.16":    "SentOctets",
	"1.3.6.1.2.1.2.2.1.10":    "ReceivedOctets",
	"1.3.6.1.2.1.2.2.1.5":     "Speed",
	"1.3.6.1.2.1.2.2.1.6":     "PhysicalAddress",
	"1.3.6.1.2.1.2.2.1.13":    "DiscardPackets",
	"1.3.6.1.2.1.2.2.1.11":    "InPackets",
	"1.3.6.1.2.1.2.2.1.17":    "OutPackets",
}
