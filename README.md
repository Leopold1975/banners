[!Linters](https://github.com/Leopold1975/banners/actions/workflows/linters.yml/badge.svg)(https://github.com/Leopold1975/banners/actions/workflows/linters.yml)
# Banners Service

## ТЗ проекта:
https://github.com/avito-tech/backend-trainee-assignment-2024

## Описание проекта

Проект представляет собой сервис, предоставляющий возможности показа и управления баннерами через документированный API. Сервис использует RBAC модель доступа.
Для реализации сервиса был использован язык **GO**.
Технологический стэк сервиса также включает: 
- **PostgreSQL**,
- **Redis**,
- **Docker**.

Сопутствующие инструменты и практики разработки: 
- Генерация REST API - [oapi-codegen](https://github.com/deepmap/oapi-codegen), совместимый со стандартной бибилотекой `chi`;
- Сборка SQL-запросов - `squirrel`;
- Логирование - `zap`;
- Миграции - `goose`;
- Авторизация - `jwt`;
- Хранение и кэширование данных - `pgx`, `go-redis`.
- Было настроено логирование, добавлена поддержка возможности синхронизировать вывод stdout/stderr в файлы, заданные в кофигурации. Логи выдаются в JSON формате с тем, для дальшейнего подключения систему мониторинга. 
- С целью изолировать данные был применен скрипт для инициализации базы данных. 
- Для поддержки актуальности кэша используется стратегия `Background Refresh`.
По условию, количество тэгов и фичей <=1000, поэтому было принято решение дополнительно нагрузить redis cache композитным индексом для ускорения ответа. Для небольшого количества фичей и тэгов (до 1000) индекс не создаст серьезных дополнительных затрат по памяти.

## Запуск сервиса

Для запуска используются следующие утилиты:

```bash
docker compose version
Docker Compose version v2.25.0

go version
go version go1.22.1 linux/amd64
```

Конфигурация запуска по умолчанию находится в директории `./configs/config_dev.yaml`

Чтобы запустить сервис:
```bash
git clone https://github.com/Leopold1975/banners.git
cd banners
make run
```

Для остановки сервиса:
```bash
make down
```

## Принятые предположения, неоговоренные в ТЗ

- Был реализован простой вариант user сервиса на сервисном слое, который впоследствии может быть вынесен в отдельный сервис. Для удобства тестирования и использования в API были добавлены эндпойнты. Подробности изменения API [здесь](https://github.com/Leopold1975/banners/commit/3bc69d235d8819e03344eb7ff6dd6ad4536d11f2#:~:text=type%3A%20string-,/user%3A,-post%3A).

- Сервис баннеров является внутренним сервисом и не общается с пользователями напрямую, поэтому ошибки возвращаются, неся максимум информации.

- Удаление баннера должно происходить для всех пользователей, пользователь, получающий баннер без флага `use_last_revision` не должен получать удаленный баннер.

- Админ всегда получает актуальные баннеры.

- Пользователь может получить баннер с любым сочетанием feaute_id, tag_id (сценарий экспериментального показа новых баннеров пользователю, которые ещё не соотнесены с пользователем).

## Работа с API сервиса
Документация API доступна по адресу `/v1/docs`
### Аутентификация админа
```bash
curl --request POST --url http://127.0.0.1:5555/v1/auth --header 'Content-Type: application/json' --data '{
    "username": "Admin",
    "password": "1234"
}'
```
Ответ:
```json
{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjY2NTMsInJvbGUiOiJhZG1pbiJ9.pF7RY9D4pFoRWe0BpF_-AbLERRDGGZLrCMzY6Sy79c8"}
```
### Создание баннера:
```bash
curl --request POST --url http://127.0.0.1:5555/v1/banner -H 'Content-Type: application/json' -H 'Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjY2NTMsInJvbGUiOiJhZG1pbiJ9.pF7RY9D4pFoRWe0BpF_-AbLERRDGGZLrCMzY6Sy79c8' --data '{
    "feature_id" : 5,
    "tag_ids" : [ 2,6 ],
    "is_active" : true,
    "content" : {"title": "text", "text": "text", "url": "text"}
}' 

```
Ответ:
```json
{"banner_id":1}
```
### Создание пользователя:
```bash
curl --request POST --url http://127.0.0.1:5555/v1/user -H 'Content-Type: application/json' --data '{
    "username": "user",
    "password": "qwerty",
    "role": "user",
    "feature_id" : 5,
    "tag_ids" : [ 2, 3 ]
}'
```
Ответ:
```json
{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjc1MjEsInJvbGUiOiJ1c2VyIn0.LLRV31xu1rzbl5ccBztai3dffsnXV5BucH6r42rOAfI"}
```
### Получение баннера пользователя:
```bash
curl --request GET --url http://127.0.0.1:5555/v1/user_banner?feature_id=5\&tag_id=2 -H 'Content-Type: application/json' -H 'Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjc1MjEsInJvbGUiOiJ1c2VyIn0.LLRV31xu1rzbl5ccBztai3dffsnXV5BucH6r42rOAfI'
```
Ответ:
```json
{"text":"text","title":"text","url":"text"}
```
### Получение баннеров по параметрам с поддержкой пагинации и лимита:
```bash
curl --request GET --url http://127.0.0.1:5555/v1/banner?feature_id=5\&tag_id=2 -H 'Content-Type: application/json' -H 'Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjY2NTMsInJvbGUiOiJhZG1pbiJ9.pF7RY9D4pFoRWe0BpF_-AbLERRDGGZLrCMzY6Sy79c8'
```
Ответ:
```json
[
  {
    "feature_id": 5,
    "tag_ids": [
      2,
      6
    ],
    "is_active": true,
    "updated_at": "2024-04-13T20:37:52.597039Z",
    "banner_id": 1,
    "created_at": "2024-04-13T20:37:52.597039Z",
    "content": {
      "text": "text",
      "title": "text",
      "url": "text"
    }
  },
  {
    "feature_id": 5,
    "tag_ids": [
      1,
      2
    ],
    "is_active": true,
    "updated_at": "2024-04-13T21:08:48.848547Z",
    "banner_id": 3,
    "created_at": "2024-04-13T21:08:48.848547Z",
    "content": {
      "text": "desc",
      "title": "lyrics",
      "url": "url"
    }
  }
]
```
### Обновление баннера:
```bash
curl --request PATCH --url http://127.0.0.1:5555/v1/banner/1 -H 'Content-Type: application/json' -H 'Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjY2NTMsInJvbGUiOiJhZG1pbiJ9.pF7RY9D4pFoRWe0BpF_-AbLERRDGGZLrCMzY6Sy79c8' --data '{
    "feature_id" : 5,
    "tag_ids" : [ 2,6 ],
    "is_active" : true,
    "content" : {"title": "not text", "text": "not text", "url": "not text"}
}' 
```
### Удаление баннера:
```bash
curl --request DELETE --url http://127.0.0.1:5555/v1/banner/1 -H 'Content-Type: application/json' -H 'Token: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3MTMxMjY2NTMsInJvbGUiOiJhZG1pbiJ9.pF7RY9D4pFoRWe0BpF_-AbLERRDGGZLrCMzY6Sy79c8'
```

## Выполненые требования
- [x] Реализовано использование 2 видов токенов для атворизации: пользовательский и админский. Получение баннера может происходить с помощью пользовательского или админского токена, а все остальные действия могут выполняться только с помощью админского токена. 
- [x] Реализован интеграционный тест на сценарий получение баннера.
- [x] Реализован функционал флага use_last_revision.
- [x] Реализован функционал выключения баннеров для пользователей.
- [x] Был настроен линтер golangci-lint.
- [x] Реализован интеграционный тест на различные сценарии использования.
