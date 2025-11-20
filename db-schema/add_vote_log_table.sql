-- Таблица для логирования всех нажатий на кнопки голосования (append-only)
-- Эта таблица записывает ВСЕ нажатия пользователей, включая случаи когда они переголосовывают

CREATE TABLE IF NOT EXISTS voting.vote_log (
    id BIGSERIAL PRIMARY KEY,
    user_telegram_id BIGINT NOT NULL,             -- Telegram ID пользователя
    poll_id BIGINT NOT NULL,                      -- ID голосования
    option_id BIGINT NOT NULL,                    -- ID выбранного варианта
    clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW() -- Время нажатия на кнопку
);

COMMENT ON TABLE voting.vote_log IS 'Лог всех нажатий на кнопки голосования (append-only, без индексов)';

