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
MFL_CONTEXT=/
MFL_ES_HOST=127.0.0.1:9500
MFL_ES_USER=user
MFL_ES_PASSWORD=password
```

## Конфигурационные файлы сбора метрик

Файлы должны находиться в директории confd (см. основной конфигурационный файл) и иметь расширение
*.yaml

```yaml
metric: MetricName
metrictype: counter # gauge, histogram, summary
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

# Запуск приложения

## Docker

В Windows:

    docker run -it --rm --name mfl --env-file .env -p 8080:8080 -v C:\Users\artur\metrics-from-logs\etc\mfl\conf.d:/etc/mfl/conf.d metricsfromlogs:0.01
    curl http://host.docker.internal:8080
