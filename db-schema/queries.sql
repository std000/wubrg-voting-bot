-- Полезные SQL запросы для работы с голосованиями

-- ================================================
-- БЫСТРАЯ ПРОВЕРКА
-- ================================================

-- Быстро посмотреть последние созданные голосования с их вариантами
SELECT 
    p.id,
    p.title,
    p.creator_telegram_id,
    p.creator_username,
    p.created_at,
    po.option_text,
    po.emoji
FROM voting.polls p
LEFT JOIN voting.poll_options po ON p.id = po.poll_id
ORDER BY p.id DESC, po.id ASC
LIMIT 20;

-- Посмотреть все данные последнего голосования
SELECT * FROM voting.polls ORDER BY id DESC LIMIT 1;

-- Посмотреть варианты последнего голосования
SELECT po.* 
FROM voting.poll_options po 
JOIN voting.polls p ON po.poll_id = p.id 
ORDER BY p.id DESC, po.id ASC 
LIMIT 10;

-- ================================================
-- ПОЛУЧЕНИЕ ДАННЫХ О ГОЛОСОВАНИЯХ
-- ================================================

-- Получить все активные голосования
SELECT 
    p.id,
    p.title,
    p.description,
    p.creator_telegram_id,
    p.created_at,
    COUNT(DISTINCT v.user_telegram_id) as total_voters,
    COUNT(v.id) as total_votes
FROM voting.polls p
LEFT JOIN voting.votes v ON p.id = v.poll_id
WHERE p.is_active = true
GROUP BY p.id
ORDER BY p.created_at DESC;

-- Получить детальную информацию о конкретном голосовании
SELECT 
    p.*,
    COUNT(DISTINCT po.id) as options_count,
    COUNT(DISTINCT v.user_telegram_id) as total_voters,
    COUNT(v.id) as total_votes
FROM voting.polls p
LEFT JOIN voting.poll_options po ON p.id = po.poll_id
LEFT JOIN voting.votes v ON p.id = v.poll_id
WHERE p.id = 1
GROUP BY p.id;

-- ================================================
-- РЕЗУЛЬТАТЫ ГОЛОСОВАНИЯ
-- ================================================

-- Получить результаты голосования с процентами
SELECT 
    po.id,
    po.option_text,
    po.emoji,
    COUNT(v.id) as vote_count,
    ROUND(
        CASE 
            WHEN (SELECT COUNT(*) FROM voting.votes WHERE poll_id = po.poll_id) > 0 
            THEN COUNT(v.id) * 100.0 / (SELECT COUNT(*) FROM voting.votes WHERE poll_id = po.poll_id)
            ELSE 0 
        END, 
        2
    ) as percentage
FROM voting.poll_options po
LEFT JOIN voting.votes v ON po.id = v.option_id
WHERE po.poll_id = 1
GROUP BY po.id, po.option_text, po.emoji, po.poll_id
ORDER BY po.id;

-- Получить топ-3 варианта ответа
SELECT 
    po.option_text,
    po.emoji,
    COUNT(v.id) as vote_count
FROM voting.poll_options po
LEFT JOIN voting.votes v ON po.id = v.option_id
WHERE po.poll_id = 1
GROUP BY po.id, po.option_text, po.emoji
ORDER BY vote_count DESC
LIMIT 3;

-- ================================================
-- ИНФОРМАЦИЯ О ГОЛОСОВАВШИХ
-- ================================================

-- Получить список всех проголосовавших
SELECT 
    v.user_first_name || COALESCE(' ' || v.user_last_name, '') as full_name,
    v.user_username,
    po.option_text,
    v.voted_at
FROM voting.votes v
JOIN voting.poll_options po ON v.option_id = po.id
WHERE v.poll_id = 1
ORDER BY v.voted_at DESC;

-- Проверить, голосовал ли конкретный пользователь
SELECT EXISTS(
    SELECT 1 FROM voting.votes 
    WHERE poll_id = 1 
      AND user_telegram_id = 123456789
) as has_voted;

