package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"wubrg-voting-bot/bot"

	"github.com/jackc/pgx/v5/pgxpool"
)

var Version = "dev"

func main() {
	// Проверка флага --version
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Println(Version)
		return
	}

	fmt.Printf("🚀 WUBRG Voting Bot версия %s\n", Version)
	// Строка подключения к PostgreSQL
	// Формат: postgresql://username:password@localhost:5432/database_name
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		databaseURL = "postgresql://postgres:postgres@localhost:5432/wubrg_voting"
	}

	// Создание пула соединений к базе данных
	ctx := context.Background()
	dbpool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("Не удалось создать пул соединений к базе данных: %v", err)
	}
	defer dbpool.Close()

	// Проверка соединения к БД
	var greeting string
	err = dbpool.QueryRow(ctx, "SELECT 'Hello from PostgreSQL!'").Scan(&greeting)
	if err != nil {
		log.Fatalf("Ошибка выполнения запроса: %v", err)
	}
	fmt.Println(greeting)
	fmt.Printf("✅ Успешное подключение к PostgreSQL через pgxpool! (макс. соединений: %d)\n", dbpool.Config().MaxConns)

	// Получение токена бота из переменной окружения
	botToken := os.Getenv("BOT_TOKEN")
	if botToken == "" {
		log.Fatal("❌ Не указан токен бота. Установите переменную окружения BOT_TOKEN")
	}

	// Создание и запуск бота
	tgBot, err := bot.New(botToken, dbpool)
	if err != nil {
		log.Fatalf("Не удалось создать бота: %v", err)
	}

	fmt.Println("✅ Telegram бот успешно запущен!")

	// Запуск бота
	tgBot.Start()
}
