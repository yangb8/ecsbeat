package beater

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
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
		if strings.HasSuffix(k, "Current") || strings.HasSuffix(k, "CurrentL1") || strings.HasSuffix(k, "CurrentL2") {
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
			_, event["ecs-node-ip"], event["ecs-node-name"], event["ecs-version"] = n.Get()
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
	case "alert":
		fallthrough
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
func GenerateEvents(cmd *Command, config *ClusterConfig, client *ecs.MgmtClient,
	done <-chan struct{}, out chan<- common.MapStr) (bool, error) {

	switch cmd.Level {
	case "system":
		for vname := range config.Vdcs {
			var resp *http.Response
			var err error
			if cmd.Type == "nsbilling" || cmd.Type == "nsbillingsample" {
				ids, err := ecs.GetNamespaceIDs(client, vname)
				if err != nil {
					logp.Err("%s: %v", cmd.Type, err)
					return true, err
				}
				nsList := struct {
					ID []string `json:"id"`
				}{}

				for i, v := range ids {
					nsList.ID = append(nsList.ID, v)
					if len(nsList.ID) == 100 || i == len(ids)-1 {
						body := new(bytes.Buffer)
						json.NewEncoder(body).Encode(nsList)
						headers := http.Header{}
						headers.Set("Content-Type", "application/json")
						resp, err = client.PostQuery(getFilledURI(cmd, ""), body, 0, headers, vname)
						if err != nil {
							logp.Err("%s: %v", cmd.Type, err)
							return true, err
						}
						// sometimes, ECS returns nil response for nsbillingsample
						if resp == nil {
							logp.Err("%s: %v", cmd.Type, err)
							return true, ErrInvalidResponseContent
						}
						decoded, err := DecodeResponse(resp)
						resp.Body.Close()
						if err != nil {
							logp.Err("%s: %v", cmd.Type, err)
							return true, err
						}
						for _, d := range decoded {
							transformEvent(d)
							addCommonFields(d, config, "", "", cmd.Type)
							if !writeEvent(done, out, common.MapStr(d)) {
								return false, nil
							}
						}
						nsList.ID = []string{}
					}
				}
			} else {
				resp, err = client.GetQuery(getFilledURI(cmd, ""), vname)
				if err != nil {
					logp.Err("%s: %v", cmd.Type, err)
					return true, err
				}
				decoded, err := DecodeResponse(resp)
				resp.Body.Close()
				if err != nil {
					logp.Err("%s: %v", cmd.Type, err)
					return true, err
				}
				for _, d := range decoded {
					transformEvent(d)
					addCommonFields(d, config, "", "", cmd.Type)
					if !writeEvent(done, out, common.MapStr(d)) {
						return false, nil
					}
				}
			}
			break
		}
	case "vdc":
		for vname, vdc := range config.Vdcs {
			resp, err := client.GetQuery(getFilledURI(cmd, ""), vname)
			if err != nil {
				logp.Err("%s: %v", cmd.Type, err)
				return true, err
			}

			decoded, err := DecodeResponse(resp)
			resp.Body.Close()
			if err != nil {
				logp.Err("%s: %v", cmd.Type, err)
				return true, err
			}
			for _, d := range decoded {
				transformEvent(d)
				if cmd.Type == "nodes" {
					if id, ok := d["id"]; ok {
						if idstr, ok := id.(string); ok {
							addCommonFields(d, config, vdc.ConfigName, vdc.GetIpById(idstr), cmd.Type)
						}
					}
				} else {
					addCommonFields(d, config, vdc.ConfigName, "", cmd.Type)
				}
				if !writeEvent(done, out, common.MapStr(d)) {
					return false, nil
				}
			}
		}
	case "node":
		for vname, vdc := range config.Vdcs {
			for _, node := range vdc.NodeInfo {
				resp, err := client.GetQuery(getFilledURI(cmd, node.ID), vname)
				if err != nil {
					logp.Err("%s: %v", cmd.Type, err)
					return true, err
				}
				decoded, err := DecodeResponse(resp)
				resp.Body.Close()
				if err != nil {
					logp.Err("%s: %v", cmd.Type, err)
					return true, err
				}
				for _, d := range decoded {
					transformEvent(d)
					addCommonFields(d, config, vdc.ConfigName, node.IP, cmd.Type)
					if !writeEvent(done, out, common.MapStr(d)) {
						return false, nil
					}
				}
			}
		}
	case "dtinfo":
		if cmd.Type == "dtinfo" {
			// try each ip until we got 2 responses w/o errors
			var fetched bool
			for _, vdc := range config.Vdcs {
				if fetched {
					break
				}
				for _, node := range vdc.NodeInfo {
					if fetched {
						break
					}
					dtInfos, err := ecs.GetDtInfos(client, node.IP)
					if err != nil {
						// try next node
						continue
					}
					dtInits, err := ecs.GetDtInits(client, node.IP)
					if err != nil {
						// try next node
						continue
					}
					fetched = true
					for _, entry := range dtInfos.DtEntries {
						for _, bad := range dtInits.DtEntries {
							if bad.DtID == entry.DtID {
								entry.DtError, entry.DtStatus, entry.DtReady, entry.DtDown = bad.DtError, bad.DtStatus, bad.DtReady, bad.DtDown
								break
							}
						}
						d := struct2Map(entry)
						transformEvent(d)
						addCommonFields(d, config, vdc.ConfigName, node.IP, cmd.Type)
						if !writeEvent(done, out, common.MapStr(d)) {
							return false, nil
						}
					}
				}
			}
		}
	}

	return true, nil
}

func struct2Map(obj interface{}) map[string]interface{} {
	typ := reflect.TypeOf(obj)
	val := reflect.ValueOf(obj)

	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
		val = val.Elem()
	}

	result := make(map[string]interface{})
	if typ.Kind() != reflect.Struct {
		return result
	}

	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		if field.Anonymous {
			continue
		}
		if name := field.Tag.Get("json"); name != "" {
			result[name] = val.Field(i).Interface()
		}
	}
	return result
}
