package bot

import (
	"sync"
)

// State представляет состояние диалога с пользователем
type State string

const (
	StateIdle              State = "idle"                // Ожидание
	StateCreatePollTitle   State = "create_poll_title"   // Создание голосования: ввод заголовка
	StateCreatePollOption  State = "create_poll_option"  // Создание голосования: ввод варианта
	StateCreatePollConfirm State = "create_poll_confirm" // Создание голосования: подтверждение
)

// DialogContext хранит контекст диалога пользователя
type DialogContext struct {
	State State                  // Текущее состояние
	Data  map[string]interface{} // Данные диалога
}

// DialogManager управляет состояниями диалогов пользователей
type DialogManager struct {
	mu       sync.RWMutex
	sessions map[int64]*DialogContext // userID -> контекст диалога
}

// NewDialogManager создает новый менеджер диалогов
func NewDialogManager() *DialogManager {
	return &DialogManager{
		sessions: make(map[int64]*DialogContext),
	}
}

// GetContext получает контекст диалога пользователя
func (dm *DialogManager) GetContext(userID int64) *DialogContext {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	ctx, exists := dm.sessions[userID]
	if !exists {
		return &DialogContext{
			State: StateIdle,
			Data:  make(map[string]interface{}),
		}
	}
	return ctx
}

// SetState устанавливает состояние диалога для пользователя
func (dm *DialogManager) SetState(userID int64, state State) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	ctx, exists := dm.sessions[userID]
	if !exists {
		ctx = &DialogContext{
			State: state,
			Data:  make(map[string]interface{}),
		}
		dm.sessions[userID] = ctx
	} else {
		ctx.State = state
	}
}

// SetData сохраняет данные в контекст диалога
func (dm *DialogManager) SetData(userID int64, key string, value interface{}) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	ctx, exists := dm.sessions[userID]
	if !exists {
		ctx = &DialogContext{
			State: StateIdle,
			Data:  make(map[string]interface{}),
		}
		dm.sessions[userID] = ctx
	}
	ctx.Data[key] = value
}

// GetData получает данные из контекста диалога
func (dm *DialogManager) GetData(userID int64, key string) (interface{}, bool) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	ctx, exists := dm.sessions[userID]
	if !exists {
		return nil, false
	}
	value, ok := ctx.Data[key]
	return value, ok
}

// ResetContext сбрасывает контекст диалога пользователя
func (dm *DialogManager) ResetContext(userID int64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	dm.sessions[userID] = &DialogContext{
		State: StateIdle,
		Data:  make(map[string]interface{}),
	}
}

// DeleteContext удаляет контекст диалога пользователя
func (dm *DialogManager) DeleteContext(userID int64) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	delete(dm.sessions, userID)
}

// GetAllSessions возвращает количество активных сессий
func (dm *DialogManager) GetAllSessions() int {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	return len(dm.sessions)
}
