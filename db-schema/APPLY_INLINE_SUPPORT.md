# Применение миграции: Поддержка inline-режима в poll_chats

## Дата: 2025-11-20

## Описание

Эта миграция добавляет поддержку inline-режима в таблицу `voting.poll_chats`:
- Добавляет поле `inline_message_id` для хранения ID inline-сообщений
- Добавляет поле `message_hash` (BIGINT) для дополнительной идентификации
- Делает поля `chat_id` и `message_id` nullable

## Файлы

- **Миграция**: `add_inline_support_to_poll_chats.sql`
- **Обновленная схема**: `schema.sql`

## Применение миграции

### Способ 1: Через psql

```bash
psql -U your_username -d your_database -f db-schema/add_inline_support_to_poll_chats.sql
```

### Способ 2: Через интерактивный psql

```bash
psql -U your_username -d your_database
```

Затем в psql:

```sql
\i db-schema/add_inline_support_to_poll_chats.sql
```

## Проверка миграции

После применения миграции проверьте структуру таблицы:

```sql
\d+ voting.poll_chats
```

Ожидаемый результат:

```
                                           Table "voting.poll_chats"
      Column       |           Type           | Nullable |              Default              
-------------------+--------------------------+----------+-----------------------------------
 id                | bigint                   | not null | nextval('...')
 poll_id           | bigint                   | not null | 
 chat_id           | bigint                   |          | 
 message_id        | bigint                   |          | 
 inline_message_id | text                     |          | 
 message_hash      | bigint                   |          | 
 created_at        | timestamp with time zone | not null | now()
Indexes:
    "poll_chats_pkey" PRIMARY KEY, btree (id)
    "unique_poll_chat_message" UNIQUE, btree (poll_id, chat_id, message_id) WHERE chat_id IS NOT NULL AND message_id IS NOT NULL
    "unique_poll_inline_message" UNIQUE, btree (poll_id, inline_message_id) WHERE inline_message_id IS NOT NULL
    "idx_poll_chats_chat_id" btree (chat_id) WHERE chat_id IS NOT NULL
    "idx_poll_chats_message_hash" btree (message_hash) WHERE message_hash IS NOT NULL
    "idx_poll_chats_message_id" btree (chat_id, message_id) WHERE chat_id IS NOT NULL AND message_id IS NOT NULL
    "idx_poll_chats_poll_id" btree (poll_id)
```

## Как работает

### Обычная публикация (/publishpoll)

Запись выглядит так:
```sql
INSERT INTO voting.poll_chats (poll_id, chat_id, message_id) 
VALUES (1, -1001234567890, 123);
```

### Inline-публикация (@bot_name vote)

Когда пользователь отправляет голосование через inline-режим:

```sql
INSERT INTO voting.poll_chats (poll_id, inline_message_id, message_hash) 
VALUES (1, 'AgAAABIDAAJDEQAC', 1234567890123);
```

## Изменения в коде

### bot.go

Добавлен обработчик:
```go
b.bot.Handle(telebot.OnChosenInlineResult, b.handleChosenInlineResult)
```

### poll.go

Добавлены функции:
- `handleChosenInlineResult(c telebot.Context) error` - обрабатывает выбор inline-результата
- `FastHash(s string) uint64` - быстрая хеш-функция для строк

## Откат миграции

Если нужно откатить миграцию:

```sql
BEGIN;

-- Удаляем новые индексы
DROP INDEX IF EXISTS voting.unique_poll_inline_message;
DROP INDEX IF EXISTS voting.unique_poll_chat_message;
DROP INDEX IF EXISTS voting.idx_poll_chats_message_hash;

-- Удаляем новые столбцы
ALTER TABLE voting.poll_chats DROP COLUMN IF EXISTS inline_message_id;
ALTER TABLE voting.poll_chats DROP COLUMN IF EXISTS message_hash;

-- Возвращаем NOT NULL для старых полей
ALTER TABLE voting.poll_chats ALTER COLUMN chat_id SET NOT NULL;
ALTER TABLE voting.poll_chats ALTER COLUMN message_id SET NOT NULL;

-- Возвращаем старый constraint
ALTER TABLE voting.poll_chats ADD CONSTRAINT unique_poll_chat_message 
    UNIQUE (poll_id, chat_id, message_id);

COMMIT;
```

## Тестирование

После применения миграции и перезапуска бота:

1. **Тест inline-режима**:
   - Откройте любой чат
   - Напишите `@your_bot_name vote`
   - Выберите голосование
   - Отправьте в чат
   - Проверьте логи бота - должна быть запись: `✅ Inline-голосование X отправлено пользователем Y (inline_msg_id=..., hash=...)`

2. **Проверка в БД**:
```sql
SELECT * FROM voting.poll_chats ORDER BY created_at DESC LIMIT 5;
```

Вы должны увидеть записи с заполненным `inline_message_id` и `message_hash`, но с NULL в `chat_id` и `message_id`.

3. **Тест обычной публикации** (не должен сломаться):
   - `/createpoll` - создайте голосование
   - `/publishpoll <ID>` - опубликуйте в чат
   - Проверьте БД - должна быть запись с заполненными `chat_id` и `message_id`, но с NULL в `inline_message_id`

