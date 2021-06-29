// cSpell:disable
package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"net"
	"net/http"
	"time"

	"github.com/elastic/go-elasticsearch/v8"
)

func (i *Instance) es(rMetric RedisMetric) error {
	es, err := getEsClient(i)
	if err != nil {
		i.logs.Error("f: es - error getEsClient: ", err)
		return err
	}

	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(rMetric.Query); err != nil {
		i.logs.Error("f: es - error encoding query: ", err)
		return err
	}
	res, err := es.Count(
		es.Count.WithContext(context.Background()),
		es.Count.WithIndex(rMetric.Index),
		es.Count.WithBody(&buf),
	)
	if err != nil {
		i.logs.Error("f: es - error search: ", err)
		return err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			i.logs.Error("f: es - error parsing respons body: ", err)
			return err
		} else {
			// Print the response status and error information.
			i.logs.Debugf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		i.logs.Errorf("f: es - error parsing the response body: %s", err)
	}
	// Print the response status, number of results, and request duration.
	i.logs.Debugf(
		"f: es - [%s] count %d ",
		res.Status(),
		int(r["count"].(float64)),
	)
	return nil
}

func getEsClient(i *Instance) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			i.config.EsHost + ":" + i.config.EsPort,
		},
		Username: i.config.EsUser,
		Password: i.config.EsPassword,
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: time.Millisecond,
			DialContext:           (&net.Dialer{Timeout: time.Nanosecond}).DialContext,
		},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		i.logs.Error("f: getEsClient - error NewClient: ", err)
		return nil, err
	}

	return es, nil
}
