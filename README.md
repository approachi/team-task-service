# team-task-service

REST API сервис для управления командами и задачами: регистрация/логин по JWT, приглашение участников в команду, CRUD задач с фильтрами и пагинацией, аудит изменений и SQL-аналитика по командам.

## Требования

- [Docker](https://docs.docker.com/get-docker/) и Docker Compose
- [Go 1.25+](https://golang.org/dl/) (для запуска тестов локально)
- [Make](https://www.gnu.org/software/make/)

## Запуск

**1. Клонируйте репозиторий и скопируйте файл с локальными секретами:**
```bash
git clone <repo-url>
cd team-task-service
cp .env.example .env
```

`.env` — только для локальной разработки (уже в `.gitignore`); в проде секреты выдаёт секрет-менеджер, а не файл в репозитории — см. [docs/COVER_LETTER.md](docs/COVER_LETTER.md), раздел "Секреты".

**2. Запустите сервис:**
```bash
make run
```

Команда поднимает MySQL и Redis, дожидается их готовности, применяет миграции (`make migrate-up`) и только затем собирает и стартует `app` — миграция всегда отрабатывает до того, как контейнер приложения примет первый запрос. Миграции сознательно оформлены как собственный шаг (`goose`), а не встроены в код приложения — см. [docs/COVER_LETTER.md](docs/COVER_LETTER.md), раздел "Инфраструктура кода".

**3. Проверьте, что сервис работает:**
```bash
curl -v -X POST http://localhost:8080/api/v1/register \
  -H "Content-Type: application/json" \
  -d '{"email": "user@example.com", "password": "password123", "name": "Alice"}'
```

Ожидаемый ответ: `HTTP 201`.

**4. Остановка:**
```bash
make down
```

Swagger UI доступен на `http://localhost:8080/swagger/index.html`, метрики Prometheus — на `http://localhost:8080/metrics`.

## API

Аутентифицированные эндпоинты требуют заголовок `Authorization: Bearer <token>`, полученный из `/login`.

### Аутентификация

```
POST /api/v1/register
POST /api/v1/login
```

**Запрос `/register`:**
```json
{
  "email": "user@example.com",
  "password": "password123",
  "name": "Alice"
}
```

### Команды

```
POST /api/v1/teams              — создать команду (вызывающий становится owner)
GET  /api/v1/teams              — список команд вызывающего
POST /api/v1/teams/{id}/invite  — пригласить существующего пользователя в команду
```

**Запрос `/teams`:**
```json
{ "name": "Backend Team" }
```

**Запрос `/teams/{id}/invite`:**
```json
{ "email": "invitee@example.com" }
```

### Задачи

```
POST /api/v1/tasks              — создать задачу
GET  /api/v1/tasks               — список задач команды (фильтры + пагинация)
PUT  /api/v1/tasks/{id}          — частичное обновление задачи
GET  /api/v1/tasks/{id}/history  — аудит изменений задачи
```

**Запрос `/tasks` (POST):**
```json
{
  "team_id": 1,
  "title": "Fix login bug",
  "description": "Users can't log in with uppercase emails",
  "assignee_to": 2
}
```

**Query-параметры `/tasks` (GET):** `team_id` (обязателен), `status` (`todo`/`in_progress`/`done`), `assignee_to`, `page`, `page_size` (по умолчанию 20, максимум 100).

### Отчёты

```
GET /api/v1/reports/teams-summary       — участники и закрытые за 7 дней задачи по каждой команде
GET /api/v1/reports/top-creators        — топ-3 автора задач по команде за месяц
GET /api/v1/reports/orphaned-assignees  — задачи, назначенные не на участника команды
```

Все три отчёта скопированы на команды вызывающего пользователя.

### Ответы

| Статус | Причина                                                                |
|--------|-------------------------------------------------------------------------|
| 200/201| Успешно                                                                  |
| 400    | Невалидный запрос (значение поля, отсутствующий обязательный параметр)  |
| 401    | Не авторизован (нет/невалидный JWT)                                     |
| 403    | Нет прав на это действие в команде                                      |
| 404    | Команда/задача не найдена                                               |
| 409    | Конфликт (email уже зарегистрирован, уже приглашён и т.п.)              |
| 500    | Внутренняя ошибка сервера                                               |

## Тестирование

Все тесты:
```bash
make test
```

Юнит-тесты:
```bash
make test-unit
```

Интеграционные тесты (используют `testcontainers-go`, требуют локальный Docker):
```bash
make test-integration
```

## Make-команды

| Команда                 | Описание                                     |
|--------------------------|-----------------------------------------------|
| `make run`               | Собрать и запустить все сервисы                |
| `make down`              | Остановить все сервисы                         |
| `make build`             | Собрать бинарник в `bin/`                      |
| `make test-unit`         | Запустить юнит-тесты                           |
| `make test-integration`  | Запустить интеграционные тесты                 |
| `make test`              | Запустить все тесты                            |
| `make db-conn`           | Подключиться к локальной БД                    |
| `make migrate-up`        | Применить все накопленные миграции             |
| `make migrate-down`      | Откатить последнюю миграцию                    |
| `make migrate-status`    | Показать применённые/ожидающие миграции        |
| `make lint`              | `golangci-lint run ./...`                      |
| `make swagger`           | Перегенерировать Swagger-спеку из аннотаций хендлеров |

---

Выбор технологий, рекомендации для production и инфраструктура описаны в [docs/COVER_LETTER.md](docs/COVER_LETTER.md).
