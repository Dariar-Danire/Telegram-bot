package main

import (
	"encoding/json"
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

// В последнем сценарии, если пользователь не зарегистрирован или не авторизован, может потребоваться его авторизовать.

var Users = GetData()
var DataBasePath = "DataBase.txt"
var userOauth User
var userRights User

var AuthanticateRequests = make(map[string]Authanticate, 100)

type UserDataGitHub struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type UserData struct {
	Name  string
	Group string
}

type User struct {
	GitHub_id   int
	Telegram_id int
	Roles       [3]string
	Data        UserData
	// MyToken     string // ЗАПОЛНИТЬ ПОЛЕ MyToken
}

type Authanticate struct {
	GitHub_id   string
	Telegram_id string
	Calldack    string
	Time        time.Time
}

const (
	CLIENT_ID     = "06f163e42c9edf8bc050"
	CLIENT_SECRET = "2dea7b2aff32ced454b3140fa3df5355755842b1"
)

func main() {
	router := http.NewServeMux()

	// Регистрируем маршруты
	router.HandleFunc("/rights", logging(rightsHendler))                    // Сюда запрос на проверку прав
	router.HandleFunc("/changeDataOfUsers", logging(changeUserData))        // Запрос на изменение данных о пользователе
	router.HandleFunc("/Oauth", logging(aouth1Hendler))                     // Запрос на получение ссылки на авторизацию
	router.HandleFunc("/CheckAccessForAdmin", logging(CheckAccessForAdmin)) // Запрос на проверку прав админа
	router.HandleFunc("/Oauth/redirect", logging(oauthHandler))             // Это для гитхаба!

	http.ListenAndServe(":8080", router)
}

// HANDLERS

func changeUserData(w http.ResponseWriter, r *http.Request) { //параметры: GitHub_id, ChangingValue, NewValue
	ChangingValue := r.FormValue("ChangingValue")
	if ChangingValue != "Roles" && ChangingValue != "Name" && ChangingValue != "Group" {
		fmt.Println("Error(При изменении данных пользователя не распознано значение которое надо поменять)")
		w.Write([]byte("Error(При изменении данных пользователя не распознано значение которое надо поменять)"))
	}
	GitHub_id, err := strconv.Atoi(r.FormValue("GitHub_id"))
	if err != nil {
		fmt.Println("Error(Неудачная попытка изменить данные пользователя)")
		w.Write([]byte("401"))
	}
	OurUser, ok := Users[GitHub_id]
	if !ok {
		fmt.Println("Error(При изменении данных пользователь не найден)")
		w.Write([]byte("402"))
	}
	if ChangingValue == "Name" {
		OurUser.Data.Name = r.FormValue("NewValue")
	} else if ChangingValue == "Group" {
		OurUser.Data.Group = r.FormValue("NewValue")
	} else if ChangingValue == "Roles" {
		Roles := r.FormValue("NewValue")
		OurUser.Roles = [3]string(strings.Split(Roles, "/")) //НАДЕЮСЬ ТАК ПРОКАТИТ  // Под вопросом
	}

	Users[OurUser.GitHub_id] = OurUser
	SafeData(Users)

	w.Write([]byte("200"))
}

func rightsHendler(w http.ResponseWriter, r *http.Request) {
	res := false

	// Вытаскиваем из запроса код действия и GitHub_id
	userRights.GitHub_id, _ = strconv.Atoi(r.URL.Query().Get("GitHub_id"))
	actionCode := r.URL.Query().Get("action_code") // Код действия

	// Получаем массив ролей вида ["", "", "Student"]
	user := Users[userRights.GitHub_id] // Значения GitHub_id там не быть не может
	Roles := user.Roles

	// Реализуем проверку прав       Метод костыльный, надо переделать
	if Roles[0] == "Admin" || Roles[1] == "Admin" || Roles[2] == "Admin" || actionCode == "changeName" || actionCode == "changeGroup" || actionCode == "Exit" || actionCode == "Enter" {
		res = true
	}
	if (Roles[0] == "Teacher" || Roles[1] == "Teacher" || Roles[2] == "Teacher") && (actionCode != "toadmin" && actionCode != "chageRole" && actionCode != "changeSchedule" && actionCode != "sourceOfAutomaticUpdates" && actionCode != "frequencyUpdate") {
		res = true
	}
	if (Roles[0] == "Student" || Roles[1] == "Student" || Roles[2] == "Student") && (actionCode != "toadmin" && actionCode != "chageRole" && actionCode != "changeSchedule" && actionCode != "sourceOfAutomaticUpdates" && actionCode != "frequencyUpdate" && actionCode != "whereIsTheGroup" && actionCode != "leaveACommentOnTheNumPairForTheGroup") {
		res = true
	}

	if !res {
		w.Write([]byte("Ошибка кода действия!"))
		return
	}

	// Определяем время жизни JWT-токена
	tokenExpiresAt := time.Now().Add(time.Minute * time.Duration(3))

	// Заполняем данными полезную нагрузку
	payload := jwt.MapClaims{
		"expires_at":  tokenExpiresAt.Unix(),
		"action_code": strings.Split(actionCode, "@")[0],
		"name":        user.Data.Name,
		"group":       user.Data.Group,
		"Id_github":   user.GitHub_id,
		"id_telegram": user.Telegram_id,
		"role1":       user.Roles[0],
		"role2":       user.Roles[1],
		"role3":       user.Roles[2],
	}

	// Создаём токен с методом шифрования HS256
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	// Подписываем токен
	tokenString, err := token.SignedString([]byte(CLIENT_SECRET)) // Возможно, ключ придётся поменять
	if err != nil {
		w.Write([]byte("Ошибка в формировании токена"))
	}

	// Отсылаем токен
	w.Write([]byte(tokenString))
}

func aouth1Hendler(w http.ResponseWriter, r *http.Request) {
	// Очистка словаря AuthorizationRequests от мусора с истёкшим временем
	for _, user := range AuthanticateRequests {
		t := time.Now()
		if t.Sub(user.Time) <= 0 {
			delete(AuthanticateRequests, user.Telegram_id)
		}
	}

	userOauth.Telegram_id, _ = strconv.Atoi(r.URL.Query().Get("chat_id")) // От бота поступает Get-запрос!
	authorizationURL := "https://github.com/login/oauth/authorize?client_id=" + CLIENT_ID + "&state=" + strconv.Itoa(userOauth.Telegram_id)

	// Устанавливаем время ожидания входа в систему и заполняем другие поля структуры
	var UserAuthanticate Authanticate
	UserAuthanticate.Calldack = r.URL.Query().Get("callback")
	UserAuthanticate.Telegram_id = strconv.Itoa(userOauth.Telegram_id)
	UserAuthanticate.Time = time.Now().Add(30 * time.Second)

	AuthanticateRequests[UserAuthanticate.Telegram_id] = UserAuthanticate

	w.Write([]byte(authorizationURL))
}

func oauthHandler(w http.ResponseWriter, r *http.Request) {
	var responceHtml = "<html><body><h1>Вы аутентифецированы!</h1></body></html>"

	// Получаем chat_id из параметра state
	var state string = r.URL.Query().Get("state")

	// Находим пользователя в очереди на вход и берём его callback
	var callback string
	for chat_id, user := range AuthanticateRequests {
		if chat_id == state {
			callback = user.Calldack
		}
	}

	// Принимам code от GitHub
	code := r.URL.Query().Get("code")
	if code == "" {
		responceHtml = "<html><body><h1>Ошибка! Вы НЕ аутентифицированы!</h1></body></html>"
		fmt.Fprint(w, responceHtml)
		go BotNotificationError(state, callback, "203") // Ошибка получения кода доступа
		log.Println("Ошибка получения кода доступа!")
		delete(AuthanticateRequests, state)
		return
	}

	// Забираем токен доступа
	accessToken, err := getAccessToken(code) // accessToken пустой приходит
	if err != nil {
		go BotNotificationError(state, callback, "201") // Ошибка доступа
		log.Println("Ошибка доступа!")
		fmt.Fprint(w, "<html><body><h1>Ошибка!Вы НЕ аутентифицированы!</h1></body></html>")
		return
	}

	// Проработать ситуацию, когда пользователь не зарегистрирован на GitHub

	// Забираем данные о пользователе (GitHub_id и Name)
	userDataGitHub, err := getUserData(accessToken)
	if err != nil {
		go BotNotificationError(state, callback, "202") // Ошибка получения данных
		log.Println("Ошибка получения данных пользователя!")
		fmt.Fprint(w, "<html><body><h1>Ошибка!Вы НЕ аутентифицированы!</h1></body></html>")
		return
	}
	userOauth.GitHub_id = userDataGitHub.Id
	userOauth.Data.Name = userDataGitHub.Name

	// Проверяем зарегистрирован ли пользователь
	Users = NewUser(Users, userOauth)
	SafeData(Users)

	// Пересылаем модулю Бот GitHub_id и chat_id пользователя
	log.Println("BotNotification запущена")
	ok := BotNotification(state, strconv.Itoa(userOauth.GitHub_id), callback)
	log.Println("BotNotification отработала")

	// Уведомляем пользователя об успешном входе
	if ok {
		fmt.Fprint(w, responceHtml)
	} else {
		fmt.Fprint(w, "Вы НЕ авторизованы!")
	}
}

func ChangeRoles(w http.ResponseWriter, r *http.Request) {
	//GitHub_id int, Roles [3]string
	GitHub_id, err := strconv.Atoi(r.URL.Query().Get("GitHub_id"))

	// Роли передаём раздельно что бы было легче их сувать в массив из 3 элементов
	Roles := [3]string{r.URL.Query().Get("Role1"), r.URL.Query().Get("Role2"), r.URL.Query().Get("Role3")}

	if err != nil || len(Roles) != 3 {
		fmt.Println("Error")
		return
	}

	OurUser, ok := Users[GitHub_id] // Типа новая структура, но копия той что хотели получить

	if ok {
		OurUser.Roles = Roles
		Users[GitHub_id] = OurUser // Перезаписываем копию с изменениями
		SafeData(Users)
	}
}

//

// СЛУЖЕБНЫЕ ФУНКЦИИ ДЛЯ HANDLERS
func BotNotification(chat_id, GitHub_id, callback string) bool {
	// Передача chat_id и GitHub_id модулю Бот на callback

	cli := http.Client{}
	requestURL := callback + "?chat_id=" + chat_id + "&GitHub_id=" + GitHub_id + "&codeError=200"

	// Формируем запрос и отсылаем
	request, _ := http.NewRequest("GET", requestURL, nil)
	response, _ := cli.Do(request)

	for response == nil {
		response, _ = cli.Do(request)
		log.Println("Ожидание ответа от бота...")
		time.Sleep(1000)
	}

	// Забираем тело ответа
	resBody, _ := io.ReadAll(response.Body)
	log.Println("Тело ответа: " + string(resBody))

	defer response.Body.Close()

	if string(resBody) == "200" {
		delete(AuthanticateRequests, chat_id)
		log.Println("Авторизация завершена")
		return true
	}
	return false
}

func BotNotificationError(chat_id, callback, codeError string) {
	// Передача кода codeError модулю Бот на callback

	cli := http.Client{}
	requestURL := callback + "?codeError=" + codeError + "&chat_id=" + chat_id

	// Формируем запрос и отсылаем
	request, _ := http.NewRequest("GET", requestURL, nil)
	response, _ := cli.Do(request)

	for response == nil {
		response, _ = cli.Do(request)
		log.Println("Ожидание ответа от бота...")
		time.Sleep(1000)
	}

	// Забираем тело ответа
	resBody, _ := io.ReadAll(response.Body)
	defer response.Body.Close()

	if string(resBody) == "209" {
		delete(AuthanticateRequests, chat_id)
		log.Println("Ошибка входа отправлена успешно")
	}
}

func logging(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Получен запрос на " + r.URL.Path)
		next(w, r)
		log.Println("Отправлен ответ на " + r.URL.Path)
	}
}

