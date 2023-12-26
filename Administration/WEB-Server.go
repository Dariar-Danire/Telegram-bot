package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"

	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/xuri/excelize/v2"
)

const JWTSecretCode = "2dea7b2aff32ced454b3140fa3df5355755842b1"
const SymbolsForCode string = "1234567890abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const SymbolsForShortCode string = "1234567890abcdef"

// -------------------Для парсинга
type Lesson struct {
	Lesson         string
	Type_of_lesson string
	Teacher        string
	Audience       string
	Commentary     string
}

type Week struct {
	FirstSubgroup  map[string]map[string]Lesson
	SecondSubgroup map[string]map[string]Lesson
}

type Schedule struct {
	OddWeek  Week
	EvenWeek Week
}

func DayOfTheWeeks(Day string) string {
	Day = strings.ToLower(Day)
	if Day == "понедельник" {
		return "Monday"
	} else if Day == "вторник" {
		return "Tuesday"
	} else if Day == "среда" {
		return "Wednesday"
	} else if Day == "четверг" {
		return "Thursday"
	} else if Day == "пятница" {
		return "Friday"
	} else if Day == "суббота" {
		return "Saturday"
	} else if Day == "воскресенье" {
		return "Sunday"
	}
	return Day
}

func ScheduleParse(FileName string) string { // Парсим файл
	Groups := make(map[string]Schedule)

	File, err := excelize.OpenFile(FileName)

	if err != nil {
		fmt.Println(err)
		return ""
	}

	rows, err := File.GetRows("курс 1 ПИ")
	if err != nil {
		fmt.Println(err)
		return ""
	}

	Day := ""
	GroupRow := -1 // Что бы знать когда группы уже появились в списках

	for i := 0; i < len(rows); i++ {
		row := rows[i]
		if len(row) <= 1 {
			continue
		}

		if row[0] == "Дни недели" {
			for j, v := range row {
				if strings.Contains(v, "вид занятий") {
					schedule := Schedule{}
					schedule.OddWeek.FirstSubgroup = make(map[string]map[string]Lesson) // Создаём map, где храним день под его названием
					schedule.OddWeek.SecondSubgroup = make(map[string]map[string]Lesson)
					schedule.EvenWeek.FirstSubgroup = make(map[string]map[string]Lesson)
					schedule.EvenWeek.SecondSubgroup = make(map[string]map[string]Lesson)
					Groups[row[j+1]] = schedule
				}
			}
			GroupRow = i
			continue
		}

		if row[0] != "" && GroupRow != -1 {
			Day = DayOfTheWeeks(row[0])

			for _, v := range Groups {
				v.OddWeek.FirstSubgroup[Day] = make(map[string]Lesson) // Создаём map, где храним пару под её номером
				v.OddWeek.SecondSubgroup[Day] = make(map[string]Lesson)
				v.EvenWeek.FirstSubgroup[Day] = make(map[string]Lesson)
				v.EvenWeek.SecondSubgroup[Day] = make(map[string]Lesson)
			}
		}

		if row[1] != "" && GroupRow != -1 {

			for j := 0; j < len(row); j++ {
				if len(rows[GroupRow]) <= j {
					break
				}
				groupSchedule, ok := Groups[rows[GroupRow][j]]
				if ok {
					week := groupSchedule.OddWeek
					if j > len(rows[0]) { //
						week = groupSchedule.EvenWeek
					}
					if row[j] != "" {
						lesson := Lesson{
							Lesson:         row[j],
							Type_of_lesson: rows[i][j-1],
							Teacher:        rows[i+1][j],
							Audience:       rows[i+2][j],
						}
						week.FirstSubgroup[Day][row[1]] = lesson
					}
					j1 := j + 1 // Оптимизация
					if len(row) > j1 && row[j1] != "" {
						lesson := Lesson{
							Lesson:         row[j1],
							Type_of_lesson: rows[i][j-1],
							Teacher:        rows[i+1][j1],
							Audience:       rows[i+2][j1],
						}
						week.SecondSubgroup[Day][row[1]] = lesson
					}
				}
			}
			i += 4
		}
	}
	schedule, err := json.Marshal(Groups)
	if err != nil {
		panic(err)
	}
	return string(schedule)
	//fmt.Println(string(schedule))
}

