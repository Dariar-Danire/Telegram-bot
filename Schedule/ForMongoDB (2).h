#pragma once
#include "Time.h"

// ���������� ������ �� �������
std::vector<std::string> split(std::string string, const char separator) {
	const int n = std::count(string.begin(), string.end(), separator);
	std::vector<std::string> strs(n + 1);

	int pos = 0;
	for (int i = 0; i <= n; i++) {
		pos = string.find(separator);
		strs[i] = string.substr(0, pos);
		string = string.substr(pos + 1);
	}

	return strs;
}

// // // // // // // // // // // // // // // // // // // // // // // // // // // // // // // // // //
// ������� ���� �������� ����, ��� ������� ����� ����� ������ ����������, ��� ����� ������/������� //
// // // // // // // // // // // // // // // // // // // // // // // // // // // // // // // // // //

// ��������� ��� �������� ������ � ������ � ���������
// ��� ����� ������������ � ����������� ��������� ��� ���?

struct Group {
	string group = "";
	string subgroup = "";

	void setGroupSTR(string group_s) {
		auto pos = group_s.find('(');

		if (pos == string::npos) {
			this->group = group_s;
		}
		else {
			size_t Size = group_s.size();
			auto pos2 = group_s.find(')') != -1? group_s.find(')'): Size;

			group = group_s.substr(0, pos);
			subgroup = group_s.substr(pos + 1, Size - pos2);
		}
	}
};




























string ScheduleForDayForStudent(Group group, string Day) {
	setlocale(LC_ALL, ".UTF8");

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << u8"Document not found" << endl;
		//res.set_content("Error: Document not found", "text/plain");
		return "";
	}

	Day = Day == u8"Tomorrow" ? WeekDay[NumberDayOfWeek() + 1] : Day;
	Day = Day == u8"Today" ? WhatDay() : Day;

	string Subgroup;
	if (group.subgroup == "1") {
		Subgroup = "FirstSubgroup";
	}
	else if (group.subgroup == "2") {
		Subgroup = "SecondSubgroup";
	}

	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	string FormatedLesson = "";

	for (int i = 1; i <= 7; i++) {
		if ((*find_one_result).view()[group.group][WhatWeek()][Subgroup][Day][to_string(i)]) {
			cout << Doc[group.group][WhatWeek()][Subgroup][Day][to_string(i)] << endl;
			json Lesson = Doc[group.group][WhatWeek()][Subgroup][Day][to_string(i)];
			FormatedLesson += u8"����� ����: " + to_string(i) +
				u8"\n����: " + Lesson["Lesson"].dump() +
				u8"\n��� �������: " + Lesson["Type_of_lesson"].dump() +
				u8"\n�������������: " + Lesson["Teacher"].dump() +
				u8"\n���������: " + Lesson["Audience"].dump() +
				u8"\n�����������: " + Lesson["Commentary"].dump() + u8"\n\n\n";
		}
	}

	if (FormatedLesson != "") {
		cout << FormatedLesson << endl;
		return FormatedLesson;
	}
	else {
		cout << u8"��������" << endl;
		return u8"��������";
	}
}





