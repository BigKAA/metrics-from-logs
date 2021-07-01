// cSpell:disable
package instance

import (
	"os"
	"strconv"
	"strings"
	"testing"

	"github.com/gomodule/redigo/redis"
	"github.com/sirupsen/logrus"
)

func getLoggerAndRedis() (*logrus.Entry, redis.Conn) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})
	level, _ := logrus.ParseLevel("debug")
	logger.SetLevel(level)
	logs := logger.WithFields(logrus.Fields{
		"pid": strconv.Itoa(os.Getpid()),
	})

	config := &Config{
		Confd:         "etc\\mfl\\conf.d\\",
		Loglevel:      "debug",
		Bindaddr:      "127.0.0.1:8080",
		Context:       "/",
		EsHost:        "127.0.0.1",
		EsPort:        "9200",
		EsUser:        "user",
		EsPassword:    "password",
		K8sPod:        "",
		K8sNamespace:  "",
		RedisServer:   "127.0.0.1",
		RedisPort:     "6379",
		RedisPassword: "",
	}

	pool := newRedisPool(config.RedisServer+":"+config.RedisPort, config.RedisPassword)
	conn := pool.Get()

	return logs, conn
}

func Test_createMetricsPage(t *testing.T) {
	logs, conn := getLoggerAndRedis()
	defer conn.Close()

	ret := createMetricsPage(logs, conn)
	if ret == "" {
		t.Fail()
		return
	}
	logs.Debug(ret)
	t.Log("Success")
}

func Test_getMetricsFromRedis(t *testing.T) {
	logs, conn := getLoggerAndRedis()
	defer conn.Close()

	ret, err := getMetricsFromRedis(logs, conn)
	if err != nil {
		t.Error("ERROR getMetricsFromRedis: ", err)
		t.Fail()
		return
	}
	logs.Debug("Keys ", strings.Join(ret, " -- "))
	t.Log("Success")
}