var GoroutineId uint64 = 0

func ScheduleUpdater(Id uint64, Link string, Time int64) { // Обновляем расписание из источника с нужным периодом
	if Link == "" {
		return
	}

	for {
		if Id != GoroutineId {
			return
		}

		_, err := url.Parse(Link)
		if err != nil {
			fmt.Println(err)
			return
		}

		File, err := os.Create(fmt.Sprintf("Schedule%d.xlsx", time.Now().Unix()))
		if err != nil {
			fmt.Println(err)
			return
		}

		client := http.Client{
			CheckRedirect: func(r *http.Request, via []*http.Request) error {
				r.URL.Opaque = r.URL.Path
				return nil
			},
		}

		resp, err := client.Get(Link)
		if err != nil {
			fmt.Println(err)
			return
		}
		defer resp.Body.Close()

		_, err = io.Copy(File, resp.Body)
		if err != nil {
			fmt.Println(err)
			return
		}

		JsonSchedule := ScheduleParse(File.Name())

		File.Close()
		if err = os.Remove(File.Name()); err != nil {
			fmt.Println(err)
			return
		}

		fmt.Println(Time)
		//fmt.Println(JsonSchedule)

		var Client = http.Client{}

		form := url.Values{}
		form.Add("schedule_json", JsonSchedule)

		request, _ := http.NewRequest("POST", "http://localhost:8089/UpdateSchedule", strings.NewReader(form.Encode()))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		response, _ := Client.Do(request)

		// Читаем тело ответа
		resBody, _ := io.ReadAll(response.Body)
		fmt.Println(string(resBody))
		if string(resBody) != "200" {
			fmt.Println("Schedule update error")
		}
		response.Body.Close() // Закрываем соединение с сервером

		time.Sleep(time.Duration(Time) * time.Second)
	}
}

//--------------------

type UserData struct {
	Name  string
	Group string
}

type User struct {
	GitHub_id   int
	Telegram_id int
	Id          string // Нужно только для построения списка участников на сайте. Значение = #A[GitHub_id]
	GH_Id       string // Нужно только для построения списка участников на сайте. Значение = A[GitHub_id]
	Roles       [3]string
	Data        UserData
}

type SessionData struct {
	GitHub_id  string
	expires_at int64
}

type InactiveSessionData struct {
	GitHub_id  string
	expires_at int64
}

// var Users map[int]User                              //список id гитхаба пользователей
var UsersGitHub_id map[string]int                                                          //список пользователей
var InactiveSessions map[string]InactiveSessionData = make(map[string]InactiveSessionData) //список сессий которые не подтвердили
var Sessions map[string]SessionData = make(map[string]SessionData)                         //список сессий

func ExpiredSessionsCollector() { // Сборщик дохлых сессий
	// Проверяем обычные сессии
	for {
		for i, Session := range Sessions {
			if Session.expires_at-time.Now().Unix() <= 0 { //Если время действия токена вышло
				delete(Sessions, i)
			}
		}

		// Проверяем не подтверждёные сессии
		for i, InactiveSession := range InactiveSessions {
			if InactiveSession.expires_at-time.Now().Unix() <= 0 { //Если время действия токена вышло
				delete(InactiveSessions, i)
			}
		}

		time.Sleep(time.Minute) // Ждём 1 минуту
	}
}

func CheckRoles(Roles []string) bool { // Проверяем наличие админки
	flag := false
	for i := 0; i < 3; i++ {
		if Roles[i] == "Administrator" {
			flag = true
		}
	}
	return flag
}

func GenerateRandomToken(str string) string { //генерируем случайный токен с id гитхаба в начале
	Token := str //в начале токена будет id гитхаба пользователя что бы уменьшить шанс генерации 2 одинаковых токенов
	for i := 0; i < (rand.Int()%10)+40; i++ {
		Token += string(SymbolsForCode[rand.Int()%len(SymbolsForCode)]) //для защиты токена к имени прибавляем строку из 40-50 случайных символов
	}
	return Token
}