string ScheduleForDayForTeacher(string Teacher, string Day) {
	setlocale(LC_ALL, "UTF8");

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << u8"Document not found" << endl;
		//res.set_content("Error: Document not found", "text/plain");
		return "";
	}

	Day = Day == u8"Tomorrow" ? WeekDay[NumberDayOfWeek() + 1] : Day;
	Day = Day == u8"Today" ? WhatDay() : Day;

	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	string FormatedLesson = "";

	for (int i = 1; i <= 7; i++) {
		auto it = Doc.begin();
		while (it != Doc.end()) {
			if ((*it)[WhatWeek()]["FirstSubgroup"][Day][to_string(i)]["Teacher"] == Teacher) {
				cout << (*it)[WhatWeek()]["FirstSubgroup"][Day][to_string(i)] << endl;
				json Lesson = (*it)[WhatWeek()]["FirstSubgroup"][Day][to_string(i)];
				FormatedLesson += u8"����� ����: " + to_string(i) +
					u8"\n����: " + Lesson["Lesson"].dump() +
					u8"\n��� �������: " + Lesson["Type_of_lesson"].dump() +
					u8"\n�������������: " + Lesson["Teacher"].dump() +
					u8"\n���������: " + Lesson["Audience"].dump() +
					u8"\n�����������: " + Lesson["Commentary"].dump() + u8"\n\n\n";
				break; // ������ ��� ������ 1 ���� � ������� ���� �� �����
			}
			if ((*it)[WhatWeek()]["SecondSubgroup"][Day][to_string(i)]["Teacher"] == Teacher) {
				cout << (*it)[WhatWeek()]["SecondSubgroup"][Day][to_string(i)] << endl;
				json Lesson = (*it)[WhatWeek()]["SecondSubgroup"][Day][to_string(i)];
				FormatedLesson += u8"����� ����: " + to_string(i) +
					u8"\n����: " + Lesson["Lesson"].dump() +
					u8"\n��� �������: " + Lesson["Type_of_lesson"].dump() +
					u8"\n�������������: " + Lesson["Teacher"].dump() +
					u8"\n���������: " + Lesson["Audience"].dump() +
					u8"\n�����������: " + Lesson["Commentary"].dump() + u8"\n\n\n";
				break; // ������ ��� ������ 1 ���� � ������� ���� �� �����
			}
			it++;
		}
	}

	if (FormatedLesson != "") {
		cout << FormatedLesson;
		return FormatedLesson;
	}
	else {
		cout << "��������" << endl;
		return u8"��������";
	}
}









string NextLessonForTeacher(string Teacher) {
	setlocale(LC_ALL, "ru");

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << "Document not found" << endl;
		//res.set_content("Error: Document not found", "text/plain");
		return "";
	}

	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	if (WhatLesson() == "7" or WhatLesson() == "") {
		cout << "���� ���." << endl;
		return u8"���� ���";
	}

	string LessonNumber = to_string(stoi(WhatLesson()) + 1);

	auto it = Doc.begin();
	while (it != Doc.end()) {
		cout << it.key() << endl;
		if ((*it)[WhatWeek()]["FirstSubgroup"][WhatDay()][LessonNumber]["Teacher"] == Teacher) {
			string Audience = (*it)[WhatWeek()]["FirstSubgroup"][WhatDay()][LessonNumber]["Audience"].dump();
			cout << Audience << endl;
			return Audience;
		}
		if ((*it)[WhatWeek()]["SecondSubgroup"][WhatDay()][LessonNumber]["Teacher"] == Teacher) {
			string Audience = (*it)[WhatWeek()]["SecondSubgroup"][WhatDay()][LessonNumber]["Audience"].dump();
			cout << Audience << endl;
			return Audience;
		}
		it++;
	}

	for (int i = stoi(WhatLesson()) + 2; i <= 7; i++) {
		it = Doc.begin();
		while (it != Doc.end()) {
			cout << it.key() << endl;
			if ((*it)[WhatWeek()]["FirstSubgroup"][WhatDay()][to_string(i)]["Teacher"] == Teacher) {
				string Audience = (*it)[WhatWeek()]["FirstSubgroup"][WhatDay()][to_string(i)]["Audience"].dump();
				cout << "��������� ���� ���, �� ���� " << i << " ���� � " << Audience << endl;
				return Audience;
			}
			if ((*it)[WhatWeek()]["SecondSubgroup"][WhatDay()][to_string(i)]["Teacher"] == Teacher) {
				string Audience = (*it)[WhatWeek()]["SecondSubgroup"][WhatDay()][to_string(i)]["Audience"].dump();
				cout << "��������� ���� ���, �� ���� " << i << " ���� � " << Audience << endl;
				return Audience;
			}
			it++;
		}
	}

	cout << "���� ���" << endl;
	return u8"���� ���";
}








