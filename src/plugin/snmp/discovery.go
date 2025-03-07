package snmp

import (
	"github.com/gosnmp/gosnmp"
	"log"
	"time"
)

// Constants for keys and messages
const (
	IP                 = "ip"
	PluginType         = "pluginType"
	RequestID          = "requestID"
	Community          = "community"
	Version            = "version"
	Status             = "status"
	Data               = "data"
	SystemName         = "systemName"
	SNMPPlugin         = "snmp"
	Fail               = "fail"
	Success            = "success"
	FieldMissing       = "field missing"
	UnsupportedPlugin  = "unsupported plugin type"
	UnsupportedSNMP    = "unsupported SNMP version"
	SNMPConnectFail    = "SNMP connection failed"
	SNMPGetFail        = "SNMP get request failed"
	SystemNameNotFound = "system name not found"
	SNMPConnectMsg     = "Connecting to SNMP device at %s"
	SNMPGetMsg         = "Performing SNMP GET request on %s"
	SysemNameOid       = "1.3.6.1.2.1.1.5.0"
	Error_             = "error"
)

func Discovery(reqData map[string]interface{}) {
	if ValidateRequest(reqData) {
		reqData[Error_] = FieldMissing

		reqData[Status] = Fail

		return
	}

	if reqData[PluginType] != SNMPPlugin {
		reqData[Error_] = UnsupportedPlugin

		reqData[Status] = Fail

		log.Println(UnsupportedPlugin)

		return
	}

	ip := reqData[IP].(string)

	community := reqData[Community].(string)

	version := reqData[Version].(string)

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
		reqData[Error_] = UnsupportedSNMP

		reqData[Status] = Fail

		log.Println(UnsupportedSNMP)

		return
	}

	log.Printf(SNMPConnectMsg, ip)

	err := snmp.Connect()

	if err != nil {
		reqData[Error_] = SNMPConnectFail

		reqData[Status] = Fail

		log.Println(SNMPConnectFail)

		return
	}

	defer snmp.Conn.Close()

	oid := SysemNameOid

	log.Printf(SNMPGetMsg, oid)

	result, err := snmp.Get([]string{oid})

	if err != nil {
		reqData[Error_] = SNMPGetFail

		reqData[Status] = Fail

		log.Println(SNMPGetFail)

		return
	}

	for _, variable := range result.Variables {
		if variable.Type == gosnmp.OctetString {
			reqData[Data] = map[string]interface{}{SystemName: string(variable.Value.([]byte))}

			reqData[Status] = Success

			return
		}
	}

	reqData[Error_] = SystemNameNotFound

	reqData[Status] = Fail

	log.Println(SystemNameNotFound)
}

func ValidateRequest(reqData map[string]interface{}) bool {

	hasMissingFields := false

	for _, field := range []string{IP, PluginType, RequestID} {

		value, ok := reqData[field]

		if !ok || value == "" {

			hasMissingFields = true
		}
	}

	return !hasMissingFields
}