func GenerateShortRandomToken() string { //для тех, кто зашёл с браузера
	Token := "" //в начале токена будет id гитхаба пользователя что бы уменьшить шанс генерации 2 одинаковых токенов
	for i := 0; i < 4; i++ {
		Token += string(SymbolsForShortCode[rand.Int()%len(SymbolsForShortCode)]) //для защиты токена к имени прибавляем строку из 40-50 случайных символов
	}
	return Token
}

func isSessionExist(req *http.Request) bool { // Проверяем наличие сессии
	Cookie, err := req.Cookie("User_Cookie")

	if errors.Is(err, http.ErrNoCookie) { //проверяем есть ли у нас куки
		fmt.Println("Нет куки")
		return false
	}

	Session, ok := Sessions[Cookie.Value]

	//fmt.Printf("\n%v\n%v\n\n", Cookie.Value, Sessions)
	if !ok { //проверяем, есть ли у нас такая сессия
		fmt.Println("Нет сессии")
		return false
	}

	if Session.expires_at-time.Now().Unix() <= 0 { //Если время действия сессии вышло
		fmt.Println("Вышло время сессии")
		delete(Sessions, Cookie.Value)
		return false
	}

	return true
}

func LoginCheck(NextFunc http.HandlerFunc) http.HandlerFunc { // Преверяем зарегистрирован ли пользователь
	return func(w http.ResponseWriter, req *http.Request) {
		if !isSessionExist(req) && req.URL.Path != "/" { //если нет сессии и мы не на главной странице
			http.Redirect(w, req, "/", http.StatusSeeOther) //перенаправляем на главную страницу
			return                                          // обрываем функцию
		}
		NextFunc(w, req)
	}
}

func CheckAccessForAdmin(NextFunc http.HandlerFunc) http.HandlerFunc { // Проверяем права админа
	return func(w http.ResponseWriter, req *http.Request) {
		// Получаем печеньку
		Cookie, _ := req.Cookie("User_Cookie")
		GitHub_id := Sessions[Cookie.Value].GitHub_id // С помощью печеньки получаем GitHub_id пользователя

		// Создаём клиент
		client := http.Client{}

		// Формируем строку запроса

		requestURL := fmt.Sprintf("http://localhost:8080/CheckAccessForAdmin?GitHub_id=%s", GitHub_id)

		// Устанавливаем заголовок, говорящий, что тело запроса - это Формы с данными
		// Отправляем запрос и получаем ответ
		request, _ := http.NewRequest("GET", requestURL, nil)
		response, _ := client.Do(request)

		resBody, _ := io.ReadAll(response.Body) // Получаем тело ответа
		defer response.Body.Close()             // Закрывает соединение с сервером

		if string(resBody) == "200" {
			NextFunc(w, req)
		} else {
			http.Redirect(w, req, "/", http.StatusSeeOther) //перенаправляем на главную страницу
		}
	}
}

func FromTheInterfaceToAnArrayOfStrings(i []interface{}) []string {
	resArray := make([]string, len(i))

	for k, role := range i {
		resArray[k] = role.(string)
	}

	return resArray
}

func StartSession(w http.ResponseWriter, req *http.Request) { //начинаем не подтверждённую сессию по просьбе бота
	JWTTokenString := req.URL.Query().Get("JWTToken")
	JWTToken, err := jwt.Parse(JWTTokenString, func(token *jwt.Token) (interface{}, error) { //!!!!!!!!!!!!!!!!!!!!!!
		return []byte(JWTSecretCode), nil
	})

	TokenValues, ok := JWTToken.Claims.(jwt.MapClaims)                 // Достаём данные из токена
	expires_at := int64(TokenValues["expires_at"].(float64))           //получаем срок годности в Unix
	GitHub_id := strconv.Itoa(int(TokenValues["Id_github"].(float64))) //получаем гитхаб id пользователя

	if ok && JWTToken.Valid { //проверяем JWT токен

		checkRoles_result := CheckRoles([]string{TokenValues["role1"].(string), TokenValues["role2"].(string), TokenValues["role3"].(string)})
		if (expires_at-time.Now().Unix() > 0) && checkRoles_result { //проверяем срок действия и право
			NewSession := InactiveSessionData{
				GitHub_id:  GitHub_id,
				expires_at: time.Now().Unix() + int64(60),
			}

			RandToken := GenerateRandomToken(GitHub_id)

			InactiveSessions[RandToken] = NewSession

			fmt.Println("Сессия начата")

			w.Write([]byte("http://localhost:8083/?code=" + RandToken))

		} else {
			fmt.Println("Время действия токена истекло или у вас недостаточно прав")
		}
	} else {
		fmt.Println(err)
	}
}

