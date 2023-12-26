package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Создать место под возвращаемый комментарий от препода

const SECRET = "2dea7b2aff32ced454b3140fa3df5355755842b1"

// БОТ. Реализовать возможность проверки токена запроса доступа от пользователя
// + Бот. Реализовать удаление вышедшего пользователя +  (заполнить адреса и код ошибки)
// + Бот. Реализовать передачу данных модулю Расписание + (заполнить URL-адрес и сделтаь проверку кода ошибки)

// База данных бота
var OpenSessions = GetOpenSessions()

// формируем начало адреса, на который будут посылаться запросы для получения обновлений или запрос на ответ пользователю
var URLadress = "https://api.telegram.org/bot6886604446:AAHbdWg9kKyNixWHTrnhkgZxwbqrpjshTjE"

// Инициализируем начальное значение параметра offset для функции getUpdate
var offset = getLastUpdateID()

// t.me/Bot_bot_GIIT_TEST123141_bot
func main() {

	// Инициализируем роутер (маршрутизатор) для нашего сервера.
	// Подробнее тут: https://vladimirchabanov.notion.site/HTTP-21b7158cf6254070887a24c8b23d7988 — в пункте "Router (маршрутизатор)"
	go callbacServe()

	// Отслеживаем новые сообщения
	for offset >= 0 {
		// Забираем массив обновлений, помещая в URL значение параметра offset
		// offset — идентификатор первого возвращаемого обновления. Должен быть на 1 больше, чем самый высокий среди идентификаторов
		// ранее полученных обновлений, поэтому в конце каждой итерации цикла, прокручивающего обновления, мы увеличиваем offset на 1
		updates, err := getUpdates(URLadress, offset)
		if err != nil {
			log.Println("HTTP Error: ", err.Error())
		}

		log.Println(updates)

		// Перебираем элементы массива обновлений
		for _, update := range updates {
			log.Println("Пришло сообщение от пользователя " + strconv.Itoa(update.Message.Chat.ChatId))

			// Проверяем есть ли chat_id пользователя в списке открытых сессий
			_, ok := OpenSessions[update.Message.Chat.ChatId]

			// Считываем код действия
			actionCode, codeADMINKA := AnalyzeTheUserRequest(update)
			// Если его нет, значит посылаем запрос на авторизацию и уведомляем пользователя
			if !ok {
				// С помощью функции, описанной ниже, получаем ссылку авторизации
				responce_link := OauthRequest(update.Message.Chat.ChatId)

				// Тут формируем текст сообщения пользователю и отсылаем
				responce := "Пожалуйста, войдите в систему по ссылке:\n " + responce_link + " \nи повторите попытку."
				err := respond(update.Message.Chat.ChatId, URLadress, responce)
				if err != nil {
					// При возникновении ошибки пишем ошибку в лог и прерываем выполнение функции
					log.Println("Ошибка ответа пользователю для обновления " + strconv.Itoa(update.Updateid) + "!")
				}
			} else {

				if actionCode == "restart" {
					err := respond(update.Message.Chat.ChatId, URLadress, "Неправильно введён код действия!")
					if err != nil {
						fmt.Println("Ошибка при отправке ответа")
					}

				} else if codeADMINKA == "0" {
					ExecuteARequest(update, actionCode, URLadress, "0")
				} else {
					// Проверяем верность кода действия
					actionCode = "toadmin"
					ExecuteARequest(update, actionCode, URLadress, codeADMINKA)
				}
			}

			// Увеличиваем на 1 значение параметра offset
			offset = update.Updateid + 1
		}
	}
}

func callbacServe() {
	router := http.NewServeMux()

	// Регистрируем адрес, на который модуль Авторизация будет отсылать GitHub_id и chat_id вошедшего пользователя
	// А также указываем функцию-handler, либо функцию, которая принимает handler как параметр
	router.HandleFunc("/callback", callbackSendMessage) // Теоретически должно работать

	// Запускаем сервер на порту 8082
	http.ListenAndServe("localhost:8082", router)
}

