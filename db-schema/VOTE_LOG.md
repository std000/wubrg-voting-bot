# Таблица vote_log - логирование нажатий на кнопки

## Описание

Таблица `voting.vote_log` предназначена для логирования всех нажатий пользователей на кнопки голосования в режиме **append-only** (только добавление записей).

В отличие от таблицы `voting.votes`, которая хранит текущее состояние голосов (один голос на пользователя), таблица `vote_log` записывает каждое нажатие, включая случаи когда пользователь переголосовывает.

## Структура таблицы

```sql
CREATE TABLE voting.vote_log (
    id BIGSERIAL PRIMARY KEY,
    user_telegram_id BIGINT NOT NULL,             -- Telegram ID пользователя
    poll_id BIGINT NOT NULL,                      -- ID голосования
    option_id BIGINT NOT NULL,                    -- ID выбранного варианта
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Время нажатия
);
```

## Особенности

1. **Append-only**: записи только добавляются, никогда не удаляются и не обновляются
2. **Без индексов**: таблица создана без индексов для максимальной скорости записи
3. **Полная история**: все действия пользователей записываются в хронологическом порядке

## Применение к базе данных

```bash
# Применить миграцию к существующей базе
psql -U postgres -d wubrg_voting -f db-schema/add_vote_log_table.sql
```

## Примеры запросов

### Посмотреть последние 10 нажатий

```sql
SELECT 
    vl.clicked_at,
    vl.user_telegram_id,
    p.title as poll_title,
    po.option_text
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
JOIN voting.poll_options po ON po.id = vl.option_id
ORDER BY vl.clicked_at DESC
LIMIT 10;
```

### Количество нажатий по каждому опросу

```sql
SELECT 
    poll_id,
    COUNT(*) as total_clicks,
    COUNT(DISTINCT user_telegram_id) as unique_users
FROM voting.vote_log
GROUP BY poll_id;
```

### Найти самых активных пользователей

```sql
SELECT 
    user_telegram_id,
    COUNT(*) as total_clicks
FROM voting.vote_log
WHERE poll_id = $1
GROUP BY user_telegram_id
ORDER BY total_clicks DESC;
```

### Временная динамика голосований

```sql
SELECT 
    DATE_TRUNC('hour', clicked_at) as hour,
    COUNT(*) as clicks_count,
    COUNT(DISTINCT user_telegram_id) as unique_users
FROM voting.vote_log
WHERE poll_id = $1
GROUP BY hour
ORDER BY hour;
```

## Обслуживание

Так как таблица работает в режиме append-only и не имеет индексов, она может расти очень быстро при активном использовании. Рекомендуется периодически:

1. Анализировать размер таблицы:
```sql
SELECT pg_size_pretty(pg_total_relation_size('voting.vote_log'));
```

2. Архивировать старые данные (опционально):
```sql
-- Создать архивную таблицу
CREATE TABLE voting.vote_log_archive_2025 AS 
SELECT * FROM voting.vote_log 
WHERE clicked_at < '2025-01-01';

-- Удалить архивированные данные
DELETE FROM voting.vote_log 
WHERE clicked_at < '2025-01-01';
```

