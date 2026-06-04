package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

const (
	callbackMenu         = "menu"
	callbackShowDefaults = "defaults:show"
	callbackSetRadius    = "defaults:set:radius"
	callbackSetTrip      = "defaults:set:trip"
	callbackSetFrame     = "defaults:set:frame"
	callbackPredict      = "predict"
)

type botApp struct {
	store  *userStore
	client *predictClient
}

func newBotApp(store *userStore, client *predictClient) *botApp {
	return &botApp{store: store, client: client}
}

func (a *botApp) handleUpdate(ctx context.Context, b *bot.Bot, update *models.Update) {
	switch {
	case update.CallbackQuery != nil:
		a.handleCallback(ctx, b, update)
	case update.Message != nil:
		a.handleMessage(ctx, b, update.Message)
	default:
		log.Printf("default")
	}
}

func (a *botApp) handleMessage(ctx context.Context, b *bot.Bot, message *models.Message) {
	userID, ok := messageUserID(message)
	if !ok {
		a.sendText(ctx, b, message.Chat.ID, "Please use this bot from your Telegram user account.")
		return
	}

	if _, err := a.store.ensureUser(ctx, userID); err != nil {
		log.Printf("ensure user failed: %v", err)
		a.sendText(ctx, b, message.Chat.ID, "Could not load user settings. Please try again in a moment.")
		return
	}

	if message.Location != nil {
		a.handleLocation(ctx, b, message, userID)
		return
	}

	switch message.Text {
	case "/start", "/settings":
		a.sendMenu(ctx, b, message.Chat.ID, userID)
	default:
		a.sendMenu(ctx, b, message.Chat.ID, userID)
	}
}

func (a *botApp) handleCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	query := update.CallbackQuery
	_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
		CallbackQueryID: query.ID,
		ShowAlert:       false,
	})

	chatID, messageID, ok := callbackMessageTarget(query)
	if !ok {
		return
	}

	userID := query.From.ID
	if _, err := a.store.ensureUser(ctx, userID); err != nil {
		log.Printf("ensure user failed: %v", err)
		a.editText(ctx, b, chatID, messageID, "Could not load user settings. Please try again in a moment.", mainMenuKeyboard())
		return
	}

	switch {
	case query.Data == callbackMenu:
		a.editMenu(ctx, b, chatID, messageID, userID)
	case query.Data == callbackShowDefaults:
		defaults, err := a.store.getDefaults(ctx, userID)
		if err != nil {
			log.Printf("get defaults failed: %v", err)
			a.editText(ctx, b, chatID, messageID, "Could not load user settings. Please try again in a moment.", mainMenuKeyboard())
			return
		}
		a.editText(ctx, b, chatID, messageID, defaultsText(defaults), mainMenuKeyboard())
	case query.Data == callbackSetRadius:
		a.editText(ctx, b, chatID, messageID, "Choose the default radius:", radiusKeyboard())
	case query.Data == callbackSetTrip:
		a.editText(ctx, b, chatID, messageID, "Choose the default trip duration:", tripDurationKeyboard())
	case query.Data == callbackSetFrame:
		a.editText(ctx, b, chatID, messageID, "Choose the default time frame:", timeFrameKeyboard())
	case query.Data == callbackPredict:
		a.askForLocation(ctx, b, chatID)
	default:
		if a.handleValueCallback(ctx, b, chatID, messageID, userID, query.Data) {
			return
		}
		a.editMenu(ctx, b, chatID, messageID, userID)
	}
}

