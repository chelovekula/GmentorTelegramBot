package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	tgbotapi "gopkg.in/telegram-bot-api.v4"
)

var botToken = "7942522926:AAFepbAda71ZY-eKNR6YEpwgBAZ4fFPzqJY" // Задайте токен через переменную окружения
var chatID = "-4792106902"                                      // Задайте ID чата через переменную окружения

func main() {
	if botToken == "" || chatID == "" {
		log.Fatal("TELEGRAM_BOT_TOKEN и TELEGRAM_CHAT_ID должны быть заданы")
	}

	r := gin.Default()

	// Настройка CORS
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "https://gmentor.ru")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// Настройка HTTPS
	httpsServer := &http.Server{
		Addr:    ":8443", // Измененный порт
		Handler: r,
	}

	// Универсальный роутер
	r.Any("/log", func(c *gin.Context) {
		// Логирование запроса
		logRequest(c)

		// Возвращаем ответ клиенту
		c.JSON(http.StatusOK, gin.H{
			"message": "Запрос успешно обработан",
		})
	})

	// Запуск HTTPS сервера
	log.Fatal(httpsServer.ListenAndServeTLS(
		"/etc/letsencrypt/live/bench.getmegit.com/fullchain.pem",
		"/etc/letsencrypt/live/bench.getmegit.com/privkey.pem",
	))
}

func logRequest(c *gin.Context) {
	// Логируем метод и путь
	method := c.Request.Method
	path := c.Request.URL.Path
	log.Printf("Метод: %s, Путь: %s", method, path)

	// Логируем параметры
	if len(c.Request.URL.Query()) > 0 {
		log.Printf("Параметры: %v", c.Request.URL.Query())
	}

	// Если это POST или PUT, логируем тело запроса и отправляем в Telegram
	if method == http.MethodPost || method == http.MethodPut {
		bodyBytes, err := ioutil.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Ошибка чтения тела запроса: %v", err)
			return
		}
		// Логируем тело
		log.Printf("Тело запроса: %s", string(bodyBytes))

		// Отправляем в Telegram
		sendToTelegram(string(bodyBytes))

		// Восстанавливаем тело запроса для дальнейшего использования
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	}
}

func sendToTelegram(body string) {
	var payload struct {
		Username string `json:"username"`
		Embeds   []struct {
			Description string `json:"description"`
		} `json:"embeds"`
	}

	if err := json.Unmarshal([]byte(body), &payload); err != nil {
		log.Printf("Ошибка разбора JSON: %v", err)
		return
	}

	for _, embed := range payload.Embeds {
		description := strings.ReplaceAll(embed.Description, "http://mentor.gurps.ru", "")
		description = formatDice(description)

		// Извлечение ключевых частей описания
		lines := strings.Split(description, "\n")
		var result, outcome, action string
		if len(lines) > 0 {
			action = lines[0] // Первое действие
		}
		if len(lines) > 1 {
			result = lines[1] // Результат броска
		}
		if len(lines) > 2 {
			outcome = lines[2] // Итог
		}

		message := fmt.Sprintf(
			"🎲 <b>Бросок:</b> %s\n👤 <b>Персонаж:</b> %s\n📋 <b>Действие:</b> %s\n🎯 <b>Результат:</b> %s",
			outcome, payload.Username, action, result,
		)
		if err := sendMessageToTelegram(message); err != nil {
			log.Printf("Ошибка отправки в Telegram: %v", err)
		}
	}
}

func sendMessageToTelegram(message string) error {
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		return fmt.Errorf("ошибка создания Telegram бота: %w", err)
	}

	chatIDInt64, err := parseChatID(chatID)
	if err != nil {
		return fmt.Errorf("ошибка преобразования chatID: %w", err)
	}

	msg := tgbotapi.NewMessage(chatIDInt64, message)
	msg.ParseMode = "HTML"
	_, err = bot.Send(msg)
	return err
}

func formatDice(input string) string {
	diceMap := map[string]string{
		"⚀": "1️⃣",
		"⚁": "2️⃣",
		"⚂": "3️⃣",
		"⚃": "4️⃣",
		"⚄": "5️⃣",
		"⚅": "6️⃣",
	}
	for k, v := range diceMap {
		input = strings.ReplaceAll(input, k, v)
	}
	return input
}

func parseChatID(chatID string) (int64, error) {
	var id int64
	_, err := fmt.Sscanf(chatID, "%d", &id)
	return id, err
}
