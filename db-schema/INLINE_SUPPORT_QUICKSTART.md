# Быстрый старт: Поддержка inline-режима

## Что было добавлено?

Теперь бот отслеживает, когда голосования отправляются через inline-режим (`@bot_name vote`).

## Применение миграции

```bash
cd /path/to/wubrg-voting-bot
psql -U your_username -d your_database -f db-schema/add_inline_support_to_poll_chats.sql
```

Или если база данных на вашем dev окружении:

```bash
psql voting_bot < db-schema/add_inline_support_to_poll_chats.sql
```

## Что изменилось?

### Таблица `voting.poll_chats`

**Было:**
```sql
poll_id    | chat_id  | message_id
-----------|----------|------------
1          | -100123  | 456
```

**Стало:**
```sql
poll_id | chat_id  | message_id | inline_message_id | message_hash
--------|----------|------------|-------------------|-------------
1       | -100123  | 456        | NULL              | NULL
2       | NULL     | NULL       | AgAAA...          | 1234567890
```

- **Обычная публикация**: `chat_id` и `message_id` заполнены, `inline_message_id` = NULL
- **Inline-публикация**: `inline_message_id` и `message_hash` заполнены, `chat_id` и `message_id` = NULL

### Код

**bot.go:**
- Добавлен обработчик `OnChosenInlineResult`

**poll.go:**
- Добавлена функция `handleChosenInlineResult()` - сохраняет информацию об inline-публикациях
- Добавлена функция `FastHash()` - быстрая хеш-функция для идентификации

## Проверка работы

1. Перезапустите бота после применения миграции
2. В любом чате напишите `@your_bot_name vote`
3. Выберите голосование и отправьте
4. Проверьте логи бота:
```
✅ Inline-голосование 123 отправлено пользователем 456 (inline_msg_id=AgAAA..., hash=1234567890)
```

5. Проверьте в БД:
```sql
SELECT poll_id, inline_message_id, message_hash, created_at 
FROM voting.poll_chats 
WHERE inline_message_id IS NOT NULL 
ORDER BY created_at DESC 
LIMIT 5;
```

## Для чего это нужно?

- **Аналитика**: Теперь можно отслеживать, сколько раз голосование было опубликовано через inline
- **Статистика**: Можно узнать, какие голосования чаще всего шарят через inline
- **Хеш**: Уникальный идентификатор для каждой публикации

## Откат

Если нужно откатить (см. подробности в `APPLY_INLINE_SUPPORT.md`):

```sql
BEGIN;
DROP INDEX IF EXISTS voting.unique_poll_inline_message;
DROP INDEX IF EXISTS voting.unique_poll_chat_message;
DROP INDEX IF EXISTS voting.idx_poll_chats_message_hash;
ALTER TABLE voting.poll_chats DROP COLUMN IF EXISTS inline_message_id;
ALTER TABLE voting.poll_chats DROP COLUMN IF EXISTS message_hash;
ALTER TABLE voting.poll_chats ALTER COLUMN chat_id SET NOT NULL;
ALTER TABLE voting.poll_chats ALTER COLUMN message_id SET NOT NULL;
ALTER TABLE voting.poll_chats ADD CONSTRAINT unique_poll_chat_message 
    UNIQUE (poll_id, chat_id, message_id);
COMMIT;
```

