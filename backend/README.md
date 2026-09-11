# Veil API

Базовый Go-бэкенд для временных сессий браузера. Сервер использует только стандартную библиотеку Go и не сохраняет историю на диск.

## Структура

```text
backend/
├── cmd/server/          запуск HTTP-сервера
├── internal/config/     переменные окружения
├── internal/httpapi/    маршруты, cookie, CORS и HTTP-ответы
└── internal/session/    модель, проверка и хранилище сессий
```

## Запуск

```bash
cd backend
go run ./cmd/server
```

Сервер будет доступен на `http://localhost:8080`.

## API

| Метод | Путь | Назначение |
|---|---|---|
| `GET` | `/health` | проверка работы сервера |
| `POST` | `/api/v1/sessions` | создать временную сессию |
| `GET` | `/api/v1/sessions/current` | получить текущую сессию |
| `PUT` | `/api/v1/sessions/current` | сохранить вкладки и настройки |
| `DELETE` | `/api/v1/sessions/current` | удалить сессию |

Идентификатор хранится в `HttpOnly` cookie. Запросы состояния должны отправлять cookie вместе с запросом.

## Настройки

| Переменная | Значение по умолчанию |
|---|---|
| `HTTP_ADDR` | `:8080` |
| `APP_ORIGIN` | `http://localhost:3000` |
| `SESSION_TTL` | `30m` |
| `SESSION_CLEANUP_INTERVAL` | `1m` |
| `SECURE_COOKIES` | `false` локально, `true` с HTTPS |

## Проверка

```bash
go test ./...
go vet ./...
```
