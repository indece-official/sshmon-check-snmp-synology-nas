# sshmon-check-snmp-synology-nas
Nagios/Checkmk-compatible SSHMon-check for Synology-NAS-Servers via SNMP

## Installation
* Download [latest Release](https://github.com/indece-official/sshmon-check-snmp-synology-nas/releases/latest)
* Move binary to `/usr/local/bin/sshmon_check_snmp_synology_nas`


## Usage
```
$> sshmon_check_snmp_synology_nas -host 192.168.178.20 -community mycommunity
```

```
Usage of sshmon_check_snmp_synology_nas:
  -community string
        Community (default "public")
  -dns string
        Use alternate dns server
  -host string
        Host
  -port int
        Port (default 161)
  -service string
        Service name (defaults to SynologyNAS_<host>)
  -v    Print the version info and exit
```

Output:
```
0 SynologyNAS_192.168.11.20 - OK - Synology NAS DS720+ (DSM 6.2-25426) on 192.168.11.20 is healthy (2 disks, 1 raids)
```

### Tested Synology-NAS-Servers
| Model | Version |
| --- | --- |
| DS720+ | DSM 6.2-25426 |

## Development
### Snapshot build

```
$> make --always-make
```

### Release build

```
$> BUILD_VERSION=1.0.0 make --always-make
```