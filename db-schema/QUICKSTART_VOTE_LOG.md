# Быстрый старт: vote_log

## Что это?

Таблица `vote_log` логирует **все** нажатия на кнопки голосования в append-only формате (без индексов).

## Структура

```sql
CREATE TABLE voting.vote_log (
    id BIGSERIAL PRIMARY KEY,
    user_telegram_id BIGINT NOT NULL,
    poll_id BIGINT NOT NULL,
    option_id BIGINT NOT NULL,
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Применить к БД

```bash
psql $DATABASE_URL -f db-schema/add_vote_log_table.sql
```

или

```bash
psql -U postgres -d wubrg_voting -f db-schema/add_vote_log_table.sql
```

## Как работает

1. **Каждое** нажатие на кнопку голосования логируется в начале транзакции
2. Затем обновляется основная таблица `votes` (текущее состояние голоса)
3. Если пользователь переголосовывает - создается новая запись в `vote_log`

## Примеры запросов

### Последние 10 нажатий:
```sql
SELECT * FROM voting.vote_log 
ORDER BY clicked_at DESC 
LIMIT 10;
```

### Количество нажатий по опросам:
```sql
SELECT 
    poll_id,
    COUNT(*) as clicks,
    COUNT(DISTINCT user_telegram_id) as users
FROM voting.vote_log
GROUP BY poll_id;
```

### Пользователи, которые кликали много раз:
```sql
SELECT 
    user_telegram_id,
    COUNT(*) as total_clicks
FROM voting.vote_log
WHERE poll_id = 123  -- ваш poll_id
GROUP BY user_telegram_id
HAVING COUNT(*) > 1
ORDER BY total_clicks DESC;
```

Больше примеров: `db-schema/vote_log_queries.sql`