-- Получить выбор конкретного пользователя
SELECT 
    po.option_text,
    v.voted_at
FROM voting.votes v
JOIN voting.poll_options po ON v.option_id = po.id
WHERE v.poll_id = 1 
  AND v.user_telegram_id = 123456789
ORDER BY v.voted_at;

-- ================================================
-- СТАТИСТИКА ПО СОЗДАТЕЛЯМ
-- ================================================

-- Получить статистику по голосованиям для каждого создателя
SELECT 
    p.creator_telegram_id,
    p.creator_username,
    COUNT(DISTINCT p.id) as total_polls,
    COUNT(DISTINCT CASE WHEN p.is_active THEN p.id END) as active_polls,
    SUM(vote_counts.vote_count) as total_votes_received
FROM voting.polls p
LEFT JOIN (
    SELECT poll_id, COUNT(*) as vote_count
    FROM voting.votes
    GROUP BY poll_id
) vote_counts ON p.id = vote_counts.poll_id
GROUP BY p.creator_telegram_id, p.creator_username
ORDER BY total_polls DESC;

-- ================================================
-- ИНФОРМАЦИЯ О ЧАТАХ
-- ================================================

-- Получить все чаты, где было опубликовано голосование
SELECT 
    pc.chat_id,
    pc.message_id,
    p.title as poll_title
FROM voting.poll_chats pc
JOIN voting.polls p ON pc.poll_id = p.id
WHERE pc.poll_id = 1
ORDER BY pc.id DESC;

-- Статистика по чатам
SELECT 
    pc.chat_id,
    COUNT(DISTINCT pc.poll_id) as polls_count
FROM voting.poll_chats pc
GROUP BY pc.chat_id
ORDER BY polls_count DESC;

-- ================================================
-- УПРАВЛЕНИЕ ГОЛОСОВАНИЯМИ
-- ================================================

-- Закрыть голосование (сделать неактивным)
-- UPDATE voting.polls SET is_active = false, updated_at = CURRENT_TIMESTAMP WHERE id = 1;

-- Удалить голос пользователя
-- DELETE FROM voting.votes WHERE poll_id = 1 AND user_telegram_id = 123456789;

-- Удалить голосование полностью (каскадно удалит все связанные данные)
-- DELETE FROM voting.polls WHERE id = 1;

-- ================================================
-- ОЧИСТКА СТАРЫХ ДАННЫХ
-- ================================================

-- Удалить неактивные голосования старше 30 дней
-- DELETE FROM voting.polls WHERE is_active = false AND created_at < NOW() - INTERVAL '30 days';

-- Закрыть истекшие голосования
-- UPDATE voting.polls SET is_active = false, updated_at = CURRENT_TIMESTAMP 
-- WHERE is_active = true AND expires_at < CURRENT_TIMESTAMP;

-- ================================================
-- ПОПУЛЯРНЫЕ ГОЛОСОВАНИЯ
-- ================================================

-- Топ-10 самых популярных голосований по количеству голосов
SELECT 
    p.id,
    p.title,
    p.creator_username,
    COUNT(DISTINCT v.user_telegram_id) as unique_voters,
    COUNT(v.id) as total_votes,
    p.created_at
FROM voting.polls p
LEFT JOIN voting.votes v ON p.id = v.poll_id
GROUP BY p.id
ORDER BY unique_voters DESC, total_votes DESC
LIMIT 10;

-- Самые активные пользователи (кто больше всего голосовал)
SELECT 
    v.user_telegram_id,
    v.user_username,
    v.user_first_name,
    COUNT(DISTINCT v.poll_id) as polls_participated,
    COUNT(v.id) as total_votes_cast
FROM voting.votes v
GROUP BY v.user_telegram_id, v.user_username, v.user_first_name
ORDER BY polls_participated DESC, total_votes_cast DESC
LIMIT 10;

