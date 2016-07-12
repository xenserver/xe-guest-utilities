package feature

import (
	"testing"
)

func TestSettingHistory(t *testing.T) {
	addSettingHistory(Configuration{Mac: "mac1", IsIPv6: true})
	addSettingHistory(Configuration{Mac: "mac1", IsIPv6: true})
	addSettingHistory(Configuration{Mac: "mac1", IsIPv6: false})
	addSettingHistory(Configuration{Mac: "mac2", IsIPv6: false})

	configurations := getSettingHistory()
	if len(configurations) != 3 {
		t.Errorf("TestSettingHistory error: history size not correct %#v\n", len(configurations))
	}

	conf := getHistoryByMac("mac2", true)
	if conf.Mac == "mac2" {
		t.Errorf("TestSettingHistory error: history of Ipv6 should not exist for mac2")
	}

	conf = getHistoryByMac("mac1", false)
	if conf.Mac != "mac1" || conf.IsIPv6 != false {
		t.Errorf("TestSettingHistory error: history of Ipv6 for mac1 does not fetch correctly")
	}

	removeHistoryByMac("mac2", true)
	conf = getHistoryByMac("mac2", false)
	if conf.Mac != "mac2" {
		t.Errorf("TestSettingHistory error: history of Ipv4 for mac2 should not been removed")
	}

	removeHistoryByMac("mac2", false)
	conf = getHistoryByMac("mac2", false)
	if conf.Mac == "mac2" {
		t.Errorf("TestSettingHistory error: history of Ipv4 for mac2 should been removed")
	}
}