func ExecuteARequest(update Update, actionCode string, URLadress string, codeADMINKF string) {

	// Функция AnalyzeTheUserRequest работает так, что actionCode останется пустым, если программа не умеет делать то, что просит пользователь
	if actionCode == "" {
		err := respond(update.Message.Chat.ChatId, URLadress, "Неправильно введён код действия!")
		if err != nil {
			log.Println("Ошибка ответа пользователю 4 для обновления " + strconv.Itoa(update.Updateid) + "!")
		}
		return
	} else if actionCode == "Enter" {
		response := "Вы уже авторизованы"
		err := respond(update.Message.Chat.ChatId, URLadress, response)
		if err != nil {
			log.Println("Ошибка ответа пользователю!")
		}
		return
	} else if actionCode == "Exit" {
		// Удаляем пользователя из списка открытых сессий //
		respose := deleteSession(update)
		err := respond(update.Message.Chat.ChatId, URLadress, respose)
		if err != nil {
			log.Println("Ошибка отправки сообщения пользователю!")
		}
		return
	}

	// Проверяем хватает ли у пользователя прав на это действие
	JWT_token := SendRequestToRightsVerification(actionCode, getUserIdGitHub(update.Message.Chat.ChatId))
	if JWT_token == "Ошибка кода действия!" {
		responce := "У вас не хватает прав!"
		err := respond(update.Message.Chat.ChatId, URLadress, responce)
		if err != nil {
			log.Println("Ошибка ответа пользователю 2 для обновления " + strconv.Itoa(update.Updateid) + "!")
		}
		return
	}

	// Отсылаем запрос модулю и забираем его ответ (ГОТОВЫЙ ДЛЯ ОТСЫЛАНИЯ ПОЛЬЗОВАТЕЛЮ)
	responce, err := sendRequestWhithUserRequest(actionCode, JWT_token, codeADMINKF) // Доработать
	if err != nil {
		log.Println("Ошибка в обновлении " + strconv.Itoa(update.Updateid) + ":\n" + err.Error())
	}

	// Отсылаем пользователю сообщение
	err = respond(update.Message.Chat.ChatId, URLadress, responce)
	if err != nil {
		log.Println("Ошибка ответа пользователю 3 для обновления" + strconv.Itoa(update.Updateid) + "!")
	}
}

// Создаём функцию, через которую мы сможем выслать ответ пользователю
func callbackSendMessage(w http.ResponseWriter, r *http.Request) {

	// Проверяем код ошибки (200 — ошибок нет)
	if r.URL.Query().Get("codeError") == "200" {
		// Создаём нового авторизованного пользователя (новую сессию)
		var user User
		user.Chat_id, _ = strconv.Atoi(r.URL.Query().Get("chat_id"))
		user.GitHub_id, _ = strconv.Atoi(r.URL.Query().Get("GitHub_id"))

		// Добавляем его в список сессий
		OpenSessions[user.Chat_id] = user
		SafeOpenSessions(OpenSessions)
		log.Println("Пользователь " + strconv.Itoa(user.Chat_id) + "вошёл")

		// Отсылаем ответ пользователю
		err := respond(user.Chat_id, URLadress, "Вы успешно вошли!")
		if err != nil {
			log.Println("Ошибка сообщения пользователю!")
		}

		// Возвращаем модулю Авторизация код успешного завершения операции
		w.Write([]byte("200"))
	} else {
		chat_id, _ := strconv.Atoi(r.URL.Query().Get("chat_id"))

		// В случае ошибки уведомляем пользователя об ошибке
		err := respond(chat_id, URLadress, "Ошибка! Пожалуйста, попробуйте войти позже")
		if err != nil {
			log.Println("Ошибка сообщения пользователю!")
		}

		w.Write([]byte("209"))
	}
}

