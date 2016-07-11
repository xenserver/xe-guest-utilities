package feature

import (
	syslog "../syslog"
	xenstoreclient "../xenstoreclient"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
	"time"
)

type FeatureIPSettingClient interface {
	Run() error
}

type FeatureIPSetting struct {
	Client  xenstoreclient.XenStoreClient
	Enabled bool
	Debug   bool
	logger  *log.Logger
}

const (
	configFile = "./ipconf.json"

	advertiseKey      = "control/feature-static-ip-setting"
	controlKey        = "xenserver/device/vif"
	token             = "FeatureIPSetting"
	macSubKey         = "/static-ip-setting/mac"
	ipenabledSubKey   = "/static-ip-setting/enabled"
	ipv6enabledSubKey = "/static-ip-setting/enabled6"
	errorCodeSubKey   = "/static-ip-setting/error-code"
	errorMsgSubKey    = "/static-ip-setting/error-msg"
	addressSubKey     = "/static-ip-setting/address"
	gatewaySubKey     = "/static-ip-setting/gateway"
	address6SubKey    = "/static-ip-setting/address6"
	gateway6SubKey    = "/static-ip-setting/gateway6"
)

const (
	LoggerName string = "FeatureIPSetting"
)

func NewFeatureIPSetting(Client xenstoreclient.XenStoreClient, Enabled bool, Debug bool) (FeatureIPSettingClient, error) {
	var loggerWriter io.Writer = os.Stderr
	var topic string = LoggerName
	if w, err := syslog.NewSyslogWriter(topic); err == nil {
		loggerWriter = w
		topic = ""
	} else {
		fmt.Fprintf(os.Stderr, "NewSyslogWriter(%s) error: %s, use stderr logging\n", topic, err)
		topic = LoggerName + ": "
	}
	logger := log.New(loggerWriter, topic, 0)

	return &FeatureIPSetting{
		Client:  Client,
		Enabled: Enabled,
		Debug:   Debug,
		logger:  logger,
	}, nil
}

func (f *FeatureIPSetting) Enable() {
	if f.Enabled {
		f.Client.Write(advertiseKey, "1")
	} else {
		f.Client.Write(advertiseKey, "0")
	}
	return
}

func (f *FeatureIPSetting) GetChildrens(key string) []string {
	var childrens []string
	value, err := f.Client.Directory(controlKey)
	if err != nil {
		f.logger.Printf("GetChildrens failed %#v\n", err)
	} else {
		subkeys := strings.Split(string(value), "\x00")
		for _, subkey := range subkeys {
			if len(subkey) != 0 {
				childrens = append(childrens, controlKey+"/"+subkey)
			}
		}
	}
	return childrens
}

type OSType int

const (
	OTHER  OSType = 0
	CENTOS OSType = 1
)

func GetCurrentOSType() OSType {
	distributionFile, err := os.OpenFile("/var/cache/xe-linux-distribution", os.O_RDONLY, 0666)
	if err != nil {
		return OTHER
	}
	defer distributionFile.Close()
	scanner := bufio.NewScanner(distributionFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			k := strings.TrimSpace(parts[0])
			v := strings.TrimSpace(strings.Trim(strings.TrimSpace(parts[1]), "\""))
			if k == "os_distro" && v == "centos" {
				return CENTOS
			}
		}
	}
	return OTHER
}

type Configuration struct {
	Mac     string
	IP      string
	Mask    string
	Gateway string
	IsIPv6  bool
}

func GetSettingHistory() []Configuration {
	configurations := []Configuration{}
	if file, err := os.Open(configFile); err == nil {
		defer file.Close()
		decoder := json.NewDecoder(file)
		decoder.Decode(&configurations)
	}
	return configurations

}

func AddSettingHistory(conf Configuration) {
	configurations := GetSettingHistory()
	found := false
	for _, configuration := range configurations {
		if configuration.Mac == conf.Mac && configuration.IsIPv6 == conf.IsIPv6 {
			found = true
		}
	}
	if !found {
		configurations = append(configurations, conf)
		if file, err := os.Create(configFile); err == nil {
			defer file.Close()
			encoder := json.NewEncoder(file)
			encoder.Encode(configurations)
		}
	}
}

func GetHistoryByMac(mac string, isIPv6 bool) Configuration {
	configurations := GetSettingHistory()
	for _, configuration := range configurations {
		if configuration.Mac == mac && configuration.IsIPv6 == isIPv6 {
			return configuration
		}
	}
	return Configuration{}
}

