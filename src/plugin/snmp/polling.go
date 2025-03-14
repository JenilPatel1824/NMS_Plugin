package snmp

import (
	"GO_Plugin/src/util"
	"encoding/hex"
	"fmt"
	"github.com/gosnmp/gosnmp"
	"log"
	"strings"
	"time"
)

const (
	systemName        = "system.name"
	systemDescription = "system.description"
	systemLocation    = "system.location"
	systemObjectID    = "system.objectId"
	systemUptime      = "system.uptime"
	systemInterfaces  = "system.interfaces"
	interfaces        = "interfaces"
	Interface_Error   = "interfaces.error"
	index             = "index"
	physicalAddress   = "interface.physical.address"
	Message           = "message"
	OID_NOT_FOUND     = "no OIDs found in util.SNMPOids"
	Nil               = "nil"
)

// FetchSNMPData retrieves SNMP data from the specified device using provided IP, community, and SNMP version.
// The function returns a map containing system information and interface details or an error description.
// It initializes an SNMP connection, fetches system-level data, extracts interface count, and retrieves interface details.
func FetchSNMPData(reqData map[string]interface{}) {

	if ValidateRequest(reqData) {

		reqData[Errors] = FieldMissing

		reqData[Status] = Fail

		return
	}

	ip := reqData[IP]

	community := reqData[Community]

	version := reqData[Version]

	g := &gosnmp.GoSNMP{
		Target:    ip.(string),
		Port:      161,
		Community: community.(string),
		Timeout:   time.Millisecond * 500,
		Retries:   0,
	}

	switch version {

	case "1":
		g.Version = gosnmp.Version1

	case "2", "2c":
		g.Version = gosnmp.Version2c

	case "3":
		g.Version = gosnmp.Version3

	default:

		reqData[Errors] = UnsupportedSNMP

		reqData[Status] = Fail

		return
	}

	err := g.Connect()

	if err != nil {

		reqData[Data] = map[string]interface{}{

			Errors: SNMPConnectFail,

			"message": err.Error(),
		}

		reqData[Status] = Fail

		return

	}

	defer g.Conn.Close()

	systemData, err := fetchSNMPSystemData(g)

	if err != nil {

		reqData[Data] = map[string]interface{}{

			Errors: SNMPConnectFail,

			Message: err.Error(),
		}

		reqData[Status] = Success

		return
	}

	numberOfInterface, _ := systemData[systemInterfaces].(int)

	data := make(map[string]interface{})

	data[systemName] = systemData[systemName]

	data[systemDescription] = systemData[systemDescription]

	data[systemLocation] = systemData[systemLocation]

	data[systemObjectID] = systemData[systemObjectID]

	uptime := systemData[systemUptime].(uint32)

	uptimeSeconds := uptime / 100

	days := uptimeSeconds / (24 * 3600)

	uptimeSeconds %= (24 * 3600)

	hours := uptimeSeconds / 3600

	uptimeSeconds %= 3600

	minutes := uptimeSeconds / 60

	seconds := uptimeSeconds % 60

	uptimeString := fmt.Sprintf("Uptime: %d days, %02d hours, %02d minutes, %02d seconds", days, hours, minutes, seconds)

	data[systemUptime] = uptimeString

	data[systemInterfaces] = systemData[systemInterfaces]

	if numberOfInterface > 0 {

		interfacesData, err := getInterfaces(g, numberOfInterface)

		if err != nil {

			data[Interface_Error] = fmt.Sprintf("Error fetching interface data: %s", err)

		} else {

			data[interfaces] = interfacesData

		}
	} else {

		data[interfaces] = []map[string]string{}

	}

	reqData[Data] = data

	reqData[Status] = Success

	return
}

// fetchSNMPSystemData queries SNMP system data using provided GoSNMP instance and configured OIDs.
// It returns a map of SNMP data with human-readable keys or an error if the operation fails.
func fetchSNMPSystemData(g *gosnmp.GoSNMP) (map[string]interface{}, error) {

	snmpData := make(map[string]interface{})

	oidArray := []string{}

	for oid := range util.SNMPOids {

		oidArray = append(oidArray, oid)

	}

	if len(oidArray) == 0 {

		return nil, fmt.Errorf(OID_NOT_FOUND)

	}

	result, err := g.Get(oidArray)

	if err != nil {

		return nil, err

	}

	for i, variable := range result.Variables {

		var valueStr interface{}

		if variable.Value == nil {

			valueStr = Nil

		} else {

			if bytes, ok := variable.Value.([]byte); ok {

				valueStr = string(bytes)

			} else {

				valueStr = variable.Value
			}

		}

		oidKey := oidArray[i]

		snmpData[util.SNMPOids[oidKey]] = valueStr
	}
	return snmpData, nil
}

// getInterface retrieves SNMP interface data for a specific index and sends it to the provided channel.
// It queries OIDs for the interface defined in the configuration and processes the SNMP responses.
// Parameters include the interface index (i), an SNMP client (g), a channel for sending interface data (ch),
// a wait group for synchronization (wg), and a mutex for threading-safe SNMP operations (mutexForSnmp).
func getInterface(i int, g *gosnmp.GoSNMP) (map[string]interface{}, error) {

	interfaceData := make(map[string]interface{})

	interfaceData[index] = fmt.Sprintf("%d", i)

	var oids []string

	var fields []string

	for oid, field := range util.InterfaceOids {

		oids = append(oids, oid+"."+fmt.Sprintf("%d", i))

		fields = append(fields, field)
	}

	result, err := g.Get(oids)

	if err != nil {

		log.Printf("Failed to fetch data for interface %d: %s", i, err)

		interfaceData[Interface_Error] = fmt.Sprintf("Failed to fetch data: %s", err)

		return interfaceData, err
	}

	for k, variable := range result.Variables {

		if variable.Value != nil {

			field := fields[k]

			var value interface{}

			if bytes, ok := variable.Value.([]byte); ok {

				if field == physicalAddress {

					mac := hex.EncodeToString(bytes)

					formattedMac := ""

					for i := 0; i < len(mac); i += 2 {

						formattedMac += strings.ToUpper(mac[i:i+2]) + " "

					}

					value = strings.TrimSpace(formattedMac)

				} else {

					value = string(bytes)
				}

			} else {

				value = variable.Value
			}

			interfaceData[field] = value
		}
	}

	return interfaceData, nil

}

// getInterfaces retrieves SNMP data for a specified number of interfaces using a GoSNMP client.
// It spawns concurrent goroutines for each interface and collects data into a slice of maps.
// Returns the slice of maps containing interface data or an error if the operation fails.
func getInterfaces(g *gosnmp.GoSNMP, numInterfaces int) ([]map[string]interface{}, error) {

	interfacesData := make([]map[string]interface{}, 0, numInterfaces)

	for i := 1; i <= numInterfaces; i++ {

		data, err := getInterface(i, g)

		if err != nil {
			return nil, err
		}
		interfacesData = append(interfacesData, data)
	}
	return interfacesData, nil
}
