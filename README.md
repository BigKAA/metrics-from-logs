# metrics-from-logs
Получении метрик в формате Prometheus на основании информации из elasticsearch.

Вторая версия программы.

Фишки:
* Позволяет горизонтальное масштабирование.
* Требуется наличие Redis

![schema](images/scheme.jpg)

## Конфигурация программы

Основная конфигурация программы производится при помощи переменных среды окружения:

* MFL_CONF_DIR - директория, в которой программа будет искать файлы с расширением *.conf
* MFL_LOG_LEVEL  - уровень важности сообщений системы логирования.
* MFL_BIND_ADDR - адрес и порт, на которых слушает запросы программа.
* MFL_CONTEXT - контекст в URL.
* MFL_ES_HOST - машина, где запущен elasticsearch. Подставляется как label метрики.
* MFL_ES_PORT - es port.
* MFL_ES_USER - пользователь с правами которого обращаются к elasticsearch.
* MFL_ES_PASSWORD - пароль пользователя.
* MFL_K8S_POD - под k8s. Подставляется как label метрики.
* MFL_K8S_NAMESPACE - namespace k8s. Подставляется как label метрики.
* MFL_REDIS_SERVER - сервер Redis. Значение по умолчанию - 127.0.0.1.
* MFL_REDIS_PORT - порт сервера Redis. Значение по умолчанию - 6379.
* MFL_REDIS_PASSWORD - пароль доступа к Redis. Если не определён - доступ без пароля.

Если приложение запускается не в k8s, переменные MFL_K8S_POD и MFL_K8S_NAMESPACE определять не надо.

Во время отладки приложения, вместо переменных среды окружения можно использовать файл .env с
помещенными в него переменными и их значениями.

Пример файла .env

```
MFL_CONF_DIR=etc\\mfl\\conf.d\\
MFL_LOG_LEVEL=debug
MFL_BIND_ADDR=127.0.0.1:8080
MFL_CONTEXT=/
MFL_ES_HOST=127.0.0.1
MFL_ES_PORT=9200
MFL_ES_USER=user
MFL_ES_PASSWORD=password
MFL_REDIS_SERVER=redis_svc
MFL_REDIS_PORT=1234
```

## Конфигурационные файлы сбора метрик

Файлы должны находиться в директории confd (см. основной конфигурационный файл) и иметь расширение
*.yaml

```yaml
metric: MetricName
metrichelp: "Небольшое описание метрики. В одну строку."
metrictype: counter # gauge, histogram, summary
query: |
    { 
        the,
        query
    }
index: "index-name-*" # Имя индекса (шаблона) в котором происходит поиск.
repeat: 60 # количество секунд. Не должно быть меньше, чем elasticsearch затратит на обработку запроса.
delay: 10 # количество секунд, задержка перед запуском цикла запросов посте старта программы.
```

# Сборка

Все, что нужно для сборки уже есть в Dockerfile

    docker build -t metricsfromlogs:0.01 "."

# Запуск приложения

## Docker

В Windows:

    docker run -it --rm --name mfl --env-file .env -p 8080:8080 -v C:\Users\artur\metrics-from-logs\etc\mfl\conf.d:/etc/mfl/conf.d metricsfromlogs:0.01
    curl http://host.docker.internal:8080