string NextLessonForStudent(Group group) {
	setlocale(LC_ALL, ".UTF8");

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << "Document not found" << endl;
		//res.set_content("Error: Document not found", "text/plain");
		return "";
	}

	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	string Subgroup = "";
	if (group.subgroup == "1") {
		Subgroup = "FirstSubgroup";
	}
	else if (group.subgroup == "2") {
		Subgroup = "SecondSubgroup";
	}

	if (WhatLesson() == "7" or WhatLesson() == "") {
		cout << u8"���� ���." << endl;
		return u8"���� ���.";
	}

	string LessonNumber = to_string(stoi(WhatLesson()) + 1);

	if ((*find_one_result).view()[group.group][WhatWeek()][Subgroup][WhatDay()][LessonNumber]) {
		string Audience = Doc[group.group][WhatWeek()][Subgroup][WhatDay()][LessonNumber]["Audience"];
		cout << Audience << endl;
		return Audience;
	}

	for (int i = stoi(WhatLesson()) + 2; i <= 7; i++) {
		if ((*find_one_result).view()[group.group][WhatWeek()][Subgroup][WhatDay()][to_string(i)]) {
			string Audience = Doc[group.group][WhatWeek()][Subgroup][WhatDay()][to_string(i)]["Audience"];
			cout << Audience << endl;
			return u8"��������� ���� ���, �� ���� " + to_string(i) + u8" ���� � " + Audience;
		}
	}

	cout << "���� ���" << endl;
	return u8"���� ���";
}

string WhereGroup(string GroupName) {
	setlocale(LC_ALL, ".UTF8");

	Group group;
	group.setGroupSTR(GroupName);

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << "Document not found" << endl;
		//res.set_content("Error: Document not found", "text/plain");
		return "";
	}

	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	string Subgroup = "";
	if (group.subgroup == "1") {
		Subgroup = "FirstSubgroup";
	}
	else if (group.subgroup == "2") {
		Subgroup = "SecondSubgroup";
	}

	cout << group.group << endl;
	cout << group.subgroup << endl;

	cout << Doc[group.group][WhatWeek()][Subgroup][WhatDay()] << endl;
	if ((*find_one_result).view()[group.group][WhatWeek()][Subgroup][WhatDay()][WhatLesson()]) {
		string Audience = Doc[group.group][WhatWeek()][Subgroup][WhatDay()][WhatLesson()]["Audience"];
		cout << Audience << endl;
		return Audience;
	}

	cout << "Group not found" << endl;
	return u8"Group not found";
}

string WhereTeacher(string Teacher) {
	setlocale(LC_ALL, ".UTF8");

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << "Document not found" << endl;
		//res.set_content("Error: Document not found", "text/plain");
		return u8"";
	}

	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	auto it = Doc.begin();

	while (it != Doc.end()) {
		if ((*it)[WhatWeek()]["FirstSubgroup"][WhatDay()][WhatLesson()]["Teacher"] == Teacher) {
			string Audience = (*it)[WhatWeek()]["FirstSubgroup"][WhatDay()][WhatLesson()]["Audience"];
			cout << Audience << endl;
			return Audience;
		}
		if ((*it)[WhatWeek()]["SecondSubgroup"][WhatDay()][WhatLesson()]["Teacher"] == Teacher) {
			string Audience = (*it)[WhatWeek()]["SecondSubgroup"][WhatDay()][WhatLesson()]["Audience"];
			cout << Audience << endl;
			return Audience;
		}
		it++;
	}

	cout << "Teacher not found" << endl;
	return u8"Teacher not found";
}

