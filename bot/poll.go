package bot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"gopkg.in/telebot.v4"
)

const SameMessageError = "telegram: Bad Request: message is not modified: specified new message content and reply markup are exactly the same as a current content and reply markup of the message (400)"

// handleCreatePoll запускает диалог создания голосования
func (b *Bot) handleCreatePoll(c telebot.Context) error {
	userID := c.Sender().ID
	b.dialog.ResetContext(userID)
	b.dialog.SetState(userID, StateCreatePollTitle)
	b.dialog.SetData(userID, "poll_options", []string{}) // Инициализируем список вариантов
	return c.Send("📊 Создание нового голосования\n\n📝 Шаг 1: Введите заголовок голосования:")
}

// handlePollTitleInput обрабатывает ввод заголовка голосования
func (b *Bot) handlePollTitleInput(c telebot.Context) error {
	userID := c.Sender().ID
	title := c.Text()

	if len(title) < 3 {
		return c.Send("❌ Заголовок слишком короткий (минимум 3 символа). Попробуйте еще раз:")
	}

	if len(title) > 200 {
		return c.Send("❌ Заголовок слишком длинный (максимум 200 символов). Попробуйте еще раз:")
	}

	b.dialog.SetData(userID, "poll_title", title)
	b.dialog.SetState(userID, StateCreatePollOption)

	return c.Send(fmt.Sprintf("✅ Заголовок сохранен: \"%s\"\n\n"+
		"📝 Шаг 2: Добавьте варианты ответа\n\n"+
		"Введите первый вариант ответа:", title))
}

// optionInputMarkup возвращает inline-клавиатуру с кнопкой "Готово"
func optionInputMarkup() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	btnDone := markup.Data("✅ Готово", "poll_done")
	markup.Inline(markup.Row(btnDone))
	return markup
}

// confirmPollMarkup возвращает inline-клавиатуру с кнопками "Подтвердить" и "Отменить"
func confirmPollMarkup() *telebot.ReplyMarkup {
	markup := &telebot.ReplyMarkup{}
	btnYes := markup.Data("✅ Подтвердить", "poll_confirm_yes")
	btnNo := markup.Data("❌ Отменить", "poll_confirm_no")
	markup.Inline(markup.Row(btnYes, btnNo))
	return markup
}

// handlePollConfirmYesCallback обрабатывает нажатие кнопки "Подтвердить" при подтверждении
func (b *Bot) handlePollConfirmYesCallback(c telebot.Context) error {
	userID := c.Sender().ID
	dialogCtx := b.dialog.GetContext(userID)

	if dialogCtx.State != StateCreatePollConfirm {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Нет активного создания голосования"})
	}

	// Получаем данные голосования
	titleInterface, _ := b.dialog.GetData(userID, "poll_title")
	optionsInterface, _ := b.dialog.GetData(userID, "poll_options")

	title := titleInterface.(string)
	options := optionsInterface.([]string)

	// Получаем username создателя
	username := c.Sender().Username

	// Сохраняем голосование в БД
	ctx := context.Background()
	pollID, err := b.savePollToDB(ctx, userID, username, title, options)
	if err != nil {
		log.Printf("❌ Ошибка сохранения голосования: %v", err)
		c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка сохранения"})
		return c.Send(fmt.Sprintf("❌ Ошибка при сохранении голосования: %v\n\nПопробуйте еще раз позже.", err))
	}

	log.Printf("✅ Пользователь %d создал голосование ID=%d: %s с %d вариантами", userID, pollID, title, len(options))

	// Формируем сообщение об успехе
	successMsg := "🎉 Голосование успешно создано!\n\n"
	successMsg += fmt.Sprintf("📝 %s\n\n", title)
	for i, option := range options {
		successMsg += fmt.Sprintf("%d. %s\n", i+1, option)
	}
	successMsg += fmt.Sprintf("\n✅ Голосование сохранено в базу данных!\n🆔 ID голосования: %d\n\n", pollID)
	successMsg += "Используйте /publishpoll " + strconv.FormatInt(pollID, 10) + " чтобы опубликовать голосование в этом чате."

	b.dialog.SetState(userID, StateIdle)
	c.Respond(&telebot.CallbackResponse{Text: "✅ Голосование создано!"})
	return c.Send(successMsg)
}

