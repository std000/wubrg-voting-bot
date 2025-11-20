# Применение таблицы vote_log

## Быстрый старт

### 1. Применить миграцию к базе данных

Выполните один из следующих вариантов:

**Вариант A - используя psql:**
```bash
psql -U postgres -d wubrg_voting -f db-schema/add_vote_log_table.sql
```

**Вариант B - используя переменную окружения DATABASE_URL:**
```bash
psql $DATABASE_URL -f db-schema/add_vote_log_table.sql
```

**Вариант C - прямое подключение:**
```bash
psql "postgresql://postgres:postgres@localhost:5432/wubrg_voting" -f db-schema/add_vote_log_table.sql
```

### 2. Перезапустить бота

После применения миграции перезапустите бота:

```bash
# Остановите текущий экземпляр бота (Ctrl+C)

# Запустите снова
export BOT_TOKEN="your_bot_token"
export DATABASE_URL="postgresql://postgres:postgres@localhost:5432/wubrg_voting"
./wubrg-voting-bot
```

### 3. Проверить работу

После перезапуска бота, каждое нажатие на кнопку голосования будет автоматически записываться в таблицу `voting.vote_log`.

Проверить можно запросом:
```sql
SELECT COUNT(*) FROM voting.vote_log;
```

## Что было изменено

### 1. Схема базы данных

- ✅ Добавлена таблица `voting.vote_log` в `db-schema/schema.sql`
- ✅ Создан файл миграции `db-schema/add_vote_log_table.sql`

### 2. Код бота

- ✅ Изменен файл `bot/poll.go`, функция `handleVote()`
- ✅ Добавлено логирование каждого нажатия на кнопку
- ✅ Различение между первым голосом (`vote`) и переголосованием (`revote`)

### 3. Документация

- ✅ `db-schema/VOTE_LOG.md` - описание таблицы и примеры использования
- ✅ `db-schema/vote_log_queries.sql` - готовые SQL-запросы для анализа

## Особенности таблицы vote_log

1. **Append-only** - записи только добавляются, никогда не изменяются
2. **Без индексов** - максимальная скорость записи
3. **Полная история** - все нажатия сохраняются
4. **Минимальные данные** - только telegram_id, poll_id, option_id, время

## Примеры анализа данных

### Посмотреть последние 10 нажатий:
```sql
SELECT 
    vl.clicked_at,
    vl.user_telegram_id,
    p.title,
    po.option_text
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
JOIN voting.poll_options po ON po.id = vl.option_id
ORDER BY vl.clicked_at DESC
LIMIT 10;
```

### Статистика нажатий:
```sql
SELECT 
    COUNT(*) as total_clicks,
    COUNT(DISTINCT user_telegram_id) as unique_users
FROM voting.vote_log;
```

Больше примеров в файле `db-schema/vote_log_queries.sql`.

## Если что-то пошло не так

### Откатить изменения

Если нужно удалить таблицу:
```sql
DROP TABLE IF EXISTS voting.vote_log;
```

### Пересоздать таблицу

```sql
DROP TABLE IF EXISTS voting.vote_log;
-- Затем заново применить миграцию
\i db-schema/add_vote_log_table.sql
```