// Формируем запрос на авторизацию
func OauthRequest(chat_id int) string {
	// Регистрируем клиента
	cli := http.Client{}

	// Формируем URL-адрес, куда будет посылаться запрос, в который помещаем chta_id и ссылку, на которую можно будет прислать GitHub_id
	oauthURL := "http://localhost:8080/Oauth?chat_id=" + strconv.Itoa(chat_id) + "&callback=http://localhost:8082/callback"

	// Формируем и отсылаем запрос соответственно
	request, _ := http.NewRequest("GET", oauthURL, nil)
	response, _ := cli.Do(request)

	for response == nil {
		response, _ = cli.Do(request)
		time.Sleep(1000)
	}

	// Забираем тело ответа в виде массива байтов
	resBody, _ := io.ReadAll(response.Body)
	defer response.Body.Close()

	// Возвращаем строку — URL-адрес, по которому можно авторизоваться
	return string(resBody)
}

// Отправляем запрос на проверку прав
func SendRequestToRightsVerification(actionCode string, GitHub_id int) string {
	// Регистрируем клиента
	cli := http.Client{}

	// Формируем URL-адрес запроса (на него будет посылаться запрос)
	rightsURL := "http://localhost:8080/rights?GitHub_id=" + strconv.Itoa(GitHub_id) + "&action_code=" + actionCode

	// Формируем запрос и отсылаем
	request, _ := http.NewRequest("GET", rightsURL, nil)
	response, _ := cli.Do(request)

	for response == nil {
		response, _ = cli.Do(request)
		time.Sleep(1000)
	}

	// Забираем JWT-токен в виде массива байтов
	resBody, _ := io.ReadAll(response.Body)
	defer response.Body.Close()

	// Возвращаем строку — JWT-токен с зашифрованной в нём информацией о пользователе
	return string(resBody)
}

