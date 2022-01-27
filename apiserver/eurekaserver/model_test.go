/**
 * Tencent is pleased to support the open source community by making Polaris available.
 *
 * Copyright (C) 2019 THL A29 Limited, a Tencent company. All rights reserved.
 *
 * Licensed under the BSD 3-Clause License (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://opensource.org/licenses/BSD-3-Clause
 *
 * Unless required by applicable law or agreed to in writing, software distributed
 * under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR
 * CONDITIONS OF ANY KIND, either express or implied. See the License for the
 * specific language governing permissions and limitations under the License.
 */

package eurekaserver

import (
	"encoding/xml"
	"fmt"
	"strings"
	"testing"
)

const (
	instanceXMLStr = "<application>\n" +
		"<name>XXL-JOB-EXECUTOR-PAAS</name>\n" +
		"<instance>\n" +
		"<instanceId>xxl-job-executor-b7c89dcf4-wrk7s:xxl-job-executor-paas:8081</instanceId>\n" +
		"<hostName>xxl-job-executor-b7c89dcf4-wrk7s</hostName>\n" +
		"<app>XXL-JOB-EXECUTOR-PAAS</app>\n" +
		"<ipAddr>10.157.22.100</ipAddr>\n" +
		"<status>UP</status>\n" +
		"<overriddenstatus>UNKNOWN</overriddenstatus>\n" +
		"<port enabled=\"true\">8081</port>\n" +
		"<securePort enabled=\"false\">443</securePort>\n" +
		"<countryId>1</countryId>\n" +
		"<dataCenterInfo class=\"com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo\">\n" +
		"<name>MyOwn</name>\n" +
		"</dataCenterInfo>\n" +
		"<leaseInfo>\n" +
		"<renewalIntervalInSecs>30</renewalIntervalInSecs>\n" +
		"<durationInSecs>90</durationInSecs>\n" +
		"<registrationTimestamp>1631448499517</registrationTimestamp>\n" +
		"<lastRenewalTimestamp>1632368761523</lastRenewalTimestamp>\n" +
		"<evictionTimestamp>0</evictionTimestamp>\n" +
		"<serviceUpTimestamp>1631448499517</serviceUpTimestamp>\n" +
		"</leaseInfo>\n" +
		"<metadata class=\"java.util.Collections$EmptyMap\">\n" +
		"<region>shanghai</region>\n" +
		"</metadata>\n" +
		"<homePageUrl>http://xxl-job-executor-b7c89dcf4-wrk7s:8081/</homePageUrl>\n" +
		"<statusPageUrl>http://xxl-job-executor-b7c89dcf4-wrk7s:8081/info</statusPageUrl>\n" +
		"<healthCheckUrl>http://xxl-job-executor-b7c89dcf4-wrk7s:8081/health</healthCheckUrl>\n" +
		"<vipAddress>xxl-job-executor-paas</vipAddress>\n" +
		"<secureVipAddress>xxl-job-executor-paas</secureVipAddress>\n" +
		"<isCoordinatingDiscoveryServer>false</isCoordinatingDiscoveryServer>\n" +
		"<lastUpdatedTimestamp>1631448499517</lastUpdatedTimestamp>\n" +
		"<lastDirtyTimestamp>1631443454988</lastDirtyTimestamp>\n" +
		"<actionType>ADDED</actionType>\n" +
		"</instance>\n" +
		"</application>"
	instanceNoMetaXMLStr = "<application>\n" +
		"<name>XXL-JOB-EXECUTOR-PAAS</name>\n" +
		"<instance>\n" +
		"<instanceId>xxl-job-executor-b7c89dcf4-wrk7s:xxl-job-executor-paas:8081</instanceId>\n" +
		"<hostName>xxl-job-executor-b7c89dcf4-wrk7s</hostName>\n" +
		"<app>XXL-JOB-EXECUTOR-PAAS</app>\n" +
		"<ipAddr>10.157.22.100</ipAddr>\n" +
		"<status>UP</status>\n" +
		"<overriddenstatus>UNKNOWN</overriddenstatus>\n" +
		"<port enabled=\"true\">8081</port>\n" +
		"<securePort enabled=\"false\">443</securePort>\n" +
		"<countryId>1</countryId>\n" +
		"<dataCenterInfo class=\"com.netflix.appinfo.InstanceInfo$DefaultDataCenterInfo\">\n" +
		"<name>MyOwn</name>\n" +
		"</dataCenterInfo>\n" +
		"<leaseInfo>\n" +
		"<renewalIntervalInSecs>30</renewalIntervalInSecs>\n" +
		"<durationInSecs>90</durationInSecs>\n" +
		"<registrationTimestamp>1631448499517</registrationTimestamp>\n" +
		"<lastRenewalTimestamp>1632368761523</lastRenewalTimestamp>\n" +
		"<evictionTimestamp>0</evictionTimestamp>\n" +
		"<serviceUpTimestamp>1631448499517</serviceUpTimestamp>\n" +
		"</leaseInfo>\n" +
		"<metadata class=\"java.util.Collections$EmptyMap\"/>\n" +
		"<homePageUrl>http://xxl-job-executor-b7c89dcf4-wrk7s:8081/</homePageUrl>\n" +
		"<statusPageUrl>http://xxl-job-executor-b7c89dcf4-wrk7s:8081/info</statusPageUrl>\n" +
		"<healthCheckUrl>http://xxl-job-executor-b7c89dcf4-wrk7s:8081/health</healthCheckUrl>\n" +
		"<vipAddress>xxl-job-executor-paas</vipAddress>\n" +
		"<secureVipAddress>xxl-job-executor-paas</secureVipAddress>\n" +
		"<isCoordinatingDiscoveryServer>false</isCoordinatingDiscoveryServer>\n" +
		"<lastUpdatedTimestamp>1631448499517</lastUpdatedTimestamp>\n" +
		"<lastDirtyTimestamp>1631443454988</lastDirtyTimestamp>\n" +
		"<actionType>ADDED</actionType>\n" +
		"</instance>\n" +
		"</application>"
)

// TestInstanceInfo_UnmarshalXML 测试反序列化XML
func TestInstanceInfo_UnmarshalXML(t *testing.T) {
	app := &Application{}
	err := xml.NewDecoder(strings.NewReader(instanceNoMetaXMLStr)).Decode(app)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("app is %+v\n", app.Instance[0].Metadata.Meta)

	builder := &strings.Builder{}
	err = xml.NewEncoder(builder).Encode(app)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("xml values is %s\n", builder.String())
}
