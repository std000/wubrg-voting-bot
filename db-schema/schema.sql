-- Схема данных для бота голосований WUBRG

-- Создание кастомной схемы
CREATE SCHEMA IF NOT EXISTS voting;

-- Таблица голосований
CREATE TABLE IF NOT EXISTS voting.polls (
    id BIGSERIAL PRIMARY KEY,
    title TEXT NOT NULL,                               -- Название голосования
    description TEXT,                                  -- Описание голосования (опционально)
    creator_telegram_id BIGINT NOT NULL,              -- Telegram ID создателя
    creator_username TEXT,                             -- Username создателя (опционально)
    is_active BOOLEAN DEFAULT true,                    -- Активно ли голосование
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),     -- Дата создания
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),     -- Дата последнего обновления
    expires_at TIMESTAMPTZ                             -- Дата окончания голосования (опционально)
);

-- Индексы для таблицы polls
CREATE INDEX IF NOT EXISTS idx_polls_creator ON voting.polls(creator_telegram_id);
CREATE INDEX IF NOT EXISTS idx_polls_is_active ON voting.polls(is_active);
CREATE INDEX IF NOT EXISTS idx_polls_created_at ON voting.polls(created_at DESC);

-- Таблица вариантов ответов
CREATE TABLE IF NOT EXISTS voting.poll_options (
    id BIGSERIAL PRIMARY KEY,
    poll_id BIGINT NOT NULL REFERENCES voting.polls(id) ON DELETE CASCADE,  -- ID голосования
    option_text TEXT NOT NULL,                                       -- Текст варианта ответа
    emoji TEXT,                                                      -- Эмодзи для визуализации голосов (nullable, по умолчанию 👍 в коде)
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Индексы для таблицы poll_options
CREATE INDEX IF NOT EXISTS idx_poll_options_poll_id ON voting.poll_options(poll_id);

-- Таблица чатов и inline-сообщений, куда запостили голосование
CREATE TABLE IF NOT EXISTS voting.poll_chats (
    id BIGSERIAL PRIMARY KEY,
    poll_id BIGINT NOT NULL REFERENCES voting.polls(id) ON DELETE CASCADE,  -- ID голосования
    chat_id BIGINT,                                                  -- ID чата Telegram (NULL для inline)
    message_id BIGINT,                                               -- ID сообщения в чате (NULL для inline)
    inline_message_id TEXT,                                          -- ID inline-сообщения (NULL для обычных)
    message_hash BIGINT,                                             -- Хеш для идентификации
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()                    -- Дата публикации
);

-- Индексы для таблицы poll_chats
CREATE INDEX IF NOT EXISTS idx_poll_chats_poll_id ON voting.poll_chats(poll_id);
CREATE INDEX IF NOT EXISTS idx_poll_chats_chat_id ON voting.poll_chats(chat_id) WHERE chat_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_poll_chats_message_id ON voting.poll_chats(chat_id, message_id) WHERE chat_id IS NOT NULL AND message_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_poll_chats_message_hash ON voting.poll_chats(message_hash) WHERE message_hash IS NOT NULL;

-- Уникальные индексы для разных типов сообщений
CREATE UNIQUE INDEX IF NOT EXISTS unique_poll_inline_message 
    ON voting.poll_chats(poll_id, inline_message_id) 
    WHERE inline_message_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS unique_poll_chat_message 
    ON voting.poll_chats(poll_id, chat_id, message_id) 
    WHERE chat_id IS NOT NULL AND message_id IS NOT NULL;

-- Таблица с проголосовавшими
CREATE TABLE IF NOT EXISTS voting.votes (
    id BIGSERIAL PRIMARY KEY,
    poll_id BIGINT NOT NULL REFERENCES voting.polls(id) ON DELETE CASCADE,           -- ID голосования
    option_id BIGINT NOT NULL REFERENCES voting.poll_options(id) ON DELETE CASCADE,  -- ID выбранного варианта
    user_telegram_id BIGINT NOT NULL,                                         -- Telegram ID проголосовавшего
    user_username TEXT,                                                       -- Username проголосовавшего (опционально)
    user_first_name TEXT,                                                     -- Имя пользователя
    user_last_name TEXT,                                                      -- Фамилия пользователя (опционально)
    voted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),                              -- Дата и время голоса
    CONSTRAINT unique_vote_per_user_option UNIQUE (poll_id, user_telegram_id)
);

-- Индексы для таблицы votes
CREATE INDEX IF NOT EXISTS idx_votes_poll_id ON voting.votes(poll_id);
CREATE INDEX IF NOT EXISTS idx_votes_option_id ON voting.votes(option_id);

-- Таблица логирования всех нажатий на кнопки (append-only)
CREATE TABLE IF NOT EXISTS voting.vote_log (
    id BIGSERIAL PRIMARY KEY,
    user_telegram_id BIGINT NOT NULL,             -- Telegram ID пользователя
    poll_id BIGINT NOT NULL,                      -- ID голосования
    option_id BIGINT NOT NULL,                    -- ID выбранного варианта
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Время нажатия на кнопку
);

-- Комментарии к таблицам
COMMENT ON TABLE voting.polls IS 'Таблица голосований';
COMMENT ON TABLE voting.poll_options IS 'Варианты ответов для голосований';
COMMENT ON TABLE voting.poll_chats IS 'Чаты и inline-сообщения, куда были опубликованы голосования';
COMMENT ON COLUMN voting.poll_chats.inline_message_id IS 'ID inline-сообщения (если голосование отправлено через inline-режим)';
COMMENT ON COLUMN voting.poll_chats.message_hash IS 'Хеш для дополнительной идентификации сообщения';
COMMENT ON TABLE voting.votes IS 'Голоса пользователей';
COMMENT ON TABLE voting.vote_log IS 'Лог всех нажатий на кнопки голосования (append-only, без индексов)';

