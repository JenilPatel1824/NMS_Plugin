package snmp

import (
	"GO_Plugin/src/util"
	"encoding/hex"
	"fmt"
	"github.com/gosnmp/gosnmp"
	"log"
	"strconv"
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
	Index             = "index"
	physicalAddress   = "interface.physical.address"
	Message           = "message"
	OID_NOT_FOUND     = "no OIDs found in util.SNMPOids"
	Nil               = "nil"
)

// FetchSNMPData retrieves SNMP data for a given IP and community string, storing results in reqData.
// @param reqData map[string]interface{} - Contains request parameters such as IP, community, version, and stores the response.
func FetchSNMPData(reqData map[string]interface{}) {

	if !ValidateRequest(reqData) {

		reqData[Errors] = FieldMissing

		reqData[Status] = Fail

		return
	}

	ip := reqData[IP].(string)

	community := reqData[Community].(string)

	version := reqData[Version].(string)

	g := &gosnmp.GoSNMP{
		Target:    ip,
		Port:      161,
		Community: community,
		Timeout:   time.Millisecond * 500,
		Retries:   1,
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

	if err := g.Connect(); err != nil {

		reqData[Data] = map[string]interface{}{

			Errors: SNMPConnectFail,

			Message: err.Error(),
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

	data := map[string]interface{}{

		systemName: systemData[systemName],

		systemDescription: systemData[systemDescription],

		systemLocation: systemData[systemLocation],

		systemObjectID: systemData[systemObjectID],
	}

	if uptime, ok := systemData[systemUptime].(uint32); ok {

		uptimeSeconds := uptime / 100

		days := uptimeSeconds / (24 * 3600)

		uptimeSeconds %= (24 * 3600)

		hours := uptimeSeconds / 3600

		uptimeSeconds %= 3600

		minutes := uptimeSeconds / 60

		seconds := uptimeSeconds % 60

		data[systemUptime] = fmt.Sprintf("Uptime: %d days, %02d hours, %02d minutes, %02d seconds", days, hours, minutes, seconds)
	}

	indexes, err := getInterfaceIndexes(g)

	if err != nil {

		data[Interface_Error] = fmt.Sprintf("Error fetching interface indexes: %s", err)

		data[systemInterfaces] = 0

	} else {

		data[systemInterfaces] = len(indexes)

		interfacesData, err := getInterfaces(g, indexes)

		if err != nil {

			data[Interface_Error] = fmt.Sprintf("Error fetching interface data: %s", err)

		} else {

			data[interfaces] = interfacesData
		}
	}

	reqData[Data] = data

	reqData[Status] = Success
}

// fetchSNMPSystemData retrieves system-related SNMP data using predefined OIDs.
// @param g *gosnmp.GoSNMP - SNMP client used to query the target device.
// @return map[string]interface{} - A map containing SNMP system data.
// @return error - Error if SNMP retrieval fails or no OIDs are found.
func fetchSNMPSystemData(g *gosnmp.GoSNMP) (map[string]interface{}, error) {

	snmpData := make(map[string]interface{})

	oidArray := make([]string, 0, len(util.SNMPOids))

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

		var value interface{}

		if variable.Value == nil {

			value = Nil

		} else if bytes, ok := variable.Value.([]byte); ok {

			value = string(bytes)

		} else {

			value = variable.Value
		}

		snmpData[util.SNMPOids[oidArray[i]]] = value
	}
	return snmpData, nil
}

// getInterfaceIndexes retrieves the indexes of device interfaces using SNMP walk on the specified OID.
// @param g *gosnmp.GoSNMP - SNMP client used to query the target device.
// @return []int - A list of interface indexes.
// @return error - Error if SNMP walk fails or index parsing fails.
func getInterfaceIndexes(g *gosnmp.GoSNMP) ([]int, error) {

	var indexes []int

	targetOID := ".1.3.6.1.2.1.31.1.1.1.1"

	err := g.Walk(targetOID, func(pdu gosnmp.SnmpPDU) error {

		oidParts := strings.Split(pdu.Name, ".")

		if len(oidParts) == 0 {

			return fmt.Errorf("invalid OID: %s", pdu.Name)
		}

		oidSuffix := oidParts[len(oidParts)-1]

		index, err := strconv.Atoi(oidSuffix)

		if err != nil {

			return fmt.Errorf("failed to parse index: %v", err)
		}

		indexes = append(indexes, index)

		return nil
	})

	if err != nil {

		return nil, fmt.Errorf("SNMP walk failed: %v", err)
	}

	return indexes, nil
}

// getInterfaces retrieves SNMP data for multiple network interfaces based on their indexes
// and aggregates the results. It queries each interface using its index and collects the data.
// @param g *gosnmp.GoSNMP - SNMP client used to query the target device for interface details.
// @param indexes []int - List of interface indexes to fetch SNMP data for each interface.
// @return []map[string]interface{} - A list of maps where each map contains SNMP data of an interface.
// @return error - Returns an error if SNMP data retrieval for interfaces encounters a failure.
func getInterfaces(g *gosnmp.GoSNMP, indexes []int) ([]map[string]interface{}, error) {

	interfacesData := make([]map[string]interface{}, 0, len(indexes))

	for _, index := range indexes {

		data, err := getInterface(index, g)

		if err != nil {

			log.Printf("Error fetching interface %d: %v (continuing)", index, err)

			continue
		}

		interfacesData = append(interfacesData, data)
	}
	return interfacesData, nil
}

// getInterface retrieves SNMP data for a specific network interface based on its index.
// It queries the device for interface details and formats the data accordingly.
// @param index int - The index of the network interface to fetch SNMP data for.
// @param g *gosnmp.GoSNMP - SNMP client used to query the target device.
// @return map[string]interface{} - A map containing SNMP data of the interface.
// @return error - Returns an error if SNMP data retrieval fails.
func getInterface(index int, g *gosnmp.GoSNMP) (map[string]interface{}, error) {

	interfaceData := make(map[string]interface{})

	interfaceData[Index] = fmt.Sprintf("%d", index)

	oids := make([]string, 0, len(util.InterfaceOids))

	fields := make([]string, 0, len(util.InterfaceOids))

	for oid, field := range util.InterfaceOids {

		oids = append(oids, fmt.Sprintf("%s.%d", oid, index))

		fields = append(fields, field)
	}

	result, err := g.Get(oids)

	if err != nil {

		interfaceData[Interface_Error] = fmt.Sprintf("SNMP get failed: %v", err)

		return interfaceData, err
	}

	for k, variable := range result.Variables {

		if variable.Value == nil {

			continue
		}

		field := fields[k]

		var value interface{}

		if bytes, ok := variable.Value.([]byte); ok {

			if field == physicalAddress {

				mac := hex.EncodeToString(bytes)

				formattedMac := strings.ToUpper(mac)

				if len(formattedMac) >= 12 {

					formattedMac = fmt.Sprintf("%s:%s:%s:%s:%s:%s",
						formattedMac[0:2], formattedMac[2:4],
						formattedMac[4:6], formattedMac[6:8],
						formattedMac[8:10], formattedMac[10:12])
				}

				value = formattedMac

			} else {

				value = string(bytes)
			}
		} else {

			value = variable.Value
		}

		interfaceData[field] = value
	}
	return interfaceData, nil
}
