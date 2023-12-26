#pragma once
#include <ctime>

const string WeekDay[] = { "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday" };

string WeekDayToEnglish(string Day) {
	if (Day == u8"Понедельник") {
		return "Monday";
	}
	else if (Day == u8"Вторник") {
		return "Tuesday";
	}
	else if (Day == u8"Среда") {
		return "Wednesday";
	}
	else if (Day == u8"Четверг") {
		return "Thursday";
	}
	else if (Day == u8"Пятница") {
		return "Friday";
	}
	else if (Day == u8"Суббота") {
		return "Saturday";
	}
	else if (Day == u8"Воскресенье") {
		return "Sunday";
	}
	return Day;
}

// Высчитываем чётная сейчас неделя или нечётная
string WhatWeek() { // Узнаём какая неделя
	string Week = "OddWeek";

	time_t now = time(nullptr);

	struct tm CurrentTime;
	localtime_s(&CurrentTime, &now);

	tm September10;
	September10.tm_year = 2023 - 1900;
	September10.tm_mon = 8;
	September10.tm_mday = 10;
	September10.tm_hour = 0;
	September10.tm_min = 0;
	September10.tm_sec = 0;
	mktime(&September10);

	if (((CurrentTime.tm_yday - September10.tm_yday) % 14) / 7) {
		Week = "EvenWeek";
	}

	return Week;
}

int NumberDayOfWeek() {
	time_t now = time(nullptr);

	struct tm CurrentTime;
	localtime_s(&CurrentTime, &now);

	return CurrentTime.tm_wday;
}

string WhatDay() {
	return WeekDay[NumberDayOfWeek()];
}

// Определить какая сейчас пара (по номеру)
string WhatLesson() {
	string num = "";

	time_t t = time(nullptr);

	struct tm now;
	localtime_s(&now, &t);

	int CurrentTime = now.tm_hour * 60 + now.tm_min;

	if (CurrentTime >= 8 and CurrentTime <= 9 * 60 + 30) {
		num = "1";
	}
	else if (CurrentTime > 9 * 60 + 30 and CurrentTime <= 11 * 60 + 20) {
		num = "2";
	}
	else if (CurrentTime > 11 * 60 + 20 and CurrentTime <= 13 * 60) {
		num = "3";
	}
	else if (CurrentTime > 13 * 60 and CurrentTime <= 14 * 60 + 50) {
		num = "4";
	}
	else if (CurrentTime > 14 * 60 + 50 and CurrentTime <= 16 * 60 + 30) {
		num = "5";
	}
	else if (CurrentTime > 16 * 60 + 30 and CurrentTime <= 18 * 60 + 10) {
		num = "6";
	}
	else if (CurrentTime > 18 * 60 + 10 and CurrentTime <= 19 * 60 + 50) {
		num = "7";
	}

	return num;
}