func (a *botApp) handleValueCallback(ctx context.Context, b *bot.Bot, chatID int64, messageID int, userID int64, data string) bool {
	parts := strings.Split(data, ":")
	if len(parts) != 3 || parts[0] != "defaults" {
		return false
	}

	var (
		defaults userDefaults
		err      error
	)

	switch parts[1] {
	case "radius":
		value, parseErr := strconv.ParseFloat(parts[2], 64)
		if parseErr != nil {
			err = parseErr
			break
		}
		defaults, err = a.store.updateRadius(ctx, userID, value)
	case "trip":
		value, parseErr := strconv.Atoi(parts[2])
		if parseErr != nil {
			err = parseErr
			break
		}
		defaults, err = a.store.updateTripDuration(ctx, userID, value)
	case "frame":
		value, parseErr := strconv.Atoi(parts[2])
		if parseErr != nil {
			err = parseErr
			break
		}
		defaults, err = a.store.updateTimeFrame(ctx, userID, value)
	default:
		return false
	}

	if err != nil {
		log.Printf("update defaults failed: %v", err)
		a.editText(ctx, b, chatID, messageID, "I could not update that setting. Please try again.", mainMenuKeyboard())
		return true
	}

	a.editText(ctx, b, chatID, messageID, "Updated.\n\n"+defaultsText(defaults), mainMenuKeyboard())
	return true
}

func (a *botApp) handleLocation(ctx context.Context, b *bot.Bot, message *models.Message, userID int64) {
	defaults, err := a.store.getDefaults(ctx, userID)
	if err != nil {
		log.Printf("get defaults failed: %v", err)
		a.sendText(ctx, b, message.Chat.ID, "I could not load your settings. Please try again in a moment.")
		return
	}

	a.sendTextWithMarkup(ctx, b, message.Chat.ID, "Checking the forecast...", &models.ReplyKeyboardRemove{RemoveKeyboard: true})

	timezone, location, err := timezoneForCoordinates(message.Location.Latitude, message.Location.Longitude)
	if err != nil {
		log.Printf("timezone lookup failed: %v", err)
		a.sendText(ctx, b, message.Chat.ID, "Failed to determine the timezone for that location.")
		a.sendMenu(ctx, b, message.Chat.ID, userID)
		return
	}

	windows, err := a.client.predict(ctx, message.Location.Latitude, message.Location.Longitude, defaults, time.Now().In(location), timezone)
	if err != nil {
		log.Printf("predict failed: %v", err)
		a.sendText(ctx, b, message.Chat.ID, "Prediction failed: "+err.Error())
		a.sendMenu(ctx, b, message.Chat.ID, userID)
		return
	}

	a.sendText(ctx, b, message.Chat.ID, formatPrediction(windows, location))
	a.sendMenu(ctx, b, message.Chat.ID, userID)
}

func (a *botApp) sendMenu(ctx context.Context, b *bot.Bot, chatID int64, userID int64) {
	defaults, err := a.store.getDefaults(ctx, userID)
	if err != nil {
		log.Printf("get defaults failed: %v", err)
		a.sendText(ctx, b, chatID, "Failed to load user settings. Please try again in a moment.")
		return
	}

	a.sendTextWithMarkup(ctx, b, chatID, menuText(defaults), mainMenuKeyboard())
}

func (a *botApp) editMenu(ctx context.Context, b *bot.Bot, chatID int64, messageID int, userID int64) {
	defaults, err := a.store.getDefaults(ctx, userID)
	if err != nil {
		log.Printf("get defaults failed: %v", err)
		a.editText(ctx, b, chatID, messageID, "Failed to load user settings. Please try again in a moment.", mainMenuKeyboard())
		return
	}

	a.editText(ctx, b, chatID, messageID, menuText(defaults), mainMenuKeyboard())
}

func (a *botApp) askForLocation(ctx context.Context, b *bot.Bot, chatID int64) {
	keyboard := &models.ReplyKeyboardMarkup{
		Keyboard: [][]models.KeyboardButton{
			{
				{Text: "Share location and predict", RequestLocation: true},
			},
		},
		ResizeKeyboard:        true,
		OneTimeKeyboard:       true,
		InputFieldPlaceholder: "Share your current location",
	}

	a.sendTextWithMarkup(ctx, b, chatID, "Please share your current location so I can call the prediction API.", keyboard)
}

func (a *botApp) sendText(ctx context.Context, b *bot.Bot, chatID int64, text string) {
	a.sendTextWithMarkup(ctx, b, chatID, text, nil)
}

func (a *botApp) sendTextWithMarkup(ctx context.Context, b *bot.Bot, chatID int64, text string, markup models.ReplyMarkup) {
	if _, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:      chatID,
		Text:        text,
		ReplyMarkup: markup,
	}); err != nil {
		log.Printf("send message failed: %v", err)
	}
}

