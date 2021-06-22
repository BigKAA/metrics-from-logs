// cSpell:disable
package instance

import (
	"strconv"
	"time"
)

// beMaster Запускаем процедуры мастера.
func (i *Instance) beMaster() error {
	if i.role == MASTER {
		return nil
	}

	i.logs.Info("start master")

	i.role = MASTER

	// Читаем метрики из конфигурационных файлов
	if i.metrics != nil {
		err := i.FillMetrics()
		if err != nil || i.metrics == nil {
			i.logs.Panic("Неудалось сформировать массив метрик. ", err)
			// если не удалось прочитать конфигурационный файл - это критическая ошибка.
			// можно выклюать программу.
			return err
		}
	}

	if i.config.Loglevel == "debug" {
		for _, m := range i.metrics {
			i.logs.Debug("{ Метрика: " + m.Mertic + ", запрос: " + m.Query + ", периодичность: " + strconv.Itoa(m.Repeat) + "}")
		}
	}

	// Создаём канал с тикером.
	expireChan := time.NewTicker(time.Millisecond * masterExpire / 2).C
	abort := make(chan bool)

	go i.doMaster(abort)

	for {
		<-expireChan
		if i.expireMaster() {
			abort <- true // завершить мастер процессы
			break
		}
	}

	return nil
}

// expireMaster Пытаемся остаться мастером :) Обновляем, если это возможно, мастер ключ.
func (i *Instance) expireMaster() bool {
	conn := i.pool.Get()
	defer conn.Close()

	if _, err := conn.Do("PEXPIRE", masterKey, masterExpire); err != nil {
		i.logs.Info("master expare")
		return true
	}
	return false
}

func (i *Instance) doMaster(abort chan bool) {
	time.Sleep(4000000000)
	i.logs.Debug("doMaster - TIK")
}