func AccessUser(w http.ResponseWriter, req *http.Request) { // Бот подтверждает пользователя
	JWTTokenString := req.URL.Query().Get("JWTToken")
	Code := req.URL.Query().Get("Code")
	JWTToken, err := jwt.Parse(JWTTokenString, func(token *jwt.Token) (interface{}, error) { //!!!!!!!!!!!!!!!!!!!!!!
		return []byte(JWTSecretCode), nil
	})

	TokenValues, ok := JWTToken.Claims.(jwt.MapClaims)                 // Достаём данные из токена
	expires_at := int64(TokenValues["expires_at"].(float64))           //получаем срок годности в Unix
	GitHub_id := strconv.Itoa(int(TokenValues["Id_github"].(float64))) //получаем гитхаб id пользователя
	if ok && JWTToken.Valid && (expires_at-time.Now().Unix() > 0) {    //проверяем JWT токен

		InactiveSession, SessionIsOk := InactiveSessions[Code]

		if SessionIsOk && (InactiveSession.expires_at-time.Now().Unix() > 0) { //проверяем срок действия и право
			//обновляем таймер и добавляем id
			InactiveSession.GitHub_id = GitHub_id
			InactiveSession.expires_at = time.Now().Unix() + int64(120)
			InactiveSessions[Code] = InactiveSession
			fmt.Println("Подтвердили пользователя")
			w.Write([]byte("200"))

		} else {
			w.Write([]byte("400"))
		}
	} else {
		w.Write([]byte("400"))
		fmt.Println(err)
	}
}

func LogOut(w http.ResponseWriter, req *http.Request) { // Пользователь выходит из сессии
	http.Redirect(w, req, "http://localhost:8083/?code=!Delete", http.StatusSeeOther)
}

