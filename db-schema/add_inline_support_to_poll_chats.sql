-- Миграция: добавление поддержки inline-режима и message_hash в таблицу poll_chats
-- Дата: 2025-11-20
-- Автор: wubrg-voting-bot

BEGIN;

-- 1. Удаляем старый UNIQUE constraint
ALTER TABLE voting.poll_chats DROP CONSTRAINT IF EXISTS unique_poll_chat_message;

-- 2. Делаем chat_id и message_id nullable (для inline-сообщений они будут NULL)
ALTER TABLE voting.poll_chats ALTER COLUMN chat_id DROP NOT NULL;
ALTER TABLE voting.poll_chats ALTER COLUMN message_id DROP NOT NULL;

-- 3. Добавляем новые поля
ALTER TABLE voting.poll_chats ADD COLUMN IF NOT EXISTS inline_message_id TEXT;
ALTER TABLE voting.poll_chats ADD COLUMN IF NOT EXISTS message_hash BIGINT;

-- 4. Добавляем UNIQUE constraint для inline-сообщений
CREATE UNIQUE INDEX IF NOT EXISTS unique_poll_inline_message 
    ON voting.poll_chats(poll_id, inline_message_id) 
    WHERE inline_message_id IS NOT NULL;

-- 5. Добавляем UNIQUE constraint для обычных сообщений
CREATE UNIQUE INDEX IF NOT EXISTS unique_poll_chat_message 
    ON voting.poll_chats(poll_id, chat_id, message_id) 
    WHERE chat_id IS NOT NULL AND message_id IS NOT NULL;

-- 6. Создаём индекс для message_hash
CREATE INDEX IF NOT EXISTS idx_poll_chats_message_hash ON voting.poll_chats(message_hash) WHERE message_hash IS NOT NULL;

-- 7. Обновляем комментарий к таблице
COMMENT ON TABLE voting.poll_chats IS 'Чаты и inline-сообщения, куда были опубликованы голосования';
COMMENT ON COLUMN voting.poll_chats.inline_message_id IS 'ID inline-сообщения (если голосование отправлено через inline-режим)';
COMMENT ON COLUMN voting.poll_chats.message_hash IS 'Хеш для дополнительной идентификации сообщения';

COMMIT;

