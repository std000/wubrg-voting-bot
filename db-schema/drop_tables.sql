-- Удаление всех таблиц (для полного пересоздания схемы)
-- ВНИМАНИЕ: Это удалит все данные!

DROP TABLE IF EXISTS voting.votes CASCADE;
DROP TABLE IF EXISTS voting.poll_chats CASCADE;
DROP TABLE IF EXISTS voting.poll_options CASCADE;
DROP TABLE IF EXISTS voting.polls CASCADE;

-- Опционально: удалить саму схему (раскомментируйте при необходимости)
-- DROP SCHEMA IF EXISTS voting CASCADE;

