package snmp

import (
	"github.com/gosnmp/gosnmp"
	"log"
	"time"
)

// Discovery performs an SNMP GET request to obtain the system name of a device using the provided IP, community, and version.
// It returns a map with the status (success/fail) and the retrieved system name (if successful).
func Discovery(ip, community, version string) map[string]interface{} {

	snmp := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: community,
		Timeout:   time.Second * 2,
		Retries:   3,
	}

	switch version {
	case "1":
		snmp.Version = gosnmp.Version1

	case "2", "2c":
		snmp.Version = gosnmp.Version2c

	case "3":
		snmp.Version = gosnmp.Version3

	default:

		log.Printf("Unsupported SNMP version: %s. Defaulting to Version2c", version)

		snmp.Version = gosnmp.Version2c
	}

	err := snmp.Connect()

	if err != nil {
		return map[string]interface{}{

			"status": "fail",

			"systemName": "",
		}
	}

	defer snmp.Conn.Close()

	oid := "1.3.6.1.2.1.1.5.0"

	result, err := snmp.Get([]string{oid})

	if err != nil {
		return map[string]interface{}{

			"status": "fail",

			"systemName": "",
		}
	}

	for _, variable := range result.Variables {

		if variable.Type == gosnmp.OctetString {

			return map[string]interface{}{

				"status": "success",

				"systemName": string(variable.Value.([]byte)),
			}
		}
	}

	return map[string]interface{}{

		"status": "fail",

		"systemName": "",
	}
}
