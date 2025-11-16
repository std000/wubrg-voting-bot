-- Миграция для добавления колонки emoji в таблицу poll_options

-- Добавляем колонку emoji (nullable, без значения по умолчанию)
ALTER TABLE voting.poll_options 
ADD COLUMN IF NOT EXISTS emoji TEXT;

-- Добавляем комментарий к колонке
COMMENT ON COLUMN voting.poll_options.emoji IS 'Эмодзи для визуализации голосов за этот вариант (nullable, по умолчанию 👍 в коде)';

