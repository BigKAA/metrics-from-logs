// cSpell:disable
package instance

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"
)

// es Запрос в es
func (i *Instance) es(rMetric *RedisMetric) (int64, error) {
	// получаем количество записей
	count, err := executeEsCount(rMetric, i)
	if err != nil {
		i.Logs.Error("f: es - error executeEsCount: ", err)
		return 0, err
	}
	i.Logs.Debug("f: es - count: ", count)

	return count, nil
}

// executeEsCount Выполнение запроса типа _count
// Возвращает число, количество записей, удовлетворяющих запросы.
func executeEsCount(rMetric *RedisMetric, i *Instance) (int64, error) {
	bQuery := []byte(rMetric.Query)
	isValid := json.Valid(bQuery)
	if !isValid {
		i.Logs.Debug("f: executeEsCount - ERROR: query string not valid: " + rMetric.Query)
		return 0, errors.New("ERROR: query string not valid: " + rMetric.Query)
	}

	uri := i.Config.EsHost + ":" + i.Config.EsPort + "/" + rMetric.Index + "/_count"

	i.Logs.Debug("f: executeEsCount - URI: ", uri)

	req, _ := http.NewRequest("POST", uri, bytes.NewBuffer(bQuery))
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(i.Config.EsUser, i.Config.EsPassword)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		i.Logs.Error("f: executeEsCount - json.NewEncoder() ERROR:", err)
		return 0, err
	}

	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)

	if !strings.HasPrefix(resp.Status, "2") {
		if !strings.HasPrefix(resp.Status, "3") {
			i.Logs.Error("f: executeEsCount - status: ", resp.Status)
			return 0, errors.New("FUNCTION RETURN NOT 2xx OR 3xx STATUS")
		}
	}

	i.Logs.Debug("f: executeEsCount - body: ", string(body))

	type RespCountShards struct {
		Total      int64 `json:"total"`
		Successful int64 `json:"successful"`
		Skipped    int64 `json:"skipped:"`
		Failed     int64 `json:"failed:"`
	}

	type RespCount struct {
		Count  int64           `json:"count"`
		Shards RespCountShards `json:"_shards"`
	}

	rs := &RespCount{}
	err = json.Unmarshal(body, &rs)
	if err != nil {
		i.Logs.Error("f: executeEsCount - json.Unmarshal error: ", err)
		return 0, err
	}
	i.Logs.Debug("f: executeEsCount - count: ", rs.Count)

	return rs.Count, nil
}