func (a *botApp) editText(ctx context.Context, b *bot.Bot, chatID int64, messageID int, text string, markup models.ReplyMarkup) {
	if _, err := b.EditMessageText(ctx, &bot.EditMessageTextParams{
		ChatID:      chatID,
		MessageID:   messageID,
		Text:        text,
		ReplyMarkup: markup,
	}); err != nil {
		log.Printf("edit message failed: %v", err)
	}
}

func messageUserID(message *models.Message) (int64, bool) {
	if message.From == nil {
		return 0, false
	}

	return message.From.ID, true
}

func callbackMessageTarget(query *models.CallbackQuery) (int64, int, bool) {
	if query.Message.Message == nil {
		return 0, 0, false
	}

	return query.Message.Message.Chat.ID, query.Message.Message.ID, true
}

func mainMenuKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "Predict", CallbackData: callbackPredict},
			},
			{
				{Text: "Show defaults", CallbackData: callbackShowDefaults},
			},
			{
				{Text: "Radius", CallbackData: callbackSetRadius},
				{Text: "Trip duration", CallbackData: callbackSetTrip},
			},
			{
				{Text: "Time frame", CallbackData: callbackSetFrame},
			},
		},
	}
}

func radiusKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "1 km", CallbackData: "defaults:radius:1"},
				{Text: "2 km", CallbackData: "defaults:radius:2"},
				{Text: "5 km", CallbackData: "defaults:radius:5"},
			},
			{
				{Text: "10 km", CallbackData: "defaults:radius:10"},
				{Text: "Back", CallbackData: callbackMenu},
			},
		},
	}
}

func tripDurationKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "10 min", CallbackData: "defaults:trip:10"},
				{Text: "20 min", CallbackData: "defaults:trip:20"},
				{Text: "30 min", CallbackData: "defaults:trip:30"},
			},
			{
				{Text: "45 min", CallbackData: "defaults:trip:45"},
				{Text: "60 min", CallbackData: "defaults:trip:60"},
			},
			{
				{Text: "Back", CallbackData: callbackMenu},
			},
		},
	}
}

func timeFrameKeyboard() *models.InlineKeyboardMarkup {
	return &models.InlineKeyboardMarkup{
		InlineKeyboard: [][]models.InlineKeyboardButton{
			{
				{Text: "30 min", CallbackData: "defaults:frame:30"},
				{Text: "60 min", CallbackData: "defaults:frame:60"},
				{Text: "120 min", CallbackData: "defaults:frame:120"},
			},
			{
				{Text: "180 min", CallbackData: "defaults:frame:180"},
				{Text: "Back", CallbackData: callbackMenu},
			},
		},
	}
}

func menuText(defaults userDefaults) string {
	return "Press Predict and share your location to find the best trip time."
}

func defaultsText(defaults userDefaults) string {
	return fmt.Sprintf(
		"Defaults:\nRadius: %.1f km\nTrip duration: %d min\nTime frame: %d min",
		defaults.DefaultRadiusKM,
		defaults.TripDurationMin,
		defaults.TimeFrameMin,
	)
}

func formatPrediction(windows []predictWindow, location *time.Location) string {
	if len(windows) == 0 {
		return "The API did not return any available prediction windows."
	}

	lines := []string{"Best start windows:"}
	for i, window := range windows {
		lines = append(lines, fmt.Sprintf(
			"%d. %s, %s, %.2f mm/h",
			i+1,
			formatPredictionWindowTime(window, location),
			window.Description,
			window.Precipitation,
		))
	}

	return strings.Join(lines, "\n")
}

func formatPredictionWindowTime(window predictWindow, location *time.Location) string {
	startTime := window.StartTime.In(location)
	if window.EndTime.IsZero() {
		return startTime.Format("15:04 MST")
	}

	endTime := window.EndTime.In(location)
	return fmt.Sprintf("%s-%s", startTime.Format("15:04"), endTime.Format("15:04 MST"))
}
