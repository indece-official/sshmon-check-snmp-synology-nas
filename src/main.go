package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/gosnmp/gosnmp"
	"github.com/miekg/dns"
)

// Variables set during build
var (
	ProjectName  string
	BuildVersion string
	BuildDate    string
)

var statusMap = []string{
	"OK",
	"WARN",
	"CRIT",
	"UNKNOWN",
}

const (
	OIDSystemStatus = ".1.3.6.1.4.1.6574.1.1.0"
	OIDPowerStatus  = ".1.3.6.1.4.1.6574.1.3.0"
	OIDModelName    = ".1.3.6.1.4.1.6574.1.5.1.0"
	OIDVersion      = ".1.3.6.1.4.1.6574.1.5.3.0"
	OIDDiskStatuses = ".1.3.6.1.4.1.6574.2.1.1.5"
	OIDRaidStatuses = ".1.3.6.1.4.1.6574.3.1.1.3"
)

func max(statusA int, statusB int) int {
	if statusA > statusB {
		return statusA
	}

	return statusB
}

var (
	flagVersion   = flag.Bool("v", false, "Print the version info and exit")
	flagService   = flag.String("service", "", "Service name (defaults to SynologyNAS_<host>)")
	flagHost      = flag.String("host", "", "Host")
	flagPort      = flag.Int("port", 161, "Port")
	flagCommunity = flag.String("community", "public", "Community")
	flagDNS       = flag.String("dns", "", "Use alternate dns server")
)

func containsInt(arr []int, val int) bool {
	for _, v := range arr {
		if v == val {
			return true
		}
	}

	return false
}

type ValueMap map[string]gosnmp.SnmpPDU

func (v ValueMap) CheckInt(name string, oid string, expectedValues []int) error {
	val, ok := v[oid]
	if !ok {
		return fmt.Errorf("No value for '%s' received", name)
	}

	if val.Type != gosnmp.Integer {
		return fmt.Errorf("Invalid type for '%s': received %s, expected Integer", name, val.Type.String())
	}

	if !containsInt(expectedValues, val.Value.(int)) {
		return fmt.Errorf("Invalid value for '%s': received %v, expected one of %v", name, val.Value, expectedValues)
	}

	return nil
}

func (v ValueMap) CheckMultipleInt(name string, oidPrefix string, expectedValues []int) (int, error) {
	i := 0

	for oid, val := range v {
		if !strings.HasPrefix(oid, oidPrefix) {
			continue
		}

		if val.Type == gosnmp.NoSuchObject {
			continue
		}

		if val.Type != gosnmp.Integer {
			return 0, fmt.Errorf("Invalid type for '%s %d': received %s, expected Integer", name, i, val.Type.String())
		}

		if !containsInt(expectedValues, val.Value.(int)) {
			return 0, fmt.Errorf("Invalid value for '%s %d': received %v, expected one of %v", name, i, val.Value, expectedValues)
		}

		i++
	}

	return i, nil
}

func (v ValueMap) GetString(name string) string {
	val, ok := v[name]
	if !ok {
		return ""
	}

	if val.Type != gosnmp.OctetString {
		return ""
	}

	return string(val.Value.([]byte))
}

func resolveDNS(host string) (string, error) {
	c := dns.Client{}
	m := dns.Msg{}

	m.SetQuestion(host+".", dns.TypeA)

	r, _, err := c.Exchange(&m, *flagDNS)
	if err != nil {
		return "", fmt.Errorf("Can't resolve '%s' on %s: %s", host, *flagDNS, err)
	}

	if len(r.Answer) == 0 {
		return "", fmt.Errorf("Can't resolve '%s' on %s: No results", host, *flagDNS)
	}

	aRecord := r.Answer[0].(*dns.A)

	return aRecord.A.String(), nil
}