// handlePollConfirmNoCallback обрабатывает нажатие кнопки "Нет" при подтверждении
func (b *Bot) handlePollConfirmNoCallback(c telebot.Context) error {
	userID := c.Sender().ID
	dialogCtx := b.dialog.GetContext(userID)

	if dialogCtx.State != StateCreatePollConfirm {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Нет активного создания голосования"})
	}

	b.dialog.ResetContext(userID)
	c.Respond(&telebot.CallbackResponse{Text: "❌ Отменено"})
	return c.Send("❌ Создание голосования отменено.\n\nИспользуйте /createpoll чтобы начать заново.")
}

// handlePollDoneCallback обрабатывает нажатие кнопки "Готово" при добавлении вариантов
func (b *Bot) handlePollDoneCallback(c telebot.Context) error {
	userID := c.Sender().ID
	dialogCtx := b.dialog.GetContext(userID)

	// Проверяем, что пользователь в нужном состоянии
	if dialogCtx.State != StateCreatePollOption {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Нет активного создания голосования"})
	}

	// Проверяем количество вариантов
	optionsInterface, _ := b.dialog.GetData(userID, "poll_options")
	options := optionsInterface.([]string)

	if len(options) < 2 {
		return c.Respond(&telebot.CallbackResponse{
			Text:      fmt.Sprintf("❌ Нужно минимум 2 варианта. Сейчас: %d", len(options)),
			ShowAlert: true,
		})
	}

	// Переходим к подтверждению
	b.dialog.SetState(userID, StateCreatePollConfirm)
	c.Respond(&telebot.CallbackResponse{})
	return b.showPollPreview(c)
}

// handlePollOptionInput обрабатывает ввод вариантов голосования
func (b *Bot) handlePollOptionInput(c telebot.Context) error {
	userID := c.Sender().ID
	option := c.Text()

	// Валидация варианта
	if len(option) < 1 {
		return c.Send("❌ Вариант не может быть пустым. Попробуйте еще раз:")
	}

	if len(option) > 100 {
		return c.Send("❌ Вариант слишком длинный (максимум 100 символов). Попробуйте еще раз:")
	}

	// Добавляем вариант
	optionsInterface, _ := b.dialog.GetData(userID, "poll_options")
	options := optionsInterface.([]string)
	options = append(options, option)
	b.dialog.SetData(userID, "poll_options", options)

	optionNumber := len(options)

	return c.Send(fmt.Sprintf("✅ Вариант %d добавлен: \"%s\"\n\n"+
		"Всего вариантов: %d\n\n"+
		"Введите следующий вариант или нажмите «Готово» для завершения:",
		optionNumber, option, optionNumber), optionInputMarkup())
}

// showPollPreview показывает превью голосования перед созданием
func (b *Bot) showPollPreview(c telebot.Context) error {
	userID := c.Sender().ID

	titleInterface, _ := b.dialog.GetData(userID, "poll_title")
	optionsInterface, _ := b.dialog.GetData(userID, "poll_options")

	title := titleInterface.(string)
	options := optionsInterface.([]string)

	preview := fmt.Sprintf("📊 Превью голосования:\n\n"+
		"━━━━━━━━━━━━━━━━━━━━\n"+
		"📝 %s\n"+
		"━━━━━━━━━━━━━━━━━━━━\n\n", title)

	for i, option := range options {
		preview += fmt.Sprintf("%d. %s\n", i+1, option)
	}

	preview += "\n━━━━━━━━━━━━━━━━━━━━\n\n" +
		"Все верно?"

	return c.Send(preview, confirmPollMarkup())
}