string AddCommentary(Group group, string LessonNumber, string Commentary, string Day, string Teacher, string week) {
	setlocale(LC_ALL, ".UTF8");
	cout << Day << endl;

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		cout << u8"Not found document" << endl;
		return u8"Error";
	}

	string Subgroup = "";
	if (group.subgroup == "1") {
		Subgroup = "FirstSubgroup";
	}
	else if (group.subgroup == "2") {
		Subgroup = "SecondSubgroup";
	}
	string Json = bsoncxx::to_json((*find_one_result).view());
	json Doc{ json::parse(Json) };

	cout << Doc[group.group][week][Subgroup][Day] << "\t" << Teacher << endl;
	if (Doc[group.group][week][Subgroup][Day][LessonNumber]["Teacher"] == Teacher) {
		Doc[group.group][week][Subgroup][Day][LessonNumber]["Commentary"] = Commentary;

		auto NewDoc = bsoncxx::from_json(Doc.dump()); // ������ �������� �� Doc ���������������� � ������ json
		Collection.update_one((*find_one_result).view(), make_document(kvp("$set", NewDoc.view())).view());

		return u8"����������� ��������";
	}
	else {
		cout << u8"Error: ��� �� ���� ����" << endl;
		return u8"Error: ��� �� ���� ����";
	}
}







// ����� �������
string whenIsTheExam() { // ��������
	return u8"�������������� � ����������������: 15.01.2024";
}




































// ���������� � ����� ������� ������� �������������� ������
std::string getData(std::string action_code, std::string code_parameters, std::string name, Group group, std::string role) {
	setlocale(LC_ALL, "ru");

	std::string response = u8"��������� ������! ���������� ��� ��� �����, ����������.";
	if (action_code == "whereIsTheNextPair") {
		if (role == "Teacher") {
			response = NextLessonForTeacher(name);
		}
		else {
			response = NextLessonForStudent(group);
		}
	}
	else if (action_code == "whereIsTheGroup") {
		// ����� ����� ������ ���� �������� � group
		std::vector<std::string> search_group = split(code_parameters, '|');

		response = WhereGroup(search_group[0]);
	}
	else if (action_code == "whereIsTheTeacher") {
		// ����� ����� ������ ���� �������� � teacher_name
		std::vector<std::string> search_teacher = split(code_parameters, '|');

		response = WhereTeacher(search_teacher[0]);
	}
	else if (action_code == "scheduleFor") {
		// ����� ����� ������ ���� �������� � ���� ������ / ������ / ������� (���������� ��� weekDay)
		std::vector<std::string> weekDay = split(code_parameters, '|');

		if (role == "Teacher") {
			response = ScheduleForDayForTeacher(name, weekDay[0]);
		}
		else {
			response = ScheduleForDayForStudent(group, weekDay[0]);
		}
	}
	else if (action_code == "setComment") {
		// ����� 3 ��������� � ����� ����, ������, ���� ������ � ����� ������ �����������.
		std::vector<std::string> parameters = split(code_parameters, '|');

		if (parameters.size() < 4) {
			return u8"Error5! ������������ ����������!";
		}

		string num_pare = parameters[0];

		Group search_group;
		search_group.setGroupSTR(parameters[1]);

		string weekDay = parameters[2];

		string comment = parameters[3];

		string week = "";

		setlocale(LC_ALL, ".UTF8");
		cout << parameters[3] << endl;
		cout << parameters[4] << endl;
		if(parameters[4] == u8"��������") {
			week = "OddWeek";
		}
		else if(parameters[4] == u8"׸����"){
			week = "EvenWeek";
		}

		setlocale(LC_ALL, ".UTF8");

		response = AddCommentary(search_group, num_pare, comment, WeekDayToEnglish(weekDay), name, week);
	}
	else if (action_code == "whenIsTheExam") {
		response = whenIsTheExam();
	}

	// // // // // // // // // // // // // // // // // // // //
	// ����������� �� �������� � ������������ � action_code  //
	// // // // // // // // // // // // // // // // // // // //

	return response;
}