func Home(w http.ResponseWriter, req *http.Request) { // Главная страница

	if req.URL.Query().Get("code") == "!Delete" {
		Cookie, _ := req.Cookie("User_Cookie")
		CookieValue := Cookie.Value

		EatenCookie := http.Cookie{
			Name:   "User_Cookie",
			Value:  "",
			MaxAge: 3600,
		}

		delete(Sessions, CookieValue) // Удаляем сессию из Sessions

		http.SetCookie(w, &EatenCookie) // Убираем куки
	}

	InactiveSession, ok := InactiveSessions[req.URL.Query().Get("code")] //пытаемся получить неактивированную сессию и тем самым проверяем код доступа

	if InactiveSession.expires_at-time.Now().Unix() <= 0 { //Если время действия токена вышло
		ok = false
		delete(InactiveSessions, req.URL.Query().Get("code"))
	}

	if isSessionExist(req) { //если у нас есть сессия

		if ok { //если у нас есть правильный код доступа
			// заменяем старую сессию на новую
			GitHub_id := InactiveSession.GitHub_id
			OldCookie, _ := req.Cookie("User_Cookie")
			OldCookieValue := OldCookie.Value

			NewToken := GenerateRandomToken(GitHub_id)

			NewCookie := http.Cookie{
				Name:     "User_Cookie",
				Value:    NewToken,
				Path:     "/",
				MaxAge:   3600,
				HttpOnly: true,
				Secure:   true,
				SameSite: http.SameSiteLaxMode,
			}

			NewSession := SessionData{
				GitHub_id:  GitHub_id,
				expires_at: time.Now().Unix() + int64(NewCookie.MaxAge),
			}

			delete(Sessions, OldCookieValue)                      // Удаляем старую сессию из Sessions
			delete(InactiveSessions, req.URL.Query().Get("code")) // Удаляем не подтверждённую сессию сессию из Sessions

			Sessions[NewToken] = NewSession //Добавляем новую сессию в список

			http.SetCookie(w, &NewCookie)
		}
		//далее просто подгружаем страницу
		tmpl, err := template.ParseFiles("Templates/index.html")
		if err != nil {
			w.Write([]byte(err.Error()))
		}
		tmpl.ExecuteTemplate(w, "index", nil)

	} else { //если у нас нет сессии

		if ok { //если у нас есть правильный код доступа
			if InactiveSession.GitHub_id != "" {
				//если код доступа верный, то делаем подтверждённую сессию
				GitHub_id := InactiveSession.GitHub_id

				RandToken := GenerateRandomToken(GitHub_id)

				NewCookie := http.Cookie{
					Name:     "User_Cookie",
					Value:    RandToken,
					Path:     "/",
					MaxAge:   3600,
					HttpOnly: true,
					Secure:   true,
					SameSite: http.SameSiteLaxMode,
				}

				Session := SessionData{
					GitHub_id:  GitHub_id,
					expires_at: time.Now().Unix() + int64(NewCookie.MaxAge),
				}

				delete(InactiveSessions, req.URL.Query().Get("code")) // Удаляем не подтверждённую сессию сессию из Sessions

				Sessions[RandToken] = Session //Добавляем новую сессию в список

				http.SetCookie(w, &NewCookie)

				//далее просто подгружаем страницу
				tmpl, err := template.ParseFiles("Templates/index.html")
				if err != nil {
					w.Write([]byte(err.Error()))
				}
				tmpl.ExecuteTemplate(w, "index", nil)
			} else { //если у нашей сессии нет id гитхаба(то есть пользователь зашёл через браузер)
				// обновляем таймер в сессии без id и высвечиваем ему его код

				InactiveSession.expires_at = time.Now().Unix() + int64(120)
				InactiveSessions[req.URL.Query().Get("code")] = InactiveSession

				//далее просто подгружаем страницу
				tmpl, err := template.ParseFiles("Templates/Login.html")
				if err != nil {
					w.Write([]byte(err.Error()))
				}

				URl := fmt.Sprintf("http://localhost:8083/?code=%s", req.URL.Query().Get("code"))

				code := struct { //создаём анонимную струткуру для того что бы отобразить код и ссылку с ним на странице
					Code, URL string
				}{
					Code: req.URL.Query().Get("code"),
					URL:  URl,
				}

				tmpl.ExecuteTemplate(w, "Login", code)
			}
		} else { //если у нас нет куки с сессией и нет кода доступа или он не верный
			//создаём не подтверждённую сессию(с коротким ключом) без гитхаб id и перенаправляем на ту же страницу но уж с адресом в параметрах
			RandToken := GenerateShortRandomToken()

			Session := InactiveSessionData{ //создаём не подтверждённую сессию
				GitHub_id:  "",
				expires_at: time.Now().Unix() + int64(120),
			}

			InactiveSessions[RandToken] = Session //Добавляем новую сессию в список
			ReqURl := fmt.Sprintf("/?code=%s", RandToken)

			http.Redirect(w, req, ReqURl, http.StatusSeeOther)
		}

	}
}

func ScheduleHandle(w http.ResponseWriter, req *http.Request) { // Страница с расписание
	tmpl, err := template.ParseFiles("Templates/ScheduleHandle.html")
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	tmpl.ExecuteTemplate(w, "ScheduleHandle", nil)
}

func RolesHandle(w http.ResponseWriter, req *http.Request) { // Страница с пользователями
	tmpl, err := template.ParseFiles("Templates/UsersHandle.html")
	if err != nil {
		w.Write([]byte(err.Error()))
	}

	payload := jwt.MapClaims{ // Заполняем данными полезную нагрузку
		"expires_at":  time.Now().Add(time.Second * time.Duration(4)).Unix(), // Определяем время жизни JWT-токена
		"action_code": "SeeUserInfo",                                         // Действие которое хотим выполнить
	}

	Token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload) // Создаём токен с методом шифрования HS256

	TokenString, err := Token.SignedString([]byte(JWTSecretCode)) // Подписываем токен
	if err != nil {                                               // Обрабатываем ошибку
		w.Write([]byte("Ошибка в формировании токена"))
	}

	var Client = http.Client{}

	res, err := Client.Get(fmt.Sprintf("http://localhost:8080/GetUsers?JWTToken=%s", TokenString))
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	var Users []User
	json.NewDecoder(res.Body).Decode(&Users)
	for i, v := range Users {
		v.Id = "#A" + strconv.Itoa(v.GitHub_id)
		v.GH_Id = "A" + strconv.Itoa(v.GitHub_id)
		Users[i] = v
	}

	tmpl.ExecuteTemplate(w, "UsersHandle", Users)
}

