# Система постов и комментариев с GraphQL

Проект реализует GraphQL API для ведения постов и иерархических комментариев, аналогично платформам вроде Хабра или Reddit. Система написана на Go, поддерживает хранение в памяти или PostgreSQL.

## Реализованные требования

- Просмотр списка постов и отдельного поста с комментариями.
- Автор поста может запретить комментирование.
- Древовидные комментарии с неограниченной вложенностью.
- Ограничение длины комментария (по умолчанию 2000 символов, настраивается через конфиг).
- Курсорная пагинация для постов и комментариев.
- GraphQL Subscriptions – клиенты, подписанные на пост, получают новые комментарии в реальном времени через WebSocket.
- Выбор хранилища параметром запуска: `memory` или `postgres`.
- Полное покрытие unit-тестами (доменные модели, сервисы, репозитории, DataLoader, подписки).
- Интеграционные тесты для проверки работы graphQL.
- Оптимизация N+1 запросов с помощью DataLoader для дочерних комментариев.
- Транзакционность при создании/изменении сущностей.

## Конфигурация

Все параметры задаются через переменные окружения. В коде уже есть значения по умолчанию, обязательным является только `PORT`.

| Переменная окружения               | Описание                                    | По умолчанию |
|------------------------------------|---------------------------------------------|--------------|
| `PORT`                             | Порт сервера                                | *обязательно*|
| `SHUTDOWN_TIMEOUT`                 | Таймаут graceful shutdown                   | `30s`        |
| `STORAGE_TYPE`                     | Тип хранилища (`memory` или `postgres`)     | `memory`     |
| `DATABASE_URL`                     | URL подключения к PostgreSQL                |              |
| `DATABASE_CONNECTION_TIMEOUT`      | Таймаут подключения к БД                     | `30s`        |
| `DOMAIN_MAX_COMMENT_LENGTH`        | Максимальная длина комментария              | `2000`       |
| `DOMAIN_MAX_POST_TITLE_LENGTH`     | Максимальная длина заголовка поста          | `200`        |
| `DOMAIN_MAX_POST_CONTENT_LENGTH`   | Максимальная длина содержимого поста        | `10000`      |
| `DOMAIN_MAX_COMMENT_PAGE_SIZE`     | Максимальный размер страницы комментариев   | `100`        |
| `DOMAIN_MAX_POST_PAGE_SIZE`        | Максимальный размер страницы постов         | `100`        |
| `SUBSCRIPTION_BUFFER_SIZE`         | Размер буфера канала подписок               | `100`        |
| `GRAPHQL_MAX_COMPLEXITY`           | Максимальная сложность запроса              | `7`          |
| `GRAPHQL_QUERY_CACHE_SIZE`         | Размер кэша запросов GraphQL                | `100`        |
| `GRAPHQL_APQ_CACHE_SIZE`           | Размер кэша Automatic Persisted Queries     | `1000`       |
| `GRAPHQL_HANDSHAKE_TIMEOUT`        | Таймаут WebSocket handshake                 | `10s`        |
| `GRAPHQL_KEEP_ALIVE_PING_INTERVAL` | Интервал keep-alive пингов WebSocket        | `30s`        |
| `HTTP_SERVER_READ_TIMEOUT`         | Read timeout HTTP сервера                    | `20s`        |
| `HTTP_SERVER_WRITE_TIMEOUT`        | Write timeout HTTP сервера                   | `20s`        |
| `HTTP_SERVER_IDLE_TIMEOUT`         | Idle timeout HTTP сервера                    | `20s`        |
| `LOG_LEVEL`                        | Уровень логирования (`debug`, `info`, `warn`, `error`) | `info`       |
| `LOG_FORMAT`                       | Формат логов (`text`, `json`)                | `text`       |
| `DATALOADER_BATCH_CAPACITY`        | Емкость батчинга DataLoader                 | `1000`       |
| `DATALOADER_WAIT_TIME`             | Время ожидания накопления ключей DataLoader | `2ms`        |

## Запуск

### Docker Compose

```bash
git clone https://github.com/TimofeyChernyshev/comment-post-system

cd comment-post-system

docker-compose up --build
```

## GraphQL schema

Схема расположена [тут](internal/infrastructure/graphql/graph/schema.graphqls)

## Makefile

`make test` - позволяет запустить все тесты и выводит в файл `coverage.out` покрытие

## Архитектура

Чистая архитектура с разделением на слои domaim, application, infrastructure, pkg.