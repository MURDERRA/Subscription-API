# Subscription Service (Go)

REST API управления подписками пользователей.

## Стек

| Компонент     | Инструмент                              |
|---------------|-----------------------------------------|
| HTTP          | Gin                                     |
| БД            | PostgreSQL 16                           |
| Драйвер       | pgx/v5 (stdlib-адаптер)                |
| Миграции      | golang-migrate (SQL встроен в бинарь)   |
| Логирование   | `log/slog` → JSON → stdout + файл      |
| Документация  | Swagger UI + openapi.yaml               |

## Быстрый старт

```fish
cp .env.example .env
docker compose up --build
```

- Swagger UI → http://localhost:8000/docs/
- Healthcheck → http://localhost:8000/health

Логи пишутся в `./logs/app.log` (JSON, один объект на строку).  
Пример записи:
```json
{"time":"2025-07-01T12:00:00Z","level":"INFO","msg":"подписка создана","id":"..."}
```

## Локальная разработка (без Docker)

```fish
# Нужен PostgreSQL на localhost:5432
cp .env.example .env
# Отредактировать DATABASE_URL: заменить @db на @localhost

go mod tidy        # генерирует go.sum
go run ./cmd/server
```

> **Первый `docker compose build`** требует `go.sum`.  
> Сгенерируйте его локально: `go mod tidy`

## API

### Формат дат

Все даты передаются строкой **`MM-YYYY`** (например `07-2025`).  
В БД хранятся как `DATE` (1-е число месяца).

### Эндпоинты

| Метод    | URL                                    | Описание                           |
|----------|----------------------------------------|------------------------------------|
| `POST`   | `/api/v1/subscriptions`               | Создать подписку                   |
| `GET`    | `/api/v1/subscriptions`               | Список (фильтры: user_id, service_name, limit, offset) |
| `GET`    | `/api/v1/subscriptions/total-cost`    | Суммарная стоимость за период      |
| `GET`    | `/api/v1/subscriptions/:id`           | Получить по ID                     |
| `PUT`    | `/api/v1/subscriptions/:id`           | Обновить                           |
| `DELETE` | `/api/v1/subscriptions/:id`           | Удалить                            |

### Примеры

```fish
# Создать
curl -X POST http://localhost:8000/api/v1/subscriptions \
  -H 'Content-Type: application/json' \
  -d '{
    "service_name": "Yandex Plus",
    "price": 400,
    "user_id": "60601fee-2bf1-4721-ae6f-7636e79a0cba",
    "start_date": "07-2025"
  }'

# Суммарная стоимость за год
curl 'http://localhost:8000/api/v1/subscriptions/total-cost?period_start=01-2025&period_end=12-2025&user_id=60601fee-2bf1-4721-ae6f-7636e79a0cba'

# Убрать дату окончания (end_date: null)
curl -X PUT http://localhost:8000/api/v1/subscriptions/<UUID> \
  -H 'Content-Type: application/json' \
  -d '{"end_date": null}'
```

### Логика подсчёта стоимости

```
total = Σ (price × кол-во месяцев активности в периоде)
```

Подписка попадает в выборку если:
- `start_date ≤ period_end`
- `end_date IS NULL OR end_date ≥ period_start`

## Структура проекта

```
sub-go/
├── cmd/server/main.go              # точка входа
├── internal/
│   ├── config/config.go            # конфиг из .env
│   ├── logger/logger.go            # JSON slog → stdout + файл
│   ├── db/db.go                    # connect + migrate
│   ├── model/subscription.go       # типы данных, DTO
│   ├── repository/subscription.go  # SQL-запросы
│   ├── service/subscription.go     # бизнес-логика
│   └── handler/subscription.go     # HTTP-хендлеры, роутинг
├── migrations/
│   ├── migrations.go               # embed FS
│   ├── 000001_init.up.sql
│   └── 000001_init.down.sql
├── docs/openapi.yaml               # OpenAPI 3.0 спецификация
├── logs/                           # → монтируется из Docker
├── docker-compose.yml
├── Dockerfile
└── .env.example
```
