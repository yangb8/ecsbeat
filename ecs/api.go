package ecs

import (
	"bufio"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// LocalVDC ...
type LocalVDC struct {
	Global            bool   `json:"global"`
	ID                string `json:"id"`
	Inactive          bool   `json:"inactive"`
	InterVdcEndPoints string `json:"interVdcEndPoints"`
	Link              struct {
		Href string `json:"href"`
		Rel  string `json:"rel"`
	} `json:"link"`
	Name              string `json:"name"`
	PermanentlyFailed bool   `json:"permanentlyFailed"`
	Remote            bool   `json:"remote"`
	SecretKeys        string `json:"secretKeys"`
	Vdc               struct {
		ID   string `json:"id"`
		Link string `json:"link"`
	} `json:"vdc"`
	VdcID   string `json:"vdcId"`
	VdcName string `json:"vdcName"`
}

// GetLocalVDC ...
func GetLocalVDC(client *MgmtClient, vdc string) (*LocalVDC, error) {
	resp, err := client.GetQuery("/object/vdcs/vdc/local.json", vdc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := LocalVDC{}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// Nodes ...
type Nodes struct {
	Node []struct {
		IP       string `json:"ip"`
		IsLocal  bool   `json:"isLocal"`
		Nodeid   string `json:"nodeid"`
		Nodename string `json:"nodename"`
		RackID   string `json:"rackId"`
		Version  string `json:"version"`
		Status   string `json:"status"`
	} `json:"node"`
}

// GetNodes ...
func GetNodes(client *MgmtClient, vdc string) (*Nodes, error) {
	resp, err := client.GetQuery("/vdc/nodes.json", vdc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := Nodes{}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// StoragePool ...
type StoragePool struct {
	Varray []struct {
		ID                   string `json:"id"`
		IsColdStorageEnabled bool   `json:"isColdStorageEnabled"`
		IsProtected          bool   `json:"isProtected"`
		Name                 string `json:"name"`
	} `json:"varray"`
}

// GetStoragePool ...
func GetStoragePool(client *MgmtClient, vdc string) (*StoragePool, error) {
	resp, err := client.GetQuery("/vdc/data-services/varrays.json", vdc)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	result := StoragePool{}
	if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	return &result, nil
}

// DtEntry ...
type DtEntry struct {
	DtID        string `json:"dt-id"`
	DtCreated   string `json:"dt-created"`
	DtError     string `json:"dt-error"`
	DtDown      int    `json:"dt-down"`
	DtLevel     string `json:"dt-level"`
	DtOwnerIP   string `json:"dt-owner-ip"`
	DtReady     int    `json:"dt-ready"`
	DtPartition string `json:"dt-partition"`
	DtStatus    string `json:"dt-status"`
	DtType      string `json:"dt-type"`
	DtTypeLvl   string `json:"dt-type-level"`
}

// DtInfos ...
type DtInfos struct {
	DtEntries []DtEntry
}

// GetDtInfos ...
func GetDtInfos(mc *MgmtClient, host string) (*DtInfos, error) {
	req, err := http.NewRequest("GET", "http://"+host+":9101/diagnostic/DumpOwnershipInfo/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Status Code %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	result := DtInfos{}
	reader := bufio.NewReader(resp.Body)
	var line string
	for {
		if line, err = reader.ReadString('\n'); err != nil {
			if err != io.EOF {
				return nil, err
			}
			break
		}
		if dt := parseDtInfoStr(line); dt != nil {
			result.DtEntries = append(result.DtEntries, *dt)
		}
	}
	return &result, nil
}

func parseDtInfoStr(s string) *DtEntry {
	var idx int

	if idx = strings.Index(s, "[id:"); idx < 0 {
		return nil
	}

	result := DtEntry{DtCreated: "false"}
	for _, part := range strings.Split(s[idx:], ",") {
		part = strings.TrimSpace(part)
		part = strings.TrimLeft(part, "[")
		part = strings.TrimRight(part, "]")
		part = strings.TrimSpace(part)
		kvPair := strings.SplitN(part, ":", 2)
		if len(kvPair) == 2 {
			key, value := strings.TrimSpace(kvPair[0]), strings.TrimSpace(kvPair[1])
			switch key {
			case "id":
				value = strings.TrimRight(value, ":")
				tmp := strings.Split(value, ":")
				value = tmp[len(tmp)-1]
				if len(value) == 0 {
					return nil
				}
				result.DtID = value
			case "owner":
				if len(value) == 0 {
					return nil
				}
				if idx = strings.LastIndex(value, ":"); idx < 0 {
					result.DtOwnerIP = value
				} else {
					result.DtOwnerIP = value[:idx]
				}
			case "creationCompleted":
				result.DtCreated = value
			}
		}
	}
	result.DtDown = 0
	result.DtReady = 1
	result.DtStatus = "ready"
	parseDtID(&result, result.DtID)

	return &result
}

// DtInitsXML ...
type DtInitsXML struct {
	Entries []struct {
		Items []struct {
			XMLName xml.Name
			Content string `xml:",innerxml"`
		} `xml:",any"`
	} `xml:"entry"`
}

// GetDtInits ...
func GetDtInits(mc *MgmtClient, host string) (*DtInfos, error) {
	req, err := http.NewRequest("GET", "http://"+host+":9101/stats/dt/DTInitStat/", nil)
	if err != nil {
		return nil, err
	}
	resp, err := mc.client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Status Code %d", resp.StatusCode)
	}
	dtXML := DtInitsXML{}
	err = xml.NewDecoder(resp.Body).Decode(&dtXML)
	resp.Body.Close()
	if err != nil {
		return nil, err
	}

	result := DtInfos{}
	for _, entry := range dtXML.Entries {
		for _, item := range entry.Items {
			if item.XMLName.Local == "type" || item.XMLName.Local == "level" || item.XMLName.Local == "total_dt_num" ||
				item.XMLName.Local == "unready_dt_num" || item.XMLName.Local == "unknown_dt_num" {
				continue
			}
			if strings.Contains(item.Content, "urn:") {
				IdsStr := strings.TrimSpace(item.Content)
				IdsStr = strings.TrimLeft(IdsStr, "[")
				IdsStr = strings.TrimRight(IdsStr, "]")
				IdsStr = strings.TrimSpace(IdsStr)
				for _, id := range strings.Split(IdsStr, ",") {
					id = strings.TrimRight(id, ":")
					tmp := strings.Split(id, ":")
					id = tmp[len(tmp)-1]
					dt := DtEntry{
						DtID:    id,
						DtError: item.XMLName.Local,
					}
					if strings.Contains(item.XMLName.Local, "ERROR_RPC_CLIENT_NO_RESPONSE") {
						dt.DtStatus, dt.DtReady, dt.DtDown = "unknown", 0, 0
					} else {
						dt.DtStatus, dt.DtReady, dt.DtDown = "unready", 0, 1
					}
					result.DtEntries = append(result.DtEntries, dt)
				}
			}
		}
	}
	return &result, nil
}

func parseDtID(d *DtEntry, dtid string) {
	parts := strings.Split(dtid, "_")
	if len(parts) >= 3 {
		d.DtType = parts[2]
	}
	if len(parts) >= 4 {
		d.DtPartition = parts[3]
	}
	if len(parts) >= 6 {
		d.DtLevel = parts[5]
		d.DtTypeLvl = d.DtType + "_" + d.DtLevel
	}
}

// NamespaceList ...
type NamespaceList struct {
	Namespace []struct {
		Name string `json:"name"`
		ID   string `json:"id"`
		Link struct {
			Rel  string `json:"rel"`
			Href string `json:"href"`
		} `json:"link"`
	} `json:"namespace"`
	Filter       string `json:"Filter"`
	NextMarker   string `json:"NextMarker"`
	NextPageLink string `json:"NextPageLink"`
}

// GetNamespaceIDs ...
func GetNamespaceIDs(client *MgmtClient, vdc string) ([]string, error) {
	var (
		idslice []string
		resp    *http.Response
		err     error
		marker  string
		result  NamespaceList
	)

	for {
		if len(marker) == 0 {
			resp, err = client.GetQuery("/object/namespaces.json", vdc)
		} else {
			resp, err = client.GetQuery(fmt.Sprintf("/object/namespaces.json?marker=%s", url.QueryEscape(marker)), vdc)
		}
		if err != nil {
			return nil, err
		}

		err = json.NewDecoder(resp.Body).Decode(&result)
		resp.Body.Close()
		if err != nil {
			return nil, err
		}
		for _, ns := range result.Namespace {
			idslice = append(idslice, ns.ID)
		}
		marker = result.NextMarker
		if len(marker) == 0 {
			break
		}
	}
	return idslice, nil
}