// Отправляем запрос модулю в соответствии с запросом пользователя
func sendRequestWhithUserRequest(actionCode, JWT_token string, codeADMINKA string) (string, error) {
	cli := http.Client{}

	// Отсылаем запрос с необходимыми данными //
	if actionCode == "toadmin" { // для админки
		// Высылаем запрос админке, куда ложим JWT-токен и GitHub_id

		var URLadress string
		if codeADMINKA == "0" {
			URLadress = "http://localhost:8083/StartSession?JWTToken=" + JWT_token + "&Code=" + codeADMINKA
		} else {
			URLadress = "http://localhost:8083/AccessUser?JWTToken=" + JWT_token + "&Code=" + codeADMINKA
		}

		// Формируем и отсылаем Get-запрос модулю
		request, err := http.NewRequest("GET", URLadress, nil)
		if err != nil {
			return "Не удалось войти в админку! Пожалуйста, попробуйте позже", err
		}

		reaponse, err := cli.Do(request)
		if err != nil {
			log.Println(err.Error())
			return "Не удалось войти в админку! Пожалуйста, попробуйте позже", nil
		}

		for reaponse == nil {
			reaponse, _ = cli.Do(request)
			log.Println("Ожидание подключения к Administration...")
			time.Sleep(1000)
		}

		// Забираем тело ответа
		resBody, err := io.ReadAll(reaponse.Body)

		// 	ЗАПОЛНИТЬ КОД ОШИБКИ
		if err != nil || string(resBody) == "" {
			return "Не удалось войти в админку! Пожалуйста, попробуйте позже", err
		}

		// Иначе, если там будет ссылка или код 200, мы формируем строку ответа:
		var responseString string
		if codeADMINKA == "0" {
			responseString = "Пожалуйста, перейдите по ссылке:\n" + string(resBody)
		} else if string(resBody) == "200" {
			responseString = "Пожалуйста, обновите страницу браузера, чтобы войти в админку."
		} else if string(resBody) == "400" {
			responseString = "Несуществующий код! Попробуйте ещё раз."
		}

		// Возвращаем отформатированный ответ
		return responseString, nil
	} else if actionCode == "changeName" || actionCode == "changeGroup" { // для изменения имени или группы пользователя

		// Распаковываем JWT-токен //
		// Извлекаем данные из токена и проверяем его
		token, _ := jwt.Parse(JWT_token, func(token *jwt.Token) (interface{}, error) {
			return []byte(SECRET), nil
		})

		// Необходимые данные
		var GitHub_id float64
		var chat_id float64

		// Вытягиваем данные из токена
		payload, ok := token.Claims.(jwt.MapClaims)
		if ok && token.Valid {
			GitHub_id = payload["Id_github"].(float64) // Значение имеет тип interface{}, поэтому через точку мы преобразовываем его в нужный тип
			chat_id = payload["id_telegram"].(float64)
		} else {
			log.Println("Ошибка распаковки JWT-токена!")
		}

		// Просим пользователя написать значение, на которое он желает поменять имя/группу
		err := respond(int(chat_id), URLadress, "Пожалуйста, введите значение. Следующее сообщение целиком будет принято за новое значение")
		if err != nil {
			return "", err
		}

		// Забираем новое значение
		// НЕ ЗАБИРАЕТ НОВОЕ ЗНАЧЕНИЕ
		offset++
		updates, _ := getUpdates(URLadress, offset)
		for len(updates) < 1 {
			updates, _ = getUpdates(URLadress, offset)
			log.Println("Ожидание подключения к Telegram...")
			time.Sleep(100)
		}
		fmt.Println(updates)
		var NewValue string
		for _, update := range updates {
			if update.Message.Chat.ChatId == int(chat_id) {
				NewValue = update.Message.Text
			}
		}

		changingValue := actionCode[6:]

		form := url.Values{}
		form.Add("GitHub_id", strconv.Itoa(int(GitHub_id)))
		form.Add("ChangingValue", changingValue)
		form.Add("NewValue", NewValue)

		URLadress := "http://localhost:8080/changeDataOfUsers"
		log.Println(URLadress)

		request, _ := http.NewRequest("POST", URLadress, strings.NewReader(form.Encode()))
		request.Header.Set("Content-type", "application/x-www-form-urlencoded")
		response, _ := cli.Do(request) // 400 Bad Request, то есть запрос даже не доходит до модуля Авторизация

		for response == nil {
			response, _ = cli.Do(request)
			log.Println("Ожидание подключения к Authorization...")
			time.Sleep(1000)
		}

		resBody, _ := io.ReadAll(response.Body)
		if string(resBody) != "200" {
			log.Println("Не удалось изменить данные!")
			err := errors.New("не удалось изменить данные")
			return "Не удалось изменить данны! Пожалуйста, попробуйте позже", err
		}

		return "Данные успешно изменены!", nil
	} else { // для расписания
		// Высылаем запрос модулю Расписание, куда ложим JWT-токен и actionCode

		URLadressSchadule := "http://localhost:8089/getSchedule" + "?actionCode=" + actionCode + "&JWTtoken=" + JWT_token

		// Формируем и отсылаем Get-запрос модулю
		request, err := http.NewRequest("GET", URLadressSchadule, nil)
		if err != nil {
			return "Не удалось выполнить запрос! Пожалуйста, попробуйте позже.", err
		}
		response, err := cli.Do(request)
		if err != nil {
			return "Не удалось выполнить запрос! Пожалуйста, попробуйте позже.", err
		}

		for response == nil {
			response, _ = cli.Do(request)
			log.Println("Ожидание подключения к Schedule...")
			time.Sleep(1000)
		}

		// Забираем тело ответа
		resBody, err := io.ReadAll(response.Body)
		if err != nil {
			return "Не удалось выполнить запрос! Пожалуйста, попробуйте позже.", err
		}

		// Вовзращаем отформатированный ответ пользователю
		return string(resBody), nil
	}
} // Возвращаем ответ пользователю