func getAccessToken(code string) (string, error) {
	// Создаёт http-клиент с дефолтными настройками
	client := http.Client{}
	requestURL := "https://github.com/login/oauth/access_token"

	// Добавляем данные в виде формы
	form := url.Values{}
	form.Add("client_id", CLIENT_ID)
	form.Add("client_secret", CLIENT_SECRET)
	form.Add("code", code)

	// Готовим и отправляем запрос
	request, _ := http.NewRequest("POST", requestURL, strings.NewReader(form.Encode()))
	request.Header.Set("Accept", "application/json") // Просим прислать ответ в формате json
	responce, _ := client.Do(request)

	defer responce.Body.Close()

	// Достаём данные из тела овтета
	var responceJSON struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		Scope       string `json:"scope"`
	}
	json.NewDecoder(responce.Body).Decode(&responceJSON)

	fmt.Println(responceJSON.AccessToken)

	return responceJSON.AccessToken, nil
}

func getUserData(AccessToken string) (UserDataGitHub, error) {
	// Создаём http-клиент с дефольтными настройками
	client := http.Client{}
	requestURL := "https://api.github.com/user"

	var data UserDataGitHub

	// Готовим и отправляем запрос
	request, _ := http.NewRequest("GET", requestURL, nil)
	request.Header.Set("Authorization", "Bearer "+AccessToken)
	responce, err := client.Do(request)
	if err != nil {
		return data, err
	}

	defer responce.Body.Close()
	json.NewDecoder(responce.Body).Decode(&data)

	return data, nil
}

