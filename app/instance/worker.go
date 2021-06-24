// cSpell:disable
package instance

import (
	"github.com/gomodule/redigo/redis"
)

// worker Основной процесс обработки метрик.
// Подписывается на канал в Redis.
// При получении собщения пытается прочитать из общей очереди id задания, запускает
// задание на обработку.
func (i *Instance) worker() {
	c := i.pool.Get()
	defer c.Close()
	psc := redis.PubSubConn{c}
	psc.Subscribe(mfl_query)
	for c.Err() == nil {
		switch v := psc.Receive().(type) {
		case redis.Message:
			i.logs.Debug("Message: " + v.Channel + " " + string(v.Data))
			go i.processRecievedMetric(string(v.Data))
		case error:
			i.logs.Error("Error channel " + mfl_query)
		}
	}
}

func (i *Instance) processRecievedMetric(key string) {
	conn := i.pool.Get()
	defer conn.Close()

	// Читаем из очереди.
	metric_key, err := conn.Do("LPOP", mfl_list)
	if err != nil {
		i.logs.Error("Redis LPOP error: ", err)
		return
	}

	values, err := redis.Values(conn.Do("HGETALL", metric_key))
	if err != nil {
		i.logs.Error("Redis HGETALL error: ", err)
		return
	}

	p := RedisMetric{}
	redis.ScanStruct(values, &p)

	i.logs.Debug("---- STRUCT: ", p)

	// Тут добавлять вызов запроса в эластик.
}