// Анализируем запрос пользователя
// // Может, придётся поменять путь формирования кода действия для изменения комментария // //
func AnalyzeTheUserRequest(update Update) (string, string) {

	// Забираем текст сообщения пользователя
	userRequest := update.Message.Text

	// message[0] — actionCode, message[1] — параметр
	var message = strings.Split(userRequest, "\n")

	codeADMINKA, err := strconv.ParseInt(message[0], 16, 64)
	if err == nil {
		real_codeADMINKA := strconv.FormatInt(codeADMINKA, 16)
		return "", real_codeADMINKA
	}

	// Определяем код действия
	switch message[0] {
	case "toadmin":
	case "Перейти в админку":
		message[0] = "toadmin"
	case "Перейти на страницу администрирования":
		message[0] = "toadmin"
	case "Где следующая пара": //
		message[0] = "whereIsTheNextPair@"
	case "Где группа": // Возможно, придётся разбить на "группа" и "подгруппа"
		message[0] = "whereIsTheGroup@" // +
	case "Где преподаватель":
		message[0] = "whereIsTheTeacher@" // +
	case "Оставить комментарий к паре":
		message[0] = "setComment@" // +
	case "Когда экзамен":
		message[0] = "whenIsTheExam@" // Просто выдаём все даты экзаменов
	case "Выйти":
		message[0] = "Exit"
	case "Войти":
		message[0] = "Enter"
	case "/start":
		message[0] = "Enter"
	case "Расписание на завтра":
		message[0] = "scheduleFor@Tomorrow"
	case "Расписание на сегодня":
		message[0] = "scheduleFor@Today"
	case "Расписание на понедельник":
		message[0] = "scheduleFor@Monday"
	case "Расписание на вторник":
		message[0] = "scheduleFor@Tuesday"
	case "Расписание на среду":
		message[0] = "scheduleFor@Wednesday"
	case "Расписание на четверг":
		message[0] = "scheduleFor@Thursday"
	case "Расписание на пятницу":
		message[0] = "scheduleFor@Friday"
	case "Расписание на субботу":
		message[0] = "scheduleFor@Saturday"
	case "Изменить имя": //
		message[0] = "changeName" //
	case "Изменить группу": //
		message[0] = "changeGroup" //
	default:
		message[0] = "restart"
	}

	var param string
	for i := 1; i < len(message); i++ {
		if i == len(message)-1 {
			param += strings.Replace(message[i], " ", "/!", -1)
			break
		}

		message[i] = strings.Replace(message[i], " ", "/!", -1)
		param += message[i] + "|"
	}

	return message[0] + param, "0"

	// Код действия: <тип действия>@<параметр 1>&<параметр 2>&<...>&<параметр n>&
	// Даже если параметров нет, тело кода действия будет заканчиваться на "@"
	// Ввод типа действия и параметров пользователем производится в том же порядке через перенос строки

	// Маска для кода действия "Оставить комментарий к паре":
	// LeaveACommentOnThePair@<номер пары>&<день недели>&<группа>&

	// Маска для кода действия "Гду группа?":
	// whereIsTheGroup@<Группа>&

	// Маска для кода действия "Где преподаватель?":
	// whereIsTheTeacher@<ФИО преподавателя>&

	// У остальных кодов действия параметров нет.
	// По запросу "Когда экзамен?" просто выводим из наименований предметов и дат экзаменов по ним
}