//

// РАБОТА СО СПИСКОМ ОТКРЫТЫХ СЕССИЙ
func GetData() map[int]User {

	fileData, err := os.ReadFile(DataBasePath)

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

	Users := make(map[int]User, len(strings.Split(Data, "\n"))-1)

	for _, UserInfo := range strings.Split(Data, "\n") {
		if len(UserInfo) > 6 {
			Info := strings.Split(UserInfo, " ")

			if len(Info) < 7 {
				continue
			}

			GitHub, err1 := strconv.Atoi(Info[0])
			Telegram, err2 := strconv.Atoi(Info[1])
			if err1 != nil || err2 != nil {
				fmt.Println("Error")
			}

			userData := UserData{
				Name:  strings.Replace(Info[5], "|", " ", -1),
				Group: strings.Replace(Info[6], "|", " ", -1),
			}

			SignedUser := User{
				GitHub_id:   GitHub,
				Telegram_id: Telegram,
				Roles:       [3]string{Info[2], Info[3], Info[4]},
				Data:        userData,
			}
			Users[GitHub] = SignedUser
		}
	}

	return Users
}

func NewUser(Users map[int]User, NewUser User) map[int]User {
	_, ok := Users[NewUser.GitHub_id]
	if ok {
		fmt.Println("Такой пользователь уже есть")
		return Users
	}

	NewUser.Roles = [3]string{"Student", "", ""}
	NewUser.Data.Name = "Фамилия Имя Отчество"
	NewUser.Data.Group = "ПИ-232"
	Users[NewUser.GitHub_id] = NewUser
	return Users
}

