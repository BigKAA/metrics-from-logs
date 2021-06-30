// cSpell:disable
package instance

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"strings"

	"github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/estransport"
	"github.com/sirupsen/logrus"
)

// es Запрос в es
func (i *Instance) es(rMetric *RedisMetric) (int64, error) {
	es, err := getEsClient(i)
	if err != nil {
		i.logs.Error("f: es - error getEsClient: ", err)
		return 0, err
	}

	// получаем количество записей
	count, err := executeEsCount(es, rMetric, i.logs)
	if err != nil {
		i.logs.Error("f: es - error executeEsCount: ", err)
		return 0, err
	}
	i.logs.Debug("f: es - count: ", count)

	return count, nil
}

// executeEsCount Выполнение запроса типа _count
// Возвращает число, количество записей, удовлетворяющих запросы.
func executeEsCount(es *elasticsearch.Client, rMetric *RedisMetric, logs *logrus.Entry) (int64, error) {

	bQuery := []byte(rMetric.Query)
	isValid := json.Valid(bQuery)
	if !isValid {
		logs.Debug("constructQuery() ERROR: query string not valid:", rMetric.Query)
	} else {
		logs.Debug("constructQuery() valid JSON:", isValid)
	}

	var b strings.Builder
	b.WriteString(rMetric.Query)
	read := strings.NewReader(b.String())

	var buf bytes.Buffer

	if err := json.NewEncoder(&buf).Encode(read); err != nil {
		logs.Error("json.NewEncoder() ERROR:", err)
	}

	res, err := es.Count(
		es.Count.WithContext(context.Background()),
		es.Count.WithIndex(rMetric.Index),
		es.Count.WithBody(&buf),
	)
	if err != nil {
		logs.Error("f: es - error search: ", err)
		return 0, err
	}
	defer res.Body.Close()

	if res.IsError() {
		var e map[string]interface{}
		if err := json.NewDecoder(res.Body).Decode(&e); err != nil {
			logs.Error("f: es - error parsing respons body: ", err)
			return 0, err
		} else {
			// Print the response status and error information.
			logs.Debugf("[%s] %s: %s",
				res.Status(),
				e["error"].(map[string]interface{})["type"],
				e["error"].(map[string]interface{})["reason"],
			)
		}
	}

	var r map[string]interface{}
	if err := json.NewDecoder(res.Body).Decode(&r); err != nil {
		logs.Errorf("f: es - error parsing the response body: %s", err)
	}
	// Print the response status, number of results, and request duration.
	logs.Debugf(
		"f: es - [%s] count %d ",
		res.Status(),
		int(r["count"].(float64)),
	)
	return int64(r["count"].(float64)), nil
}

// getEsClient Получаем клиент elasticsearch с указанными конфигурационными
// параметрами.
func getEsClient(i *Instance) (*elasticsearch.Client, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{
			i.config.EsHost + ":" + i.config.EsPort,
		},
		Username: i.config.EsUser,
		Password: i.config.EsPassword,
		// Transport: &http.Transport{
		// 	MaxIdleConnsPerHost:   10,
		// 	ResponseHeaderTimeout: time.Millisecond,
		// 	DialContext:           (&net.Dialer{Timeout: time.Nanosecond}).DialContext,
		// },
		Logger: &estransport.JSONLogger{Output: os.Stdout},
	}

	es, err := elasticsearch.NewClient(cfg)
	if err != nil {
		i.logs.Error("f: getEsClient - error NewClient: ", err)
		return nil, err
	}

	return es, nil
}
