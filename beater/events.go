package beater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/yangb8/ecsbeat/ecs"
)

// ErrInvalidResponseContent defines invalid response contenet error
var ErrInvalidResponseContent = fmt.Errorf("invalid response content")

// DecodeResponse ...
func DecodeResponse(resp *http.Response) ([]map[string]interface{}, error) {
	var m map[string]interface{}
	err := json.NewDecoder(resp.Body).Decode(&m)
	if err != nil {
		return nil, err
	}

	if embedded, ok := m["_embedded"]; ok {
		return DecodeResponseForEmbeddedInstances(embedded)
	} else if alert, ok := m["alert"]; ok {
		return DecodeResponseForAlertEvent(alert)
	} else if auditevent, ok := m["auditevent"]; ok {
		return DecodeResponseForAlertEvent(auditevent)
	} else if nsbilling, ok := m["namespace_billing_infos"]; ok {
		return DecodeResponseForNsBilling(nsbilling)
	} else if nsbillingsample, ok := m["namespace_billing_sample_infos"]; ok {
		return DecodeResponseForNsBilling(nsbillingsample)
	}
	return []map[string]interface{}{m}, nil
}

// DecodeResponseForEmbeddedInstances ...
func DecodeResponseForEmbeddedInstances(embedded interface{}) ([]map[string]interface{}, error) {
	embeddedMap, ok := embedded.(map[string]interface{})
	if !ok {
		return nil, ErrInvalidResponseContent
	}
	result := make([]map[string]interface{}, 0)
	if instances, ok := embeddedMap["_instances"]; ok {
		instancesSlice, ok := instances.([]interface{})
		if !ok {
			return nil, ErrInvalidResponseContent
		}
		for _, inst := range instancesSlice {
			instMap, ok := inst.(map[string]interface{})
			if !ok {
				return nil, ErrInvalidResponseContent
			}
			result = append(result, instMap)
		}
	}
	return result, nil
}

// DecodeResponseForAlertEvent ...
func DecodeResponseForAlertEvent(alerts interface{}) ([]map[string]interface{}, error) {
	alertSlice, ok := alerts.([]interface{})
	if !ok {
		return nil, ErrInvalidResponseContent
	}
	result := make([]map[string]interface{}, 0)
	for _, alt := range alertSlice {
		if altMap, ok := alt.(map[string]interface{}); ok {
			result = append(result, altMap)
		}
	}
	return result, nil
}

// DecodeResponseForNsBilling ...
func DecodeResponseForNsBilling(nsbillings interface{}) ([]map[string]interface{}, error) {
	nsbillingslice, ok := nsbillings.([]interface{})
	if !ok {
		return nil, ErrInvalidResponseContent
	}
	result := make([]map[string]interface{}, 0)
	for _, b := range nsbillingslice {
		bMap, ok := b.(map[string]interface{})
		if !ok {
			return nil, ErrInvalidResponseContent
		}
		result = append(result, bMap)
	}
	return result, nil
}

func addEntryToEvent(event map[string]interface{}, key string, value interface{}) {
	if entries, ok := value.([]interface{}); ok {
		if len(entries) > 0 {
			if mapEntry, ok := entries[0].(map[string]interface{}); ok {
				for k, v := range mapEntry {
					if k != "t" {
						event[key+"_"+k] = v
					}
				}
			}
		}
	} else {
		event[key] = value
	}
}

func transformEvent(event map[string]interface{}) {
	delete(event, "_links")
	for k, v := range event {
		if strings.HasSuffix(k, "Current") {
			if mapEntry, ok := v.(map[string]interface{}); ok {
				for k1, v2 := range mapEntry {
					addEntryToEvent(event, k+"_"+k1, v2)
				}
			} else {
				addEntryToEvent(event, k, v)
			}
			delete(event, k)
		}
	}
}