func SafeData(Users map[int]User) {
	var Data string
	for _, UserInfo := range Users {

		GitHub := strconv.Itoa(UserInfo.GitHub_id)
		Telegram := strconv.Itoa(UserInfo.Telegram_id)

		Data += GitHub + " " +
			Telegram + " " +
			UserInfo.Roles[0] + " " +
			UserInfo.Roles[1] + " " +
			UserInfo.Roles[2] + " " +
			strings.Replace(UserInfo.Data.Name, " ", "|", -1) + " " +
			strings.Replace(UserInfo.Data.Group, " ", "|", -1) + " \n"

	}

	file, err := os.Create(DataBasePath)

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

//

// ДЛЯ РАБОТЫ С АДМИНКОЙ //
func CheckAccessForAdmin(w http.ResponseWriter, r *http.Request) {
	GitHub_id, err := strconv.Atoi(r.FormValue("GitHub_id"))
	if err != nil {
		w.Write([]byte("401"))
		return
	}
	Roles := Users[GitHub_id].Roles
	flag := false
	for i := 3; i < 3; i++ {
		if Roles[i] == "Admin" {
			flag = true
		}
	}
	if flag {
		w.Write([]byte("200"))
	} else {
		w.Write([]byte("400"))
	}
}

func GetUsers(w http.ResponseWriter, r *http.Request) { // Функция для выдачи списка пользователей модулю администрирования
	JWTTokenString := r.URL.Query().Get("JWTToken") // Используем JWT токен что бы кто попало не получал доступ к данным пользователей
	JWTToken, err := jwt.Parse(JWTTokenString, func(token *jwt.Token) (interface{}, error) {
		return []byte(CLIENT_SECRET), nil
	})

	if err != nil {
		fmt.Println(err)
		return
	}

	TokenValues, ok := JWTToken.Claims.(jwt.MapClaims)
	expires_at := int64(TokenValues["expires_at"].(float64))
	if ok && JWTToken.Valid && expires_at-time.Now().Unix() > 0 && TokenValues["action_code"].(string) == "SeeUserInfo" { //проверяем JWT токен из модуля администрирования
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated) // Статус код 200
		var users []User
		for _, user := range Users {
			users = append(users, user)
		}
		fmt.Printf("%v", Users)
		json.NewEncoder(w).Encode(users)
	}
}

//