func UpdateSchedule(w http.ResponseWriter, req *http.Request) { // обновляем расписание
	if req.Method != "POST" {
		return
	}

	File, _, err := req.FormFile("File") // Получаем файл

	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	defer File.Close()

	NewFile, err := os.Create(fmt.Sprintf("Schedule%d.xlsx", time.Now().Unix()))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.Copy(NewFile, File)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	JsonSchedule := ScheduleParse(NewFile.Name())

	NewFile.Close()
	if err = os.Remove(NewFile.Name()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var Client = http.Client{}

	form := url.Values{}
	form.Add("schedule_json", JsonSchedule)

	request, _ := http.NewRequest("POST", "http://localhost:8089/UpdateSchedule", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, _ := Client.Do(request)

	// Читаем тело ответа
	resBody, _ := io.ReadAll(response.Body)
	fmt.Println(string(resBody))
	if string(resBody) != "200" {
		fmt.Println("Schedule update error")
	}
	defer response.Body.Close() // Закрываем соединение с сервером

	http.Redirect(w, req, "/ScheduleHandle/", http.StatusSeeOther)
}

func UpdatePeriod(w http.ResponseWriter, req *http.Request) { // бновляем период обновления расписания из источника
	if req.Method != "POST" {
		return
	}

	Link := req.FormValue("Link")
	Time := req.FormValue("Time")

	//Link = "https://github.com/batareika4/File/raw/main/Schedule.xlsx"

	PeriodOfTime, err := strconv.Atoi(Time)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if GoroutineId == 18446744073709551615 {
		GoroutineId = 0
	} else {
		GoroutineId = (GoroutineId + 1)
	}
	go ScheduleUpdater(GoroutineId, Link, int64(PeriodOfTime))

	http.Redirect(w, req, "/ScheduleHandle/", http.StatusSeeOther)
}

func ChangeUserRoles(w http.ResponseWriter, req *http.Request) { // Меняем роли пользователя
	GitHub_id := req.FormValue("GitHub_id")
	Role1 := req.FormValue("Student")
	Role2 := req.FormValue("Teacher")
	Role3 := req.FormValue("Administrator")

	var netClient = http.Client{}

	// Роли передаём раздельно что бы было легче их сувать в массив из 3 элементов
	res, err := netClient.Get(fmt.Sprintf("http://localhost:8080/ChangeRoles?GitHub_id=%s&Role1=%s&Role2=%s&Role3=%s", GitHub_id, Role1, Role2, Role3))
	if err != nil {
		panic(err)
	}
	defer res.Body.Close()

	http.Redirect(w, req, "/RolesHandle/", http.StatusSeeOther)
}

func main() {
	go ExpiredSessionsCollector()
	Styles := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", Styles))
	http.HandleFunc("/", Home)
	http.HandleFunc("/StartSession/", StartSession)
	http.HandleFunc("/AccessUser/", AccessUser)
	http.HandleFunc("/LogOut/", LoginCheck(LogOut))
	http.HandleFunc("/ScheduleHandle/", LoginCheck(CheckAccessForAdmin(ScheduleHandle)))
	http.HandleFunc("/RolesHandle/", LoginCheck(CheckAccessForAdmin(RolesHandle)))
	http.HandleFunc("/UpdateSchedule/", LoginCheck(CheckAccessForAdmin(UpdateSchedule)))
	http.HandleFunc("/UpdatePeriod/", LoginCheck(CheckAccessForAdmin(UpdatePeriod)))
	http.HandleFunc("/ChangeUserRoles/", LoginCheck(CheckAccessForAdmin(ChangeUserRoles)))
	http.ListenAndServe(":8083", nil)
}