func RemoveHistoryByMac(mac string, isIPv6 bool) {
	configurations := GetSettingHistory()
	found := false
	index := 0
	for i, configuration := range configurations {
		if configuration.Mac == mac && configuration.IsIPv6 == isIPv6 {
			found = true
			index = i
		}
	}

	if found {
		configurations = append(configurations[:index], configurations[index+1:]...)
		if file, err := os.Create(configFile); err == nil {
			defer file.Close()
			encoder := json.NewEncoder(file)
			encoder.Encode(configurations)
		}
	}
}

func (f *FeatureIPSetting) UnConfigStaticIP(vifKey string, mac string, isIPv6 bool, osType OSType) error {
	f.logger.Printf("FeatureIPSetting Unset IP information for %s on OS %#v\n", vifKey, osType)

	conf := GetHistoryByMac(mac, isIPv6)
	if conf.Mac != mac && conf.IsIPv6 != isIPv6 {
		f.logger.Printf("FeatureIPSetting Unset return\n")

		return nil
	}
	// deconfig ip
	RemoveHistoryByMac(mac, isIPv6)
	return nil
}

func (f *FeatureIPSetting) ConfigStaticIP(vifKey string, mac string, isIPv6 bool, osType OSType) error {
	addressKey := vifKey + addressSubKey
	gatewatKey := vifKey + gatewaySubKey
	if isIPv6 {
		addressKey = vifKey + address6SubKey
		gatewatKey = vifKey + gateway6SubKey
	}

	if address, err := f.Client.Read(addressKey); err == nil {
		switch osType {
		case CENTOS:
			f.logger.Printf("FeatureIPSetting Set IP %s on Centos\n", address)
		default:
			f.logger.Printf("FeatureIPSetting Set IP %s on Other OS\n", address)
		}
		AddSettingHistory(Configuration{Mac: mac, IsIPv6: isIPv6})
	} else {
		f.logger.Printf("FeatureIPSetting Set IP failed with %s\n", err.Error())
	}

	if gateway, err := f.Client.Read(gatewatKey); err == nil {
		switch osType {
		case CENTOS:
			f.logger.Printf("FeatureIPSetting Set gateway with %s on Centos\n", gateway)
		default:
			f.logger.Printf("FeatureIPSetting Set gateway with %s on other OS\n", gateway)
		}
		AddSettingHistory(Configuration{Mac: mac, IsIPv6: isIPv6})
	} else {
		f.logger.Printf("FeatureIPSetting Set gateway failed with %s\n", err.Error())
	}
	return nil
}

func (f *FeatureIPSetting) Run() error {
	err := f.Client.Watch(controlKey, token)
	if err != nil {
		f.logger.Printf("Watch(\"%#v\") error: %#v\n", controlKey, err)
		return err
	}

	f.logger.Printf("Start watch on %#v\n", controlKey)
	go func() {
		osType := GetCurrentOSType()
		ticker := time.Tick(4 * time.Second)
		for {
			f.Enable()
			if _, ok := f.Client.WatchEvent(controlKey); ok {
				childrens := f.GetChildrens(controlKey)
				for _, subkey := range childrens {
					f.logger.Printf("Start checking key %s", subkey)
					macKey := subkey + macSubKey
					mac, err := f.Client.Read(macKey)
					if err != nil {
						f.logger.Printf("FeatureIPSetting get mac for %#v failed with %#v\n", macKey, err)
						continue
					}

					ipenabledKey := subkey + ipenabledSubKey
					if ipenabled, err := f.Client.Read(ipenabledKey); err == nil {
						if ipenabled == "1" {
							f.ConfigStaticIP(subkey, mac, false, osType)
						} else {
							f.UnConfigStaticIP(subkey, mac, false, osType)
						}
						if err = f.Client.Rm(ipenabledKey); err != nil {
							f.logger.Printf("FeatureIPSetting remove key %#v failed with %#v\n", ipenabledKey, err)
						}
					}

					ipv6enabledKey := subkey + ipv6enabledSubKey
					if ipv6enabled, err := f.Client.Read(ipv6enabledKey); err == nil {
						if ipv6enabled == "1" {
							f.ConfigStaticIP(subkey, mac, true, osType)
						} else {
							f.UnConfigStaticIP(subkey, mac, true, osType)
						}
						if err = f.Client.Rm(ipv6enabledKey); err != nil {
							f.logger.Printf("FeatureIPSetting remove key %#v failed with %#v\n", ipv6enabledKey, err)
						}
					}

				}
			}
			select {
			case <-ticker:
				continue
			}

		}
	}()
	return nil
}