// savePollToDB сохраняет голосование в базу данных
func (b *Bot) savePollToDB(ctx context.Context, creatorID int64, creatorUsername string, title string, options []string) (int64, error) {
	// Начинаем транзакцию
	tx, err := b.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("ошибка начала транзакции: %w", err)
	}
	defer tx.Rollback(ctx)

	// Вставляем голосование
	var pollID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO voting.polls (title, creator_telegram_id, creator_username, is_active, created_at, updated_at)
		 VALUES ($1, $2, $3, true, NOW(), NOW())
		 RETURNING id`,
		title, creatorID, creatorUsername,
	).Scan(&pollID)
	if err != nil {
		return 0, fmt.Errorf("ошибка создания голосования: %w", err)
	}

	// Вставляем варианты ответов
	for _, option := range options {
		_, err = tx.Exec(ctx,
			`INSERT INTO voting.poll_options (poll_id, option_text, created_at)
			 VALUES ($1, $2, NOW())`,
			pollID, option,
		)
		if err != nil {
			return 0, fmt.Errorf("ошибка добавления варианта '%s': %w", option, err)
		}
	}

	// Коммитим транзакцию
	if err = tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("ошибка фиксации транзакции: %w", err)
	}

	return pollID, nil
}

// PollOption представляет вариант ответа в голосовании
type PollOption struct {
	ID    int64
	Text  string
	Emoji string
	Votes []Vote
}

// Vote представляет один голос
type Vote struct {
	UserID    int64
	Username  string
	FirstName string
	LastName  string
}

// PollData представляет данные голосования
type PollData struct {
	ID         int64
	Title      string
	Options    []PollOption
	TotalVotes int
}

// handleListPolls показывает список активных голосований пользователя
func (b *Bot) handleListPolls(c telebot.Context) error {
	ctx := context.Background()
	userID := c.Sender().ID

	rows, err := b.db.Query(ctx,
		`SELECT id, title, created_at 
		 FROM voting.polls 
		 WHERE is_active = true AND creator_telegram_id = $1
		 ORDER BY created_at DESC 
		 LIMIT 10`,
		userID)
	if err != nil {
		log.Printf("❌ Ошибка получения списка голосований: %v", err)
		return c.Send("❌ Ошибка получения списка голосований")
	}
	defer rows.Close()

	var polls []struct {
		ID        int64
		Title     string
		CreatedAt time.Time
	}

	for rows.Next() {
		var poll struct {
			ID        int64
			Title     string
			CreatedAt time.Time
		}
		if err := rows.Scan(&poll.ID, &poll.Title, &poll.CreatedAt); err != nil {
			log.Printf("❌ Ошибка чтения данных голосования: %v", err)
			continue
		}
		polls = append(polls, poll)
	}

	if len(polls) == 0 {
		return c.Send("📊 У вас нет активных голосований.\n\nИспользуйте /createpoll чтобы создать новое.")
	}

	msg := "📊 Ваши активные голосования:\n\n"
	for i, poll := range polls {
		msg += fmt.Sprintf("%d. %s\n   🆔 ID: %d | 📅 %s\n\n",
			i+1, poll.Title, poll.ID, poll.CreatedAt.Format("02.01.2006 15:04"))
	}
	msg += "Используйте /publishpoll <ID> чтобы опубликовать голосование"

	return c.Send(msg)
}

// getPollData получает данные голосования из БД одним запросом с JOIN
func (b *Bot) getPollData(ctx context.Context, pollID int64) (*PollData, error) {
	// Получаем всё одним запросом с JOIN
	rows, err := b.db.Query(ctx,
		`SELECT 
		     p.id, p.title,
		     po.id as option_id, po.option_text, po.emoji,
		     v.user_telegram_id, v.user_username, v.user_first_name, v.user_last_name
		 FROM voting.polls p
		 LEFT JOIN voting.poll_options po ON po.poll_id = p.id
		 LEFT JOIN voting.votes v ON v.poll_id = p.id AND v.option_id = po.id
		 WHERE p.id = $1 AND p.is_active = true
		 ORDER BY po.id, v.voted_at`,
		pollID)
	if err != nil {
		return nil, fmt.Errorf("ошибка получения данных голосования: %w", err)
	}
	defer rows.Close()

	var poll *PollData
	optionMap := make(map[int64]*PollOption)

	for rows.Next() {
		var pollIDResult int64
		var title string
		var optionID *int64
		var optionText *string
		var emoji *string
		var voteUserID *int64
		var voteUsername *string
		var voteFirstName *string
		var voteLastName *string

		if err := rows.Scan(&pollIDResult, &title,
			&optionID, &optionText, &emoji,
			&voteUserID, &voteUsername, &voteFirstName, &voteLastName); err != nil {
			return nil, err
		}

		// Инициализируем poll один раз
		if poll == nil {
			poll = &PollData{
				ID:      pollIDResult,
				Title:   title,
				Options: make([]PollOption, 0),
			}
		}

		// Добавляем вариант, если есть
		if optionID != nil && optionText != nil {
			option, exists := optionMap[*optionID]
			if !exists {
				// Создаём новый вариант
				emojiValue := "👍"
				if emoji != nil && *emoji != "" {
					emojiValue = *emoji
				}
				newOption := PollOption{
					ID:    *optionID,
					Text:  *optionText,
					Emoji: emojiValue,
					Votes: make([]Vote, 0),
				}
				poll.Options = append(poll.Options, newOption)
				// Сохраняем указатель на последний добавленный вариант
				optionMap[*optionID] = &poll.Options[len(poll.Options)-1]
				option = optionMap[*optionID]
			}

			// Добавляем голос, если есть
			if voteUserID != nil {
				vote := Vote{
					UserID: *voteUserID,
				}
				if voteUsername != nil {
					vote.Username = *voteUsername
				}
				if voteFirstName != nil {
					vote.FirstName = *voteFirstName
				}
				if voteLastName != nil {
					vote.LastName = *voteLastName
				}
				option.Votes = append(option.Votes, vote)
				poll.TotalVotes++
			}
		}
	}

	if poll == nil {
		return nil, fmt.Errorf("голосование не найдено")
	}

	return poll, nil
}

// formatPollMessage форматирует голосование в красивый текст
func formatPollMessage(poll *PollData) string {
	// Получаем текущую дату
	msg := fmt.Sprintf(poll.Title)

	for _, opt := range poll.Options {
		voteCount := len(opt.Votes)
		percentage := 0
		if poll.TotalVotes > 0 {
			percentage = (voteCount * 100) / poll.TotalVotes
		}

		// Вычисляем количество эмодзи (примерно 1 эмодзи на 6-7%)
		thumbsCount := (percentage + 6) / 7
		if thumbsCount > 14 {
			thumbsCount = 14
		}
		thumbs := strings.Repeat(opt.Emoji, thumbsCount)
		if thumbs == "" && voteCount > 0 {
			thumbs = opt.Emoji
		}

		msg += fmt.Sprintf("\n%s – %d\n", opt.Text, voteCount)

		if voteCount > 0 {
			msg += fmt.Sprintf("%s %d%%\n", thumbs, percentage)

			// Список пользователей
			usernames := make([]string, 0)
			for _, vote := range opt.Votes {
				if vote.Username != "" {
					usernames = append(usernames, "@"+vote.Username)
				} else if vote.FirstName != "" {
					usernames = append(usernames, vote.FirstName)
				}
			}
			if len(usernames) > 0 {
				msg += strings.Join(usernames, ", ")
			}
			msg += "\n"
		} else {
			msg += fmt.Sprintf("▫️ %d%%\n", percentage)
		}
	}

	msg += fmt.Sprintf("\n\n👥 %d people voted so far.", poll.TotalVotes)

	return msg
}

// handlePublishPoll публикует голосование в чат
func (b *Bot) handlePublishPoll(c telebot.Context) error {
	// Парсим ID голосования из команды
	args := strings.Fields(c.Text())
	if len(args) < 2 {
		return c.Send("❌ Укажите ID голосования.\n\nИспользование: /publishpoll <ID>\n\nПосмотрите список голосований: /listpolls")
	}

	pollID, err := strconv.ParseInt(args[1], 10, 64)
	if err != nil {
		return c.Send("❌ Некорректный ID голосования")
	}

	ctx := context.Background()
	userID := c.Sender().ID

	// Проверяем, что пользователь является владельцем голосования
	var creatorID int64
	err = b.db.QueryRow(ctx,
		`SELECT creator_telegram_id FROM voting.polls WHERE id = $1 AND is_active = true`,
		pollID).Scan(&creatorID)
	if err != nil {
		log.Printf("❌ Ошибка проверки владельца голосования: %v", err)
		return c.Send("❌ Голосование не найдено или не активно")
	}

	if creatorID != userID {
		log.Printf("⚠️ Пользователь %d попытался опубликовать чужое голосование %d (владелец: %d)", userID, pollID, creatorID)
		return c.Send("❌ Вы можете публиковать только свои голосования.\n\nПосмотрите список своих голосований: /listpolls")
	}

	poll, err := b.getPollData(ctx, pollID)
	if err != nil {
		log.Printf("❌ Ошибка получения данных голосования: %v", err)
		return c.Send(fmt.Sprintf("❌ Ошибка: %v", err))
	}

	// Создаем inline-кнопки для голосования
	markup := &telebot.ReplyMarkup{}
	rows := make([]telebot.Row, 0)

	for _, opt := range poll.Options {
		btn := markup.Data(opt.Text, "vote", strconv.FormatInt(pollID, 10), strconv.FormatInt(opt.ID, 10))
		rows = append(rows, markup.Row(btn))
	}
	markup.Inline(rows...)

	// Отправляем голосование
	msg := formatPollMessage(poll)
	sentMsg, err := c.Bot().Send(c.Chat(), msg, markup)
	if err != nil {
		log.Printf("❌ Ошибка отправки голосования: %v", err)
		return c.Send("❌ Ошибка отправки голосования")
	}

	// Сохраняем информацию о публикации в БД
	_, err = b.db.Exec(ctx,
		`INSERT INTO voting.poll_chats (poll_id, chat_id, message_id) 
		 VALUES ($1, $2, $3)
		 ON CONFLICT (poll_id, chat_id, message_id) DO NOTHING`,
		pollID, c.Chat().ID, sentMsg.ID)
	if err != nil {
		log.Printf("❌ Ошибка сохранения информации о публикации: %v", err)
	}

	log.Printf("✅ Голосование %d опубликовано в чат %d", pollID, c.Chat().ID)
	return nil
}

// handleVote обрабатывает голосование пользователя
func (b *Bot) handleVote(c telebot.Context) error {
	data := c.Data() // формат: "pollID|optionID"
	if !strings.HasPrefix(data, "\fvote|") {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка данных"})
	}
	data = strings.TrimPrefix(data, "\fvote|")
	parts := strings.Split(data, "|")
	if len(parts) != 2 {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка данных"})
	}

	pollID, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка данных голосования"})
	}

	optionID, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil {
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка данных варианта"})
	}

	user := c.Sender()
	ctx := context.Background()

	// Начинаем транзакцию
	tx, err := b.db.Begin(ctx)
	if err != nil {
		log.Printf("❌ Ошибка начала транзакции: %v", err)
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка обработки голоса"})
	}
	defer tx.Rollback(ctx)

	// Логируем нажатие на кнопку в vote_log (append-only) - в самом начале транзакции
	_, err = tx.Exec(ctx,
		`INSERT INTO voting.vote_log (user_telegram_id, poll_id, option_id)
		 VALUES ($1, $2, $3)`,
		user.ID, pollID, optionID)
	if err != nil {
		log.Printf("❌ Ошибка записи в vote_log: %v", err)
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка логирования"})
	}

	// Сохраняем или обновляем голос
	_, err = tx.Exec(ctx,
		`INSERT INTO voting.votes (poll_id, option_id, user_telegram_id, user_username, user_first_name, user_last_name)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (poll_id, user_telegram_id) 
		 DO UPDATE SET option_id = EXCLUDED.option_id, voted_at = NOW()
		 WHERE votes.option_id != EXCLUDED.option_id`,
		pollID, optionID, user.ID, user.Username, user.FirstName, user.LastName)
	if err != nil {
		log.Printf("❌ Ошибка сохранения голоса: %v", err)
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка сохранения голоса"})
	}

	// Фиксируем транзакцию
	if err = tx.Commit(ctx); err != nil {
		log.Printf("❌ Ошибка фиксации транзакции: %v", err)
		return c.Respond(&telebot.CallbackResponse{Text: "❌ Ошибка обработки голоса"})
	}

	// Планируем обновление всех сообщений этого голосования через очередь
	b.updateQueue.Schedule(pollID)

	return c.Respond(&telebot.CallbackResponse{Text: "✅ Ваш голос учтен!"})
}

// handleInlineQuery обрабатывает inline-запросы (@bot_name)
func (b *Bot) handleInlineQuery(c telebot.Context) error {
	query := c.Query()

	// Кнопка "Создать голосование" — всегда отображается над результатами
	createPollButton := &telebot.QueryResponseButton{
		Text:  "📊 Создать голосование",
		Start: "createpoll",
	}

	if query.Text != "vote" {
		return c.Answer(&telebot.QueryResponse{
			Results:   telebot.Results{},
			CacheTime: 10,
			Button:    createPollButton,
		})
	}

	ctx := context.Background()

	// Получаем ID текущего пользователя
	userID := c.Sender().ID

	// Получаем активные голосования с вариантами и голосами одним запросом (избегаем N+1)
	rows, err := b.db.Query(ctx,
		`WITH recent_polls AS (
		     SELECT id, title, created_at
		     FROM voting.polls
		     WHERE is_active = true 
		       AND creator_telegram_id = $1
		     ORDER BY created_at DESC
		     LIMIT 5
		 )
		 SELECT 
		     p.id, p.title, p.created_at,
		     po.id as option_id, po.option_text, po.emoji,
		     v.user_telegram_id, v.user_username, v.user_first_name, v.user_last_name
		 FROM recent_polls p
		 LEFT JOIN voting.poll_options po ON po.poll_id = p.id
		 LEFT JOIN voting.votes v ON v.option_id = po.id AND v.poll_id = p.id
		 ORDER BY p.created_at DESC, po.id, v.voted_at`,
		userID)
	if err != nil {
		log.Printf("❌ Ошибка получения списка голосований для inline: %v", err)
		return c.Answer(&telebot.QueryResponse{
			Results:   telebot.Results{},
			CacheTime: 10,
			Button:    createPollButton,
		})
	}
	defer rows.Close()

	// Собираем данные голосований из плоского результата
	pollsMap := make(map[int64]*PollData)
	pollsOrder := make([]int64, 0)
	optionsMap := make(map[int64]*PollOption)

	for rows.Next() {
		var pollID int64
		var title string
		var createdAt time.Time
		var optionID *int64
		var optionText *string
		var emoji *string
		var voteUserID *int64
		var voteUsername *string
		var voteFirstName *string
		var voteLastName *string

		if err := rows.Scan(&pollID, &title, &createdAt,
			&optionID, &optionText, &emoji,
			&voteUserID, &voteUsername, &voteFirstName, &voteLastName); err != nil {
			log.Printf("❌ Ошибка чтения данных голосования: %v", err)
			continue
		}

		// Создаем или получаем голосование
		poll, exists := pollsMap[pollID]
		if !exists {
			poll = &PollData{
				ID:      pollID,
				Title:   title,
				Options: make([]PollOption, 0),
			}
			pollsMap[pollID] = poll
			pollsOrder = append(pollsOrder, pollID)
		}

		// Добавляем вариант, если есть
		if optionID != nil && optionText != nil {
			optionKey := *optionID
			option, optExists := optionsMap[optionKey]
			if !optExists {
				emojiValue := "👍"
				if emoji != nil && *emoji != "" {
					emojiValue = *emoji
				}
				option = &PollOption{
					ID:    *optionID,
					Text:  *optionText,
					Emoji: emojiValue,
					Votes: make([]Vote, 0),
				}
				poll.Options = append(poll.Options, *option)
				// Сохраняем указатель на последний добавленный вариант
				optionsMap[optionKey] = &poll.Options[len(poll.Options)-1]
				option = optionsMap[optionKey]
			}

			// Добавляем голос, если есть
			if voteUserID != nil {
				vote := Vote{
					UserID: *voteUserID,
				}
				if voteUsername != nil {
					vote.Username = *voteUsername
				}
				if voteFirstName != nil {
					vote.FirstName = *voteFirstName
				}
				if voteLastName != nil {
					vote.LastName = *voteLastName
				}
				option.Votes = append(option.Votes, vote)
				poll.TotalVotes++
			}
		}
	}

	// Формируем результаты для inline-режима
	results := make(telebot.Results, 0)

	for _, pollID := range pollsOrder {
		poll := pollsMap[pollID]

		// Форматируем сообщение голосования
		pollText := formatPollMessage(poll)

		// Создаем inline-кнопки для голосования
		markup := &telebot.ReplyMarkup{}
		btnRows := make([]telebot.Row, 0)
		for _, opt := range poll.Options {
			btn := markup.Data(opt.Text, "vote", strconv.FormatInt(poll.ID, 10), strconv.FormatInt(opt.ID, 10))
			btnRows = append(btnRows, markup.Row(btn))
		}
		markup.Inline(btnRows...)

		// Получаем дату создания (можно сохранить в PollData, но для простоты используем текущее время)
		result := &telebot.ArticleResult{
			ResultBase: telebot.ResultBase{
				ID:          strconv.FormatInt(poll.ID, 10),
				Type:        "article",
				ReplyMarkup: markup,
			},
			Title: poll.Title,
			Text:  pollText,
		}

		results = append(results, result)
	}

	// Если ничего не найдено, показываем информационное сообщение
	if len(results) == 0 {
		noResultMsg := "📊 Нет активных голосований"

		result := &telebot.ArticleResult{
			ResultBase: telebot.ResultBase{
				ID:   "no_results",
				Type: "article",
			},
			Title:       noResultMsg,
			Description: "Создайте новое голосование с помощью /createpoll",
			Text:        fmt.Sprintf("%s\n\nИспользуйте команду /createpoll в личном чате с ботом, чтобы создать новое голосование.", noResultMsg),
		}
		results = append(results, result)
	}

	return c.Answer(&telebot.QueryResponse{
		Results:   results,
		CacheTime: 10, // Кешировать на 10 секунд
		Button:    createPollButton,
	})
}

// handleChosenInlineResult обрабатывает событие выбора inline-результата
// (когда пользователь отправляет голосование в чат через inline-режим)
func (b *Bot) handleChosenInlineResult(c telebot.Context) error {
	// Пытаемся получить InlineResult из контекста
	result := c.InlineResult()
	if result == nil {
		log.Printf("⚠️ InlineResult is nil")
		return nil
	}

	// ResultID содержит pollID (мы устанавливали его в handleInlineQuery)
	pollID, err := strconv.ParseInt(result.ResultID, 10, 64)
	if err != nil {
		log.Printf("❌ Ошибка парсинга poll ID из inline result: %v", err)
		return nil
	}

	// InlineMessageID - уникальный идентификатор inline-сообщения
	inlineMessageID := result.MessageID
	if inlineMessageID == "" {
		log.Printf("⚠️ InlineMessageID пустой для poll_id=%d", pollID)
		return nil
	}

	ctx := context.Background()

	// Сохраняем информацию об отправке inline-голосования
	_, err = b.db.Exec(ctx,
		`INSERT INTO voting.poll_chats (poll_id, inline_message_id, message_hash, created_at) 
		 VALUES ($1, $2, $3, NOW())
		 ON CONFLICT (poll_id, inline_message_id) WHERE inline_message_id IS NOT NULL 
		 DO NOTHING`,
		pollID, inlineMessageID, uint64(0))
	if err != nil {
		log.Printf("❌ Ошибка сохранения inline-публикации в poll_chats: %v", err)
		return nil
	}

	log.Printf("✅ Inline-голосование %d отправлено пользователем %d (inline_msg_id=%s, hash=%d)",
		pollID, c.Sender().ID, inlineMessageID, uint64(0))

	return nil
}

// updatePollMessages обновляет все опубликованные сообщения для указанного голосования.
// Пропускает обновление если хеш сообщения не изменился.
// Вызывается из воркера очереди обновлений.
func (b *Bot) updatePollMessages(pollID int64) {
	ctx := context.Background()

	// Получаем актуальные данные голосования
	poll, err := b.getPollData(ctx, pollID)
	if err != nil {
		log.Printf("❌ [UpdateWorker] Ошибка получения данных голосования %d: %v", pollID, err)
		return
	}

	// Форматируем текст и вычисляем хеш
	msg := formatPollMessage(poll)
	newHash := int64(FastHash(msg))

	markup := &telebot.ReplyMarkup{}
	btnRows := make([]telebot.Row, 0)
	for _, opt := range poll.Options {
		btn := markup.Data(opt.Text, "vote", strconv.FormatInt(pollID, 10), strconv.FormatInt(opt.ID, 10))
		btnRows = append(btnRows, markup.Row(btn))
	}
	markup.Inline(btnRows...)

	// Получаем все опубликованные сообщения для этого голосования (включая хеш)
	rows, err := b.db.Query(ctx,
		`SELECT id, chat_id, message_id, inline_message_id, message_hash FROM voting.poll_chats WHERE poll_id = $1`,
		pollID)
	if err != nil {
		log.Printf("❌ [UpdateWorker] Ошибка получения чатов для голосования %d: %v", pollID, err)
		return
	}
	defer rows.Close()

	updated := 0
	skipped := 0
	for rows.Next() {
		var rowID int64
		var chatID *int64
		var messageID *int64
		var inlineMessageID *string
		var messageHash *int64

		if err := rows.Scan(&rowID, &chatID, &messageID, &inlineMessageID, &messageHash); err != nil {
			log.Printf("❌ [UpdateWorker] Ошибка чтения данных poll_chats: %v", err)
			continue
		}

		// Проверяем хеш — если не изменился, пропускаем обновление
		if messageHash != nil && *messageHash == newHash {
			skipped++
			continue
		}

		var editErr error
		var ok bool
		if inlineMessageID != nil && *inlineMessageID != "" {
			// Inline-сообщение
			storedMsg := &telebot.StoredMessage{
				MessageID: *inlineMessageID,
			}
			_, editErr = b.bot.Edit(storedMsg, msg, markup)
			ok = CheckIsUpdatingSuccess(editErr)
			if !ok {
				log.Printf("❌ [UpdateWorker] Ошибка обновления inline-сообщения %s (poll=%d): %v",
					*inlineMessageID, pollID, editErr)
			}
		} else if chatID != nil && messageID != nil {
			// Обычное сообщение в чате
			storedMsg := &telebot.StoredMessage{
				MessageID: strconv.FormatInt(*messageID, 10),
				ChatID:    *chatID,
			}
			_, editErr = b.bot.Edit(storedMsg, msg, markup)
			ok = CheckIsUpdatingSuccess(editErr)
			if !ok {
				log.Printf("❌ [UpdateWorker] Ошибка обновления сообщения (chat=%d, msg=%d, poll=%d): %v",
					*chatID, *messageID, pollID, editErr)
			}
		} else {
			continue
		}

		// После успешного обновления сохраняем новый хеш
		if ok {
			_, err = b.db.Exec(ctx,
				`UPDATE voting.poll_chats SET message_hash = $1 WHERE id = $2`,
				newHash, rowID)
			if err != nil {
				log.Printf("❌ [UpdateWorker] Ошибка сохранения хеша для poll_chats id=%d: %v", rowID, err)
			}
			updated++
		}
	}

	log.Printf("✅ [UpdateWorker] Голосование %d: обновлено %d, пропущено %d (хеш не изменился)", pollID, updated, skipped)
}

func CheckIsUpdatingSuccess(editErr error) bool {
	if editErr == nil {
		return true
	}
	if errors.Is(editErr, telebot.ErrTrueResult) {
		return true
	}
	if editErr.Error() == SameMessageError {
		return true
	}

	return false
}

// FastHash быстрая хеш-функция для строк
func FastHash(s string) uint64 {
	var h uint64 = 146527 // random prime-ish

	for i := 0; i < len(s); i++ {
		h = (h * 31) ^ uint64(s[i])
	}

	return h
}
