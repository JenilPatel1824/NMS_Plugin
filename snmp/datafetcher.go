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

const (
	systemName        = "system.name"
	systemDescription = "system.description"
	systemLocation    = "system.location"
	systemObjectID    = "system.objectId"
	systemUptime      = "system.uptime"
	systemInterfaces  = "system.interfaces"
	interfaces        = "interfaces"
	Error             = "interfaces.error"
	indexKey          = "index"
)

// FetchSNMPData retrieves SNMP data from the specified device using provided IP, community, and SNMP version.
// The function returns a map containing system information and interface details or an error description.
// It initializes an SNMP connection, fetches system-level data, extracts interface count, and retrieves interface details.
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

	systemData, err := fetchSNMPSystemData(g)

	if err != nil {

		return map[string]interface{}{"error": fmt.Sprintf("Error fetching SNMP system data: %s", err)}
	}

	str := systemData[systemInterfaces]

	numberOfInterface, err := strconv.Atoi(str)

	if err != nil {

		log.Printf("Error converting string to int: %s", err)
	}

	data := make(map[string]interface{})

	data[systemName] = systemData[systemName]

	data[systemDescription] = systemData[systemDescription]

	data[systemLocation] = systemData[systemLocation]

	data[systemObjectID] = systemData[systemObjectID]

	data[systemUptime] = systemData[systemUptime]

	data[systemInterfaces] = systemData[systemInterfaces]

	if numberOfInterface > 0 {

		interfacesData, err := getInterfaces(g, numberOfInterface)

		if err != nil {

			data[Error] = fmt.Sprintf("Error fetching interface data: %s", err)

		} else {

			data[interfaces] = interfacesData

		}
	} else {

		data[interfaces] = []map[string]string{}

	}
	return data
}

// fetchSNMPSystemData queries SNMP system data using provided GoSNMP instance and configured OIDs.
// It returns a map of SNMP data with human-readable keys or an error if the operation fails.
func fetchSNMPSystemData(g *gosnmp.GoSNMP) (map[string]string, error) {

	snmpData := make(map[string]string)

	oidArray := []string{}

	for oid := range config.SNMPOids {

		oidArray = append(oidArray, oid)

	}

	if len(oidArray) == 0 {

		return nil, fmt.Errorf("no OIDs found in config.SNMPOids")

	}

	result, err := g.Get(oidArray)

	if err != nil {

		return nil, err

	}

	for i, variable := range result.Variables {

		var valueStr string

		if variable.Value == nil {

			valueStr = "nil"

		} else {

			switch v := variable.Value.(type) {

			case string:
				valueStr = v

			case int:
				valueStr = fmt.Sprintf("%d", v)

			case uint:
				valueStr = fmt.Sprintf("%d", v)

			case float64:
				valueStr = fmt.Sprintf("%f", v)

			case []byte:
				valueStr = string(v)

			case []int:
				byteSlice := make([]byte, len(v))

				for j, b := range v {

					byteSlice[j] = byte(b)

				}

				valueStr = string(byteSlice)

			default:
				valueStr = fmt.Sprintf("%v", v)
			}
		}

		oidKey := oidArray[i]

		snmpData[config.SNMPOids[oidKey]] = valueStr
	}
	return snmpData, nil
}

// getInterface retrieves SNMP interface data for a specific index and sends it to the provided channel.
// It queries OIDs for the interface defined in the configuration and processes the SNMP responses.
// Parameters include the interface index (i), an SNMP client (g), a channel for sending interface data (ch),
// a wait group for synchronization (wg), and a mutex for threading-safe SNMP operations (mutexForSnmp).
func getInterface(i int, g *gosnmp.GoSNMP, ch chan<- map[string]string, wg *sync.WaitGroup, mutexForSnmp *sync.Mutex) {

	defer wg.Done()

	interfaceData := make(map[string]string)

	interfaceData[indexKey] = fmt.Sprintf("%d", i)

	var oids []string

	var fields []string

	for oid, field := range config.InterfaceOids {

		oids = append(oids, oid+"."+fmt.Sprintf("%d", i))

		fields = append(fields, field)
	}

	mutexForSnmp.Lock()
	result, err := g.Get(oids)
	mutexForSnmp.Unlock()

	if err != nil {

		log.Printf("Failed to fetch data for interface %d: %s", i, err)

		interfaceData[Error] = fmt.Sprintf("Failed to fetch data: %s", err)

		ch <- interfaceData

		return
	}

	for k, variable := range result.Variables {

		if variable.Value != nil {

			field := fields[k]

			var value string

			switch v := variable.Value.(type) {

			case []byte:
				if field == "PhysicalAddress" {

					value = hex.EncodeToString(v)

				} else {

					value = string(v)
				}

			default:
				value = fmt.Sprintf("%v", v)
			}
			interfaceData[field] = value
		}
	}
	ch <- interfaceData
}

// getInterfaces retrieves SNMP data for a specified number of interfaces using a GoSNMP client.
// It spawns concurrent goroutines for each interface and collects data into a slice of maps.
// Returns the slice of maps containing interface data or an error if the operation fails.
func getInterfaces(g *gosnmp.GoSNMP, numInterfaces int) ([]map[string]string, error) {

	interfacesData := make([]map[string]string, 0, numInterfaces)

	var snmp sync.Mutex

	ch := make(chan map[string]string, numInterfaces)

	wg := sync.WaitGroup{}

	for i := 1; i <= numInterfaces; i++ {

		wg.Add(1)

		go getInterface(i, g, ch, &wg, &snmp)
	}

	go func() {

		wg.Wait()

		close(ch)
	}()

	for data := range ch {

		interfacesData = append(interfacesData, data)
	}
	return interfacesData, nil
}