func addCommonFields(event map[string]interface{}, config *ClusterConfig, vdc, node, etype string) {
	event["@version"] = "1.0"
	event["@timestamp"] = common.Time(time.Now())
	event["type"] = "ecsbeat"
	event["ecs-customer"] = config.CustomerName
	event["ecs-event-type"] = etype
	if v, ok := config.Vdcs[vdc]; ok {
		event["ecs-vdc-cfgname"], event["ecs-vdc-id"], event["ecs-vdc-name"] = v.Get()
		if n, ok := v.NodeInfo[node]; ok {
			event["ecs-node-ip"], event["ecs-node-name"], event["ecs-version"] = n.Get()
		}
	}
}

func getFilledURI(cmd *Command, ip string) string {
	switch cmd.Type {
	case "nsbillingsample":
		// -5 mins to make sure the round time won't be in the future
		// cmd.Interval must be multiple of 5 mins
		t := time.Now().Add(-5 * time.Minute).Round(5 * time.Minute)
		return cmd.URI + fmt.Sprintf("&start_time=%s&end_time=%s",
			t.Add(-cmd.Interval).Format(time.RFC3339)[:16],
			t.Format(time.RFC3339)[:16])
	case "auditevent":
		t := time.Now().Add(-1 * time.Minute).Round(1 * time.Minute)
		return cmd.URI + fmt.Sprintf("?start_time=%s&end_time=%s",
			t.Add(-cmd.Interval).Format(time.RFC3339)[:16],
			t.Format(time.RFC3339)[:16])
	case "disks":
		fallthrough
	case "processes":
		return fmt.Sprintf(cmd.URI, ip)
	default:
		return cmd.URI
	}
}

// GenerateEvents ...
func GenerateEvents(cmd *Command, config *ClusterConfig, client *ecs.MgmtClient) ([]common.MapStr, error) {
	events := make([]common.MapStr, 0)
	switch cmd.Level {
	case "system":
		for vname := range config.Vdcs {
			var resp *http.Response
			var err error
			if cmd.Type == "nsbilling" || cmd.Type == "nsbillingsample" {
				ids, err := ecs.GetNamespaceIDs(client, vname)
				if err != nil {
					return nil, err
				}
				nsList := struct {
					ID []string `json:"id"`
				}{}
				for _, v := range ids {
					nsList.ID = append(nsList.ID, v)
				}

				body := new(bytes.Buffer)
				json.NewEncoder(body).Encode(nsList)
				headers := http.Header{}
				headers.Set("Content-Type", "application/json")
				resp, err = client.PostQuery(getFilledURI(cmd, ""), body, 0, headers, vname)
			} else {
				resp, err = client.GetQuery(getFilledURI(cmd, ""), vname)
			}
			if err != nil {
				return nil, err
			}

			decoded, err := DecodeResponse(resp)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			for _, d := range decoded {
				transformEvent(d)
				addCommonFields(d, config, "", "", cmd.Type)
				events = append(events, common.MapStr(d))
			}
			break
		}
	case "vdc":
		for vname, vdc := range config.Vdcs {
			resp, err := client.GetQuery(getFilledURI(cmd, ""), vname)
			if err != nil {
				return nil, err
			}

			decoded, err := DecodeResponse(resp)
			resp.Body.Close()
			if err != nil {
				return nil, err
			}
			for _, d := range decoded {
				transformEvent(d)
				addCommonFields(d, config, vdc.ConfigName, "", cmd.Type)
				events = append(events, common.MapStr(d))
			}
		}
	case "node":
		for vname, vdc := range config.Vdcs {
			for _, node := range vdc.NodeInfo {
				var resp *http.Response
				var err error
				if cmd.Type == "dtinfo" {
					resp, err = client.GetQueryBase("http", "9101", getFilledURI(cmd, ""), vname)
				} else {
					resp, err = client.GetQuery(getFilledURI(cmd, node.IP), vname)
				}
				if err != nil {
					return nil, err
				}

				decoded, err := DecodeResponse(resp)
				resp.Body.Close()
				if err != nil {
					return nil, err
				}
				for _, d := range decoded {
					transformEvent(d)
					addCommonFields(d, config, vdc.ConfigName, node.IP, cmd.Type)
					events = append(events, common.MapStr(d))
				}
			}
		}
	}

	return events, nil
}
