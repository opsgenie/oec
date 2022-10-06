# OPSGENIE EDGE CONNECTOR

[![Build Status](https://github.com/opsgenie/oec/workflows/test/badge.svg?branch=master)](https://github.com/opsgenie/oec/actions?query=workflow%3Atest)
[![Coverage Status](https://coveralls.io/repos/github/opsgenie/oec/badge.svg?branch=master)](https://coveralls.io/github/opsgenie/oec?branch=master)
[![Go Report Card](https://goreportcard.com/badge/github.com/opsgenie/oec)](https://goreportcard.com/report/github.com/opsgenie/oec)
[![GoDoc](https://godoc.org/github.com/opsgenie/oec?status.svg)](https://godoc.org/github.com/opsgenie/oec)
[![Contact Support](https://img.shields.io/badge/-Contact%20Support-blue)](https://support.atlassian.com/contact/#/)
[![Public Issue Tracker](https://img.shields.io/badge/-Public%20Issue%20Tracker-blue)](https://jira.atlassian.com/browse/OPSGENIE-803?jql=project%3DOPSGENIE%20AND%20component%20in%20(%22OEC%20-%20Configuration%22%2C%20%22OEC%20-%20Installation%22)%20and%20resolution%20is%20EMPTY)


Opsgenie Edge Connector (OEC) is a lightweight application that provides:

* Opsgenie integration for systems that don't need the inbound internet
* Ability to run executables and scripts triggered by Opsgenie
* Deployment on-premises or in the customer’s cloud environment

OEC integrates with a number of monitoring and ITSM tools, allowing Opsgenie to send actions back to keep various toolsets in sync across the organization. OEC also hosts custom scripts that can be executed remotely.

## Supported Script Technologies

OEC includes support for running Groovy, Python and Go scripts, along with any .sh shell script or executable.

OEC supports environment variables, arguments, and flags that are passed to scripts. These can be set globally for all scripts or locally on a per script basis. Stderr and stdout options are also available.

## Support for Git

OEC provides the ability to retrieve files from Git.

Configuration files for OEC can be maintained in Git to ensure version control. Likewise, scripts and credentials can be kept in Git and retrieved when needed so that credentials are not stored locally.

## Prerequisites

You need Python 3.0 or later to run [OEC scripts](https://github.com/opsgenie/oec-scripts). You can have multiple Python versions (2.x) installed on the same system without problems.

In Ubuntu based Linux distribution, you can install Python 3 like this:
```
$ sudo apt-get install python3 python3-pip
```
For other Operating systems, packages are available at
```
http://www.python.org/getit/
```

```OEC uses default Python version of your system.```

## Building OEC executable
Clone repository to: $GOPATH/src/github.com/opsgenie/oec
```
$ mkdir -p $GOPATH/src/github.com/opsgenie
$ cd $GOPATH/src/github.com/opsgenie
$ git clone git@github.com:opsgenie/oec.git
```
Enter the directory which includes main.go and build executable
```
$ cd $GOPATH/src/github.com/opsgenie/oec/main/
$ go build main.go 
```
## Configuration
### Environment Variables
####Prerequisites

For setting configuration file properties such as location and path:

* First, you should set some environment variables for the locate configuration file.
There are two options here, you can get the configuration file from a local drive or by using git.

For reading configuration files from a local drive:

* Set `OEC_CONF_SOURCE_TYPE` and `OEC_CONF_LOCAL_FILEPATH` variables.

From reading configuration files from a git repository:

* Set `OEC_CONF_SOURCE_TYPE`, `OEC_CONF_GIT_URL`, `OEC_CONF_GIT_FILEPATH`, `OEC_CONF_GIT_PRIVATE_KEY_FILEPATH`, and `OEC_CONF_GIT_PASSPHRASE` variables.

```If you are using a public repository, you should use an https format of a git url and you do not need to set private key and passphrase.```

For more information, you can visit [OEC documentation page](https://docs.opsgenie.com/docs/oec-configuration#section-environment-variables)
### Flag
Prometheus default metrics can be grabbed from `http://localhost:<port-number>/metrics`

To run multiple OEC in the same environment, -oec-metrics flag should be set as distinct port number values.
`-oec-metrics <port-number>`

### Logs
OEC log file is located:

* On Windows: `var/log/opsgenie/oec<pid>.log`
* On Linux: `/var/log/opsgenie/oec<pid>.log`
* At the end of the file name of the log, there is program identifier (pid) to identify which process is running.

### Configuration File
OEC supports json and yaml file extension with fields. 

For definition of all fields which should be provided in configuration file, you can visit [OEC documentation page](https://docs.opsgenie.com/docs/oec-configuration#section-configuration-file) 

## Running
You can run executable that you build according the building OEC executables section.
```
OEC_CONF_SOURCE_TYPE=LOCAL OEC_CONF_LOCAL_FILEPATH=$OEC_FILE_PATH ./main
```
Also you can run OEC by using Docker. For more information, please visit [documentation](https://docs.opsgenie.com/docs/oec-running)

## Contact Support
You can find open bugs and suggestions for OEC on our [public issue tracker](https://jira.atlassian.com/browse/OPSGENIE-803?jql=project%3DOPSGENIE%20AND%20component%20in%20(%22OEC%20-%20Configuration%22%2C%20%22OEC%20-%20Installation%22)%20and%20resolution%20is%20EMPTY). If you are experiencing an issue with OEC, or if you want to raise a new bug or suggestion you can reach out [Opsgenie support](https://support.atlassian.com/contact/#/).

