# metrics-from-logs
Получении метрик в формате Prometheus на основании информации из elasticseach.

В основном предполагается, что в elasticsearch хранятся логи. 

## Конфигурация программы

Основная конфигурация программы производится при помощи переменных среды окружения:

* MFL_CONF_DIR - директория, в которой программа будет искать файлы с расширением *.conf
* MFL_LOG_LEVEL  - уровень важности сообщений системы логирования.
* MFL_BIND_ADDR - адрес и порт, на которых слушает запросы программа.

Во время отладки приложения, вместо переменных среды окружения можно использовать файл .env с
помещенными в него переменными и их значениями.

Пример файла .env

```
MFL_CONF_DIR=etc\\mfl\\conf.d\\
MFL_LOG_LEVEL=debug
MFL_BIND_ADDR=127.0.0.1:8080
```

## Конфигурационные файлы сбора метрик

Файлы должны находиться в директории confd (см. основной конфигурационный файл) и иметь расширение
*.yaml

```yaml
metric: MetricName
metrictype: counter # gauge, histogram, summary
esserver: es.any.com
esserverport: 12345
esuser: user
espassword: password
query: |
    { 
        the,
        query
    }
repeat: 60 # количество секунд
```

# Сборка

Все, что нужно для сборки уже есть в Dockerfile

    docker build -t metricsfromlogs:0.01 "."
