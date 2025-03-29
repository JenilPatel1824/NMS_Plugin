package snmp

import (
	"github.com/gosnmp/gosnmp"
	"log"
	"strings"
	"time"
)

// Constants for keys and messages
const (
	IP                 = "ip"
	PluginType         = "pluginType"
	RequestID          = "requestId"
	Community          = "community"
	Version            = "version"
	Status             = "status"
	Data               = "data"
	SystemName         = "systemName"
	SNMPPlugin         = "snmp"
	Fail               = "fail"
	Success            = "success"
	UnsupportedPlugin  = "unsupported plugin type"
	UnsupportedSNMP    = "unsupported SNMP version"
	SNMPConnectFail    = "SNMP connection failed"
	SNMPGetFail        = "SNMP get request failed"
	SystemNameNotFound = "system name not found"
	SNMPConnectMsg     = "Connecting to SNMP device at %s"
	SNMPGetMsg         = "Performing SNMP GET request on %s"
	SysemNameOid       = "1.3.6.1.2.1.1.5.0"
	Errors             = "error"
	Port               = "port"
)

// Discovery performs SNMP discovery for a given network device.
// It validates the request, establishes an SNMP connection, and retrieves system information.
// @param reqData map[string]interface{} - A map containing request data including IP, community, and SNMP version.
// If validation fails, error messages and status updates are stored in reqData.
func Discovery(reqData map[string]interface{}) {

	if reqData[PluginType] != SNMPPlugin {

		reqData[Errors] = UnsupportedPlugin

		reqData[Status] = Fail

		log.Println(UnsupportedPlugin)

		return
	}

	snmp := &gosnmp.GoSNMP{
		Target:    reqData[IP].(string),
		Port:      uint16(reqData[Port].(float64)),
		Community: reqData[Community].(string),
		Timeout:   time.Millisecond * 500,
		Retries:   0,
	}

	switch reqData[Version].(string) {

	case "1":
		snmp.Version = gosnmp.Version1

	case "2", "2c":
		snmp.Version = gosnmp.Version2c

	case "3":
		snmp.Version = gosnmp.Version3

	default:
		reqData[Errors] = UnsupportedSNMP

		reqData[Status] = Fail

		log.Println(UnsupportedSNMP)

		return
	}

	log.Printf(SNMPConnectMsg, reqData[IP].(string))

	err := snmp.Connect()

	if err != nil {

		reqData[Errors] = SNMPConnectFail

		reqData[Status] = Fail

		log.Println(SNMPConnectFail)

		return
	}

	defer snmp.Conn.Close()

	oid := SysemNameOid

	log.Printf(SNMPGetMsg, oid)

	result, err := snmp.Get([]string{oid})

	if err != nil {

		reqData[Errors] = SNMPGetFail

		reqData[Status] = Fail

		return
	}

	for _, variable := range result.Variables {

		if variable.Type == gosnmp.OctetString {

			reqData[Data] = map[string]interface{}{SystemName: string(variable.Value.([]byte))}

			reqData[Status] = Success

			return
		}
	}

	reqData[Errors] = SystemNameNotFound

	reqData[Status] = Fail

	log.Println(SystemNameNotFound)
}

// ValidateRequest checks whether the required fields are present in the request data.
// @param reqData map[string]interface{} - The request data containing key-value pairs.
// @return bool - Returns true if all required fields are present, otherwise false.
func ValidateRequest(reqData map[string]interface{}) bool {

	requiredFields := []string{IP, PluginType, RequestID, Port}

	for _, field := range requiredFields {

		value, exists := reqData[field]

		if !exists {

			return false
		}

		if field == Port {

			if v, ok := value.(float64); ok {

				reqData[field] = int(v)
			}
			continue
		}

		Value, ok := value.(string)

		if !ok || strings.TrimSpace(Value) == "" {

			return false
		}
	}

	return true
}
