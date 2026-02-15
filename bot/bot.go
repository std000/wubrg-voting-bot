package bot

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"gopkg.in/telebot.v4"
)

type Bot struct {
	bot         *telebot.Bot
	db          *pgxpool.Pool
	dialog      *DialogManager
	updateQueue *UpdateQueue
}

// New создает и настраивает новый экземпляр бота
func New(token string, db *pgxpool.Pool) (*Bot, error) {
	pref := telebot.Settings{
		Token:  token,
		Poller: &telebot.LongPoller{Timeout: 10 * time.Second},
	}

	tgBot, err := telebot.NewBot(pref)
	if err != nil {
		return nil, fmt.Errorf("не удалось создать бота: %w", err)
	}

	b := &Bot{
		bot:         tgBot,
		db:          db,
		dialog:      NewDialogManager(),
		updateQueue: NewUpdateQueue(),
	}

	// Регистрация обработчиков
	b.registerHandlers()

	return b, nil
}

// registerHandlers регистрирует все обработчики команд и сообщений
func (b *Bot) registerHandlers() {
	// Обработчик команды /start
	b.bot.Handle("/start", b.handleStart)

	// Обработчик команды /help
	b.bot.Handle("/help", b.handleHelp)

	// Обработчик команды /status
	b.bot.Handle("/status", b.handleStatus)

	// Обработчик команды /cancel - отмена диалога
	b.bot.Handle("/cancel", b.handleCancel)

	// Обработчик команды /createpoll - создать голосование
	b.bot.Handle("/createpoll", b.handleCreatePoll)

	// Обработчик команды /listpolls - показать список голосований
	b.bot.Handle("/listpolls", b.handleListPolls)

	// Обработчик команды /publishpoll - опубликовать голосование
	b.bot.Handle("/publishpoll", b.handlePublishPoll)

	// Обработчик callback-кнопок (роутер)
	b.bot.Handle(telebot.OnCallback, b.handleCallback)

	// Обработчик inline-запросов
	b.bot.Handle(telebot.OnQuery, b.handleInlineQuery)

	// Обработчик выбора inline-результата (когда пользователь отправляет голосование в чат)
	b.bot.Handle(telebot.OnInlineResult, b.handleChosenInlineResult)

	// Обработчик текстовых сообщений (с учетом состояния диалога)
	b.bot.Handle(telebot.OnText, b.handleText)
}

// handleStart обрабатывает команду /start
func (b *Bot) handleStart(c telebot.Context) error {
	// Проверяем deep-link параметр (например, /start createpoll)
	payload := c.Message().Payload
	if payload == "createpoll" {
		return b.handleCreatePoll(c)
	}

	return c.Send("👋 Привет! Я бот для голосования WUBRG.\n\nИспользуй /help чтобы узнать доступные команды.")
}

// handleHelp обрабатывает команду /help
func (b *Bot) handleHelp(c telebot.Context) error {
	helpText := `📋 Доступные команды:

/start - Начать работу с ботом
/help - Показать это сообщение
/status - Проверить статус подключения к БД

🗳 Голосования:
/createpoll - Создать новое голосование
/listpolls - Показать список голосований
/publishpoll <ID> - Опубликовать голосование

📲 Inline-режим:
Используйте @bot_name в любом чате, чтобы:
• Найти и опубликовать голосование
• Поиск по названию голосования

/cancel - Отменить текущий диалог`
	return c.Send(helpText)
}

// handleStatus обрабатывает команду /status
func (b *Bot) handleStatus(c telebot.Context) error {
	ctx := context.Background()
	var result string
	err := b.db.QueryRow(ctx, "SELECT version()").Scan(&result)
	if err != nil {
		return c.Send("❌ Ошибка подключения к базе данных")
	}
	return c.Send("✅ База данных подключена и работает!")
}

// handleCallback роутер для callback-кнопок
func (b *Bot) handleCallback(c telebot.Context) error {
	data := c.Data()
	switch {
	case strings.HasPrefix(data, "\fvote|"):
		return b.handleVote(c)
	case strings.HasPrefix(data, "\fpoll_done"):
		return b.handlePollDoneCallback(c)
	case strings.HasPrefix(data, "\fpoll_confirm_yes"):
		return b.handlePollConfirmYesCallback(c)
	case strings.HasPrefix(data, "\fpoll_confirm_no"):
		return b.handlePollConfirmNoCallback(c)
	default:
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Неизвестная команда"})
	}
}

// handleText обрабатывает текстовые сообщения с учетом состояния диалога
func (b *Bot) handleText(c telebot.Context) error {
	userID := c.Sender().ID
	ctx := b.dialog.GetContext(userID)

	// Обработка в зависимости от состояния
	switch ctx.State {
	case StateCreatePollTitle:
		return b.handlePollTitleInput(c)
	case StateCreatePollOption:
		return b.handlePollOptionInput(c)
	default:
		// Обычный режим без диалога
		return c.Send(fmt.Sprintf("Вы написали: %s\n\nИспользуйте /help для списка команд", c.Text()))
	}
}

// handleCancel отменяет текущий диалог
func (b *Bot) handleCancel(c telebot.Context) error {
	userID := c.Sender().ID
	ctx := b.dialog.GetContext(userID)

	if ctx.State == StateIdle {
		return c.Send("❌ Нет активного диалога для отмены.")
	}

	b.dialog.ResetContext(userID)
	return c.Send("✅ Диалог отменен. Вы вернулись в обычный режим.")
}

// startUpdateWorker запускает горутину-воркер для обработки очереди обновлений
func (b *Bot) startUpdateWorker() {
	go func() {
		log.Println("📨 [UpdateWorker] Воркер обновления сообщений запущен")
		for range b.updateQueue.notify {
			polls := b.updateQueue.drain()
			for _, pollID := range polls {
				b.updatePollMessages(pollID)
			}
		}
	}()
}

// Start запускает бота
func (b *Bot) Start() {
	log.Println("🤖 Бот начал прослушивание сообщений...")
	b.startUpdateWorker()
	b.bot.Start()
}
