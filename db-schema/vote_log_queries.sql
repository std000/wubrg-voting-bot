-- Примеры SQL-запросов для анализа таблицы vote_log

-- ===================================
-- Базовые запросы
-- ===================================

-- 1. Все нажатия на кнопки за последний час
SELECT 
    vl.id,
    vl.clicked_at,
    vl.user_telegram_id,
    p.title as poll_title,
    po.option_text
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
JOIN voting.poll_options po ON po.id = vl.option_id
WHERE vl.clicked_at > NOW() - INTERVAL '1 hour'
ORDER BY vl.clicked_at DESC;

-- 2. Количество нажатий по каждому опросу
SELECT 
    p.id,
    p.title,
    COUNT(*) as total_clicks,
    COUNT(DISTINCT vl.user_telegram_id) as unique_users
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
GROUP BY p.id, p.title
ORDER BY total_clicks DESC;

-- 3. Последние 20 действий конкретного пользователя
SELECT 
    vl.clicked_at,
    p.title,
    po.option_text
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
JOIN voting.poll_options po ON po.id = vl.option_id
WHERE vl.user_telegram_id = :user_id
ORDER BY vl.clicked_at DESC
LIMIT 20;

-- ===================================
-- Анализ активности пользователей
-- ===================================

-- 4. Пользователи с наибольшим количеством нажатий (самые активные)
SELECT 
    vl.user_telegram_id,
    COUNT(*) as total_clicks,
    COUNT(DISTINCT vl.poll_id) as polls_participated
FROM voting.vote_log vl
GROUP BY vl.user_telegram_id
ORDER BY total_clicks DESC
LIMIT 20;

-- 5. История голосований пользователя в конкретном опросе
SELECT 
    vl.clicked_at,
    po.option_text,
    LEAD(po.option_text) OVER (ORDER BY vl.clicked_at) as next_choice
FROM voting.vote_log vl
JOIN voting.poll_options po ON po.id = vl.option_id
WHERE vl.poll_id = :poll_id 
  AND vl.user_telegram_id = :user_id
ORDER BY vl.clicked_at;

-- 6. Сколько раз пользователь нажимал на кнопки в каждом опросе
SELECT 
    p.id,
    p.title,
    COUNT(*) as total_clicks,
    COUNT(DISTINCT vl.user_telegram_id) as unique_users
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
GROUP BY p.id, p.title
ORDER BY total_clicks DESC;

-- ===================================
-- Временной анализ
-- ===================================

-- 7. Активность голосований по часам
SELECT 
    DATE_TRUNC('hour', clicked_at) as hour,
    COUNT(*) as clicks_count,
    COUNT(DISTINCT user_telegram_id) as unique_users
FROM voting.vote_log
WHERE poll_id = :poll_id
GROUP BY hour
ORDER BY hour;

-- 8. Активность по дням недели
SELECT 
    TO_CHAR(clicked_at, 'Day') as day_of_week,
    EXTRACT(ISODOW FROM clicked_at) as day_number,
    COUNT(*) as total_clicks,
    COUNT(DISTINCT user_telegram_id) as unique_users
FROM voting.vote_log
GROUP BY day_of_week, day_number
ORDER BY day_number;

-- 9. Пиковые часы активности
SELECT 
    EXTRACT(HOUR FROM clicked_at) as hour,
    COUNT(*) as clicks_count,
    COUNT(DISTINCT user_telegram_id) as unique_users
FROM voting.vote_log
GROUP BY hour
ORDER BY clicks_count DESC;

-- ===================================
-- Анализ паттернов голосования
-- ===================================

-- 10. Пользователи, которые нажимали на кнопки несколько раз в одном опросе
SELECT 
    vl.poll_id,
    vl.user_telegram_id,
    COUNT(*) as clicks_count,
    MIN(vl.clicked_at) as first_click,
    MAX(vl.clicked_at) as last_click,
    MAX(vl.clicked_at) - MIN(vl.clicked_at) as time_between_first_and_last
FROM voting.vote_log vl
GROUP BY vl.poll_id, vl.user_telegram_id
HAVING COUNT(*) > 1
ORDER BY clicks_count DESC;

-- 11. Средняя скорость принятия решения (время до первого клика)
WITH poll_published AS (
    SELECT 
        poll_id,
        MIN(created_at) as published_at
    FROM voting.poll_chats
    GROUP BY poll_id
),
first_clicks AS (
    SELECT 
        poll_id,
        user_telegram_id,
        MIN(clicked_at) as first_click_at
    FROM voting.vote_log
    GROUP BY poll_id, user_telegram_id
)
SELECT 
    pp.poll_id,
    AVG(fc.first_click_at - pp.published_at) as avg_time_to_click,
    MIN(fc.first_click_at - pp.published_at) as min_time_to_click,
    MAX(fc.first_click_at - pp.published_at) as max_time_to_click
FROM poll_published pp
JOIN first_clicks fc ON fc.poll_id = pp.poll_id
WHERE fc.first_click_at > pp.published_at
GROUP BY pp.poll_id;

-- 12. Переходы между вариантами ответов
WITH numbered_choices AS (
    SELECT 
        poll_id,
        user_telegram_id,
        option_id,
        clicked_at,
        LAG(option_id) OVER (PARTITION BY poll_id, user_telegram_id ORDER BY clicked_at) as previous_option
    FROM voting.vote_log
)
SELECT 
    nc.poll_id,
    po_from.option_text as from_option,
    po_to.option_text as to_option,
    COUNT(*) as transition_count
FROM numbered_choices nc
JOIN voting.poll_options po_from ON po_from.id = nc.previous_option
JOIN voting.poll_options po_to ON po_to.id = nc.option_id
WHERE nc.previous_option IS NOT NULL
GROUP BY nc.poll_id, po_from.option_text, po_to.option_text
ORDER BY transition_count DESC;

-- ===================================
-- Системная информация
-- ===================================

-- 13. Размер таблицы vote_log
SELECT 
    pg_size_pretty(pg_total_relation_size('voting.vote_log')) as total_size,
    pg_size_pretty(pg_relation_size('voting.vote_log')) as table_size,
    (SELECT COUNT(*) FROM voting.vote_log) as row_count;

-- 14. Статистика по датам (для планирования архивации)
SELECT 
    DATE(clicked_at) as date,
    COUNT(*) as clicks_count,
    COUNT(DISTINCT user_telegram_id) as unique_users,
    COUNT(DISTINCT poll_id) as active_polls
FROM voting.vote_log
GROUP BY date
ORDER BY date DESC
LIMIT 30;

-- 15. Самые активные опросы за последнюю неделю
SELECT 
    p.id,
    p.title,
    p.created_at,
    COUNT(*) as total_clicks,
    COUNT(DISTINCT vl.user_telegram_id) as unique_voters,
    MAX(vl.clicked_at) as last_activity
FROM voting.vote_log vl
JOIN voting.polls p ON p.id = vl.poll_id
WHERE vl.clicked_at > NOW() - INTERVAL '7 days'
GROUP BY p.id, p.title, p.created_at
ORDER BY total_clicks DESC
LIMIT 10;

