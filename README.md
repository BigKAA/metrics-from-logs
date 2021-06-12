# metrics-from-logs
Получении метрик в формате Prometheus на основании информации из elasticseach.

В основном предполагается, что в elasticsearch хранятся логи. 

## Конфигурационный файл

Главный конфигурационный файл определяется параметром -с. Значение по умолчанию:

    -c config.yaml

### Формат файла

```yaml
main:
  confd: "etc\\mfl\\conf.d"
  loglevel: "debug"
```

* confd - директория, в которой программа будет искать файлы с расширением *.conf
* loglevel - уровень важности сообщений.

## Конфигурационные файлы сбора метрик

Файлы должны находиться в директории confd (см. основной конфигурационный файл) и иметь расширение
*.yaml

```yaml
metric: MetricName
metrictype: counter # gauge, histogram, summary
esserver: es.any.com
esserverport: 12345
esuserfrom: conf # file, env
esuser: user
espasswordfrom: conf # file, env
espassword: password
query: |
    { 
        the,
        query
    }
```
