package snmp

import (
	"GO_Plugin/config"
	"encoding/hex"
	"fmt"
	"github.com/gosnmp/gosnmp"
	"log"
	"strconv"
	"sync"
	"time"
)

var (
	mutexForSnmp sync.Mutex
	mutexForData sync.Mutex
)

/*
FetchSNMPData collects SNMP data from a specified device and returns it as a map containing system and interface details.
Parameters: ip - IP address of the target device, community - SNMP community string, version - SNMP protocol version.
Returns: A map[string]interface{} containing system information and interface data, or an error message on failure.
*/
func FetchSNMPData(ip, community, version string) map[string]interface{} {

	g := &gosnmp.GoSNMP{
		Target:    ip,
		Community: community,
		Port:      161,
		Timeout:   2 * time.Second,
		Retries:   2,
	}

	switch version {
	case "1":
		g.Version = gosnmp.Version1
	case "2", "2c":
		g.Version = gosnmp.Version2c
	case "3":
		g.Version = gosnmp.Version3
	default:
		log.Printf("Unsupported SNMP version: %s. Defaulting to Version2c", version)
		g.Version = gosnmp.Version2c
	}

	err := g.Connect()

	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Error connecting to SNMP target: %s", err)}
	}
	defer g.Conn.Close()

	// Fetch SNMP system data
	systemData, err := FetchSNMPSystemData(g)
	if err != nil {
		return map[string]interface{}{"error": fmt.Sprintf("Error fetching SNMP system data: %s", err)}
	}

	//get number of interface and convert to int
	str := systemData["system.interfaces"]
	numberOfInterface, err := strconv.Atoi(str)
	if err != nil {
		log.Printf("Error converting string to int: %s", err)
	}

	// Store system data in the map
	data := make(map[string]interface{})
	data["system.name"] = systemData["system.name"]
	data["system.description"] = systemData["system.description"]
	data["system.location"] = systemData["system.location"]
	data["system.objectId"] = systemData["system.objectId"]
	data["system.uptime"] = systemData["system.uptime"]
	data["system.interfaces"] = systemData["system.interfaces"]

	// Fetch interface data
	if numberOfInterface > 0 {
		interfacesData, err := GetInterfaces(g, numberOfInterface)
		if err != nil {
			data["interfaces.error"] = fmt.Sprintf("Error fetching interface data: %s", err)
		} else {
			data["interfaces"] = interfacesData
		}
	} else {
		data["interfaces"] = []map[string]string{}
	}

	return data
}

/*
FetchSNMPSystemData retrieves SNMP system data using the provided GoSNMP client and returns it as a map of key-value pairs.
It processes predefined OIDs from the configuration and maps them to corresponding system data fields.
Returns a map containing the system data or an error if the SNMP operation fails.
*/
func FetchSNMPSystemData(g *gosnmp.GoSNMP) (map[string]string, error) {

	snmpData := make(map[string]string)

	// Initialize oidArray with the OIDs from config
	oidArray := []string{}
	for oid := range config.SNMPOids {
		oidArray = append(oidArray, oid)
	}

	// Check if oidArray is empty
	if len(oidArray) == 0 {
		return nil, fmt.Errorf("no OIDs found in config.SNMPOids")
	}

	// Fetch SNMP data
	result, err := g.Get(oidArray)
	if err != nil {
		return nil, err
	}

	// Process result and convert interface{} to string
	for i, variable := range result.Variables {
		var valueStr string

		// Check if variable.Value is nil or empty
		if variable.Value == nil {
			valueStr = "nil"
		} else {
			// Convert variable.Value to string using type assertion or fmt.Sprint
			switch v := variable.Value.(type) {
			case string:
				valueStr = v
			case int:
				valueStr = fmt.Sprintf("%d", v)
			case uint:
				valueStr = fmt.Sprintf("%d", v)
			case float64:
				valueStr = fmt.Sprintf("%f", v)
			case []byte: // Handle byte slices directly
				valueStr = string(v) // Convert byte slice to string
			case []int: // Handle int slices (byte array representation)
				byteSlice := make([]byte, len(v))
				for j, b := range v {
					byteSlice[j] = byte(b) // Convert each int to byte
				}
				valueStr = string(byteSlice) // Convert byte slice to string
			default:
				valueStr = fmt.Sprintf("%v", v) // Fallback for other types
			}
		}

		// Get the corresponding key from config.SNMPOids using the index
		oidKey := oidArray[i]
		snmpData[config.SNMPOids[oidKey]] = valueStr // Use the mapped key
	}

	return snmpData, nil
}

/*
getInterface fetches SNMP data for a specific interface and appends the results to a shared slice of interface data.
It supports retry logic for SNMP requests and processes SNMP variable results based on predefined OIDs and fields.
The function uses synchronization mechanisms to ensure thread safety during SNMP data fetch and result storage.
*/
func getInterface(i int, g *gosnmp.GoSNMP, interfacesData *[]map[string]string, wg *sync.WaitGroup) {

	defer wg.Done()

	interfaceData := make(map[string]string)

	interfaceData["Index"] = fmt.Sprintf("%d", i)

	// Prepare a slice of OIDs for the current interface index
	var oids []string
	var fields []string // Keep track of the field names
	for oid, field := range config.InterfaceOids {

		oids = append(oids, oid+"."+fmt.Sprintf("%d", i))

		fields = append(fields, field) // Corresponding field for each OID
	}

	// Retry logic: Attempt SNMP request up to 2 times
	var result *gosnmp.SnmpPacket
	var err error

	mutexForSnmp.Lock()
	result, err = g.Get(oids)
	mutexForSnmp.Unlock()

	if err != nil {
		log.Printf("Failed  to fetch data for interface : %s", err)

		interfaceData["Error"] = fmt.Sprintf("Failed to fetch data : %s", err)

		mutexForData.Lock()

		*interfacesData = append(*interfacesData, interfaceData)

		mutexForData.Unlock()

		return
	}

	// Process the SNMP results
	for k, variable := range result.Variables {

		if variable.Value != nil {
			field := fields[k] // Get the corresponding field name

			var value string

			switch v := variable.Value.(type) {
			case []byte:
				// Convert byte array to hexadecimal string for PhysicalAddress
				if field == "PhysicalAddress" {
					value = hex.EncodeToString(v)
				} else {
					// Otherwise, treat as a normal string
					value = string(v)
				}
			default:
				// Convert other types to string
				value = fmt.Sprintf("%v", v)
			}

			interfaceData[field] = value
		}
	}

	// Append the collected data for this interface
	mutexForData.Lock()
	*interfacesData = append(*interfacesData, interfaceData)
	mutexForData.Unlock()
}

/*
GetInterfaces retrieves SNMP data for a specified number of interfaces from the given SNMP connection.
It returns a slice of maps containing key-value pairs for interface properties or an error if any occurs.
*/
func GetInterfaces(g *gosnmp.GoSNMP, numInterfaces int) ([]map[string]string, error) {
	interfacesData := make([]map[string]string, 0, numInterfaces)

	wg := sync.WaitGroup{}
	// Loop over interface indexes
	for i := 1; i <= numInterfaces; i++ {
		wg.Add(1)
		go getInterface(i, g, &interfacesData, &wg)
	}

	wg.Wait()

	return interfacesData, nil
}