// Получаем последнее обновление
func getUpdates(URLadress string, offset int) ([]Update, error) {
	// Формируем URLadress
	URLadress = URLadress + "/getUpdates" + "?offset=" + strconv.Itoa(offset)

	// Отправляем запрос и высылаем ответ на сервер
	response, err := http.Get(URLadress)
	if err != nil {
		return nil, err
	}

	// Предусматриваем, чтобы тело ответа закрылось перед окончанием действия функции
	defer response.Body.Close()

	// Забираем тело ответа в виде json-файла
	respBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	// Создаём переменную, в которую поместим ответ от сервера Telegram
	var restResponse BotResponse
	err = json.Unmarshal(respBody, &restResponse)
	if err != nil {
		return nil, err
	}

	// Возвращаем массив обновлений и пустую ошибку
	return restResponse.Result, nil
}

// Отправляем ответ
func respond(chat_id int, URLadress string, response string) error {
	// Собираем сообщение пользователю и chat_id адресата
	var botMessage BotMessage
	botMessage.ChatId = chat_id
	botMessage.Text = response

	// Собираем данные структуры botMessage в json-файл
	buf, err := json.Marshal(botMessage)
	if err != nil {
		return err
	}

	// Формируем POST-запрос серверу Telegram и отправляем.
	// application/json — формат отправляемых с запросом данных
	// bytes.NewBuffer(buf) — создаёт и инициализирует новый буфер, используя buf как его начальное содержимое
	// Подробнее (надо с vpn): https://pkg.go.dev/bytes#NewBuffer
	myResponse, err := http.Post(URLadress+"/sendMessage", "application/json", bytes.NewBuffer(buf))
	log.Println(myResponse)
	if err != nil {
		return err
	}

	// Возвращаем пустую ошибку
	log.Println("Ответ отправлен пользователю " + strconv.Itoa(chat_id))
	return nil
}

// Работа с базой
func getUserIdGitHub(chat_id int) int {
	session, ok := OpenSessions[chat_id]
	if !ok {
		return 0
	}

	return session.GitHub_id
}

func GetOpenSessions() map[int]User {

	fileData, err := os.ReadFile("OpenSessions.txt")

	defer func() {
		panicValue := recover()
		if panicValue != nil {
			fmt.Println(panicValue)
		}
	}()

	if err != nil {
		panic(err)
	}

	Data := string(fileData)

	Sessions := make(map[int]User, len(strings.Split(Data, "\n"))-1)

	for _, SessionInfo := range strings.Split(Data, "\n") {
		if len(SessionInfo) > 1 {
			Info := strings.Split(SessionInfo, " ")

			if len(Info) < 2 {
				continue
			}

			Telegram, err1 := strconv.Atoi(Info[1])
			GitHub, err2 := strconv.Atoi(Info[0])
			if err1 != nil || err2 != nil {
				fmt.Println("Error")
			}

			SignedUser := User{
				Chat_id:   Telegram,
				GitHub_id: GitHub,
			}
			Sessions[Telegram] = SignedUser
		}
	}

	return Sessions
}

// Сохранить обновлённый список сессий
func SafeOpenSessions(Sessions map[int]User) {
	var Data string
	for _, SessionInfo := range Sessions {
		GitHub := strconv.Itoa(SessionInfo.GitHub_id)
		Telegram := strconv.Itoa(SessionInfo.Chat_id)

		Data += GitHub + " " + Telegram + " \n"
	}

	file, err := os.Create("OpenSessions.txt")

	defer func() {
		panicValue := recover()
		if panicValue != nil {
			fmt.Println(panicValue)
		}
	}()

	if err != nil {
		panic(err)
	}

	defer file.Close()

	file.WriteString(Data)
}

func deleteSession(update Update) string {
	session := OpenSessions[update.Message.Chat.ChatId]
	delete(OpenSessions, session.Chat_id)

	SafeOpenSessions(OpenSessions)

	return "Вы вышли из системы."
}

//

func getLastUpdateID() int {
	offsett := 0
	updates, err := getUpdates(URLadress, offsett)
	if err != nil {
		log.Println("Ошибка при получении обновлений.")
	}

	for _, update := range updates {
		offsett = update.Updateid
	}

	return offsett
}
