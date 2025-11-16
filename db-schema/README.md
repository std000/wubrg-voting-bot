# Схема данных для WUBRG Voting Bot

Эта папка содержит схему базы данных PostgreSQL для бота голосований в Telegram.

## Кастомная схема PostgreSQL

Все таблицы размещены в отдельной схеме `voting`, а не в стандартной `public`. Это обеспечивает:
- Лучшую организацию базы данных
- Изоляцию от других приложений
- Простоту управления правами доступа

Во всех запросах используются явные пути: `voting.polls`, `voting.votes` и т.д.

## Структура таблиц

### 1. `polls` - Голосования
Основная таблица с информацией о голосованиях.

**Поля:**
- `id` - уникальный идентификатор голосования
- `title` - название голосования (обязательное)
- `description` - описание голосования (опционально)
- `creator_telegram_id` - Telegram ID создателя
- `creator_username` - username создателя
- `is_active` - активно ли голосование
- `created_at` - дата создания
- `updated_at` - дата последнего обновления
- `expires_at` - дата окончания (опционально)

### 2. `poll_options` - Варианты ответов
Таблица с вариантами ответов для каждого голосования.

**Поля:**
- `id` - уникальный идентификатор варианта
- `poll_id` - ссылка на голосование
- `option_text` - текст варианта ответа
- `created_at` - дата создания

**Связи:**
- `poll_id` → `polls.id` (ON DELETE CASCADE)

### 3. `poll_chats` - Чаты с голосованиями
Таблица для отслеживания, в какие чаты было опубликовано голосование.

**Поля:**
- `id` - уникальный идентификатор записи
- `poll_id` - ссылка на голосование
- `chat_id` - Telegram ID чата
- `message_id` - ID сообщения в чате
- `created_at` - дата публикации (автоматически устанавливается)

**Связи:**
- `poll_id` → `polls.id` (ON DELETE CASCADE)

### 4. `votes` - Голоса пользователей
Таблица с информацией о голосах пользователей.

**Поля:**
- `id` - уникальный идентификатор голоса
- `poll_id` - ссылка на голосование
- `option_id` - ссылка на выбранный вариант
- `user_telegram_id` - Telegram ID пользователя
- `user_username` - username пользователя
- `user_first_name` - имя пользователя
- `user_last_name` - фамилия пользователя
- `voted_at` - дата и время голоса

**Связи:**
- `poll_id` → `polls.id` (ON DELETE CASCADE)
- `option_id` → `poll_options.id` (ON DELETE CASCADE)

**Ограничения:**
- Уникальная комбинация `(poll_id, user_telegram_id)` - 
  пользователь может проголосовать в голосовании только один раз

## Установка схемы

### Создание таблиц
```bash
psql -U postgres -d wubrg_voting -f db-schema/schema.sql
```

### Удаление всех таблиц
```bash
psql -U postgres -d wubrg_voting -f db-schema/drop_tables.sql
```

### Создание тестовых данных
```bash
psql -U postgres -d wubrg_voting -f db-schema/sample_data.sql
```

## Примеры запросов

### Получить все активные голосования пользователя
```sql
SELECT * FROM polls 
WHERE creator_telegram_id = 123456789 
  AND is_active = true 
ORDER BY created_at DESC;
```

### Получить результаты голосования
```sql
SELECT 
    po.option_text,
    COUNT(v.id) as vote_count
FROM poll_options po
LEFT JOIN votes v ON po.id = v.option_id
WHERE po.poll_id = 1
GROUP BY po.id, po.option_text, po.option_order
ORDER BY po.option_order;
```

### Получить всех проголосовавших (для неанонимного голосования)
```sql
SELECT 
    v.user_first_name,
    v.user_last_name,
    v.user_username,
    po.option_text,
    v.voted_at
FROM votes v
JOIN poll_options po ON v.option_id = po.id
WHERE v.poll_id = 1
ORDER BY v.voted_at DESC;
```

### Проверить, голосовал ли пользователь
```sql
SELECT EXISTS(
    SELECT 1 FROM votes 
    WHERE poll_id = 1 
      AND user_telegram_id = 123456789
) as has_voted;
```

## Особенности схемы

1. **Каскадное удаление** - при удалении голосования автоматически удаляются все связанные варианты ответов, голоса и записи о чатах.

2. **Индексы** - созданы индексы для часто используемых полей для ускорения запросов.

3. **Уникальные ограничения** - предотвращают дублирование голосов и вариантов ответов.

4. **Гибкость типов** - использование TEXT вместо VARCHAR для полей с текстом (PostgreSQL оптимизирован для TEXT).

5. **Временные ограничения** - поле `expires_at` позволяет автоматически завершать голосования.