func main() {
	var err error

	flag.Parse()

	if *flagVersion {
		fmt.Printf("%s %s (Build %s)\n", ProjectName, BuildVersion, BuildDate)
		fmt.Printf("\n")
		fmt.Printf("https://github.com/indece-official/sshmon-check-snmp-synology-nas\n")
		fmt.Printf("\n")
		fmt.Printf("Copyright 2020 by indece UG (haftungsbeschränkt)\n")

		os.Exit(0)

		return
	}

	serviceName := *flagService
	if serviceName == "" {
		serviceName = fmt.Sprintf("SynologyNAS_%s", *flagHost)
	}

	host := *flagHost
	if *flagDNS != "" {
		host, err = resolveDNS(host)
		fmt.Printf(
			"2 %s - %s - Error connecting via SNMP to '%s': %s\n",
			serviceName,
			statusMap[2],
			*flagHost,
			err,
		)

		os.Exit(0)

		return
	}

	gosnmp.Default.Target = *flagHost
	gosnmp.Default.Port = uint16(*flagPort)
	gosnmp.Default.Community = *flagCommunity
	gosnmp.Default.Version = gosnmp.Version2c
	err = gosnmp.Default.Connect()
	if err != nil {
		fmt.Printf(
			"2 %s - %s - Error connecting via SNMP to '%s': %s\n",
			serviceName,
			statusMap[2],
			*flagHost,
			err,
		)

		os.Exit(0)

		return
	}
	defer gosnmp.Default.Conn.Close()

	valueMap := ValueMap{}

	result, err := gosnmp.Default.Get([]string{
		OIDSystemStatus,
		OIDPowerStatus,
		OIDModelName,
		OIDVersion,
	})
	if err != nil {
		fmt.Printf(
			"2 %s - %s - Error reading System-OIDs via SNMP from '%s': %s\n",
			serviceName,
			statusMap[2],
			*flagHost,
			err,
		)

		os.Exit(0)

		return
	}
	for _, variable := range result.Variables {
		valueMap[variable.Name] = variable
	}

	variables, err := gosnmp.Default.WalkAll(OIDDiskStatuses)
	if err != nil {
		fmt.Printf(
			"2 %s - %s - Error reading Disk-OIDs via SNMP from '%s': %s\n",
			serviceName,
			statusMap[2],
			*flagHost,
			err,
		)

		os.Exit(0)

		return
	}
	for _, variable := range variables {
		valueMap[variable.Name] = variable
	}

	variables, err = gosnmp.Default.WalkAll(OIDRaidStatuses)
	if err != nil {
		fmt.Printf(
			"2 %s - %s - Error reading Raid-OIDs via SNMP from '%s': %s\n",
			serviceName,
			statusMap[2],
			*flagHost,
			err,
		)

		os.Exit(0)

		return
	}
	for _, variable := range variables {
		valueMap[variable.Name] = variable
	}

	status := 0
	errMsgs := []string{}

	// systemStatus Integer
	//   Normal(1)
	//   Failed(2)
	err = valueMap.CheckInt(
		"System Status",
		OIDSystemStatus,
		[]int{1},
	)
	if err != nil {
		status = max(status, 2)
		errMsgs = append(errMsgs, err.Error())
	}

	// powerStatus Integer
	//   Normal(1)
	//   Failed(2)
	err = valueMap.CheckInt(
		"Power Status",
		OIDPowerStatus,
		[]int{1},
	)
	if err != nil {
		status = max(status, 2)
		errMsgs = append(errMsgs, err.Error())
	}

	// diskStatus Integer
	//   Normal(1)
	//   Initialized(2)
	//   NotInitialized(3)
	//   SystemPartitionFailed(4)
	//   Crashed(5)
	countDisks, err := valueMap.CheckMultipleInt(
		"Disk Status",
		OIDDiskStatuses,
		[]int{1, 2, 3},
	)
	if err != nil {
		status = max(status, 2)
		errMsgs = append(errMsgs, err.Error())
	}

	// raidStatus Integer
	//   Normal(1)
	//   Repairing(2)
	//   Migrating(3)
	//   Expanding(4)
	//   Deleting(5)
	//   Creating(6)
	//   RaidSyncing(7)
	//   RaidParityChecking(8)
	//   RaidAssembling(9)
	//   Canceling(10)
	//   Degrade(11)
	//   Crashed(12)
	//   DataScrubbing (13)
	//   RaidDeploying (14)
	//   RaidUnDeploying (15)
	//   RaidMountCache (16)
	//   RaidUnmountCache (17)
	//   RaidExpandingUnfinishedSHR (18)
	//   RaidConvertSHRToPool (19)
	//   RaidMigrateSHR1ToSHR2 (20)
	//   RaidUnknownStatus (21)
	countRaids, err := valueMap.CheckMultipleInt(
		"Raid Status",
		OIDRaidStatuses,
		[]int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 13, 14, 15, 16, 17, 18, 19, 20},
	)
	if err != nil {
		status = max(status, 2)
		errMsgs = append(errMsgs, err.Error())
	}

	modelName := valueMap.GetString(OIDModelName)
	version := valueMap.GetString(OIDVersion)

	errMsg := ""
	if len(errMsgs) > 0 {
		errMsg = ": " + strings.Join(errMsgs, ", ")
	}

	healthStr := "healthy"
	if status != 0 {
		healthStr = "unhealthy"
	}

	fmt.Printf(
		"%d %s - %s - Synology NAS %s (%s) on %s is %s (%d disks, %d raids)%s\n",
		status,
		serviceName,
		statusMap[status],
		modelName,
		version,
		*flagHost,
		healthStr,
		countDisks,
		countRaids,
		errMsg,
	)

	os.Exit(0)
}
