#include <iostream>
#include <string>
#include <vector>


#include <codecvt>
#include <cstdint>
#include <locale>


#include <jwt-cpp/jwt.h>
#include <chrono>

#define CPPHTTPLIB_FORM_URL_ENCODED_PAYLOAD_MAX_LENGTH 1024 * 1024 * 255
#include "include/json.hpp"
#include "include/httplib.h"

#include <bsoncxx/json.hpp>
#include <mongocxx/client.hpp>
#include <mongocxx/instance.hpp>

#include <bsoncxx/builder/basic/document.hpp>
#include <mongocxx/stdx.hpp>
#include <mongocxx/uri.hpp>

using namespace httplib;
using namespace std;
using json = nlohmann::json;
using bsoncxx::builder::basic::kvp;
using bsoncxx::builder::basic::make_array;
using bsoncxx::builder::basic::make_document;

const string SECRET = "2dea7b2aff32ced454b3140fa3df5355755842b1";
const auto uri = mongocxx::uri{ "mongodb+srv://mongo_user:LXkEj4dJLpY19PT0@cluster0.6yzdyrb.mongodb.net/" };

#include "ForMongoDB.h"



struct JWT_token_p {
	int expires_at;
	std::string name;
	std::string action_code;
	Group group;
	std::string roles[3];

	void setGroupSTR(std::string group_s) {
		group.setGroupSTR(group_s);
	}
};

//                                                   (                             )
void getScheduleForTheUser(const httplib::Request& req, httplib::Response& res) {
	setlocale(LC_ALL, "Russian");

	std::string action_code = req.has_param("actionCode") ? req.get_param_value("actionCode") : "";
	std::string JWT_token = req.has_param("JWTtoken") ? req.get_param_value("JWTtoken") : "";

	if (action_code == "" || JWT_token == "") {
		res.set_content(u8"                          !", "text/plain");
		return;
	}

	//                action_code             
	std::vector<std::string> code_params = split(action_code, '@');
	action_code = code_params[0];
	std::string parameters = code_params[1];
	while (parameters.find("/!") !=  -1) {
		parameters.replace(parameters.find("/!"), 2, " ");
	}

	//        JWY-     
	auto decoded_token = jwt::decode(JWT_token);
	JWT_token_p token;

	try {
		//              -            
		auto verifier = jwt::verify().allow_algorithm(jwt::algorithm::hs256{ SECRET });
		//                
		verifier.verify(decoded_token);

		//                    ,            .             catch
		auto payload = decoded_token.get_payload_claims();

		token.expires_at = payload["expires_at"].as_int();
		token.action_code = payload["action_code"].as_string();
		token.setGroupSTR(payload["group"].as_string());
		token.name = payload["name"].as_string();
		token.roles[0] = payload["role1"].as_string();
		token.roles[1] = payload["role2"].as_string();
		token.roles[2] = payload["role3"].as_string();
	}
	catch (...) {
		std::cout << "                           !" << std::endl;
		res.set_content("Error400", "text/plain");
		return;
	}

	if (token.expires_at < std::chrono::seconds(std::time(NULL)).count()) {
		std::cout << "                          !" << std::endl;
		res.set_content("Error401.", "text/plain");
		return;
	}
	if (token.action_code != action_code) {
		std::cout << "                                                      !" << std::endl;
		res.set_content("Error402", "text/plain");
		return;
	}
	if (token.group.group == "" || token.group.subgroup == "") {
		std::cout << "     \"      \"             !" << std::endl;
		res.set_content("     \"      \"             !", "text/plain");
		return;
	}
	if (token.roles[0] != "Student" && token.roles[1] != "Teacher") {
		std::cout << "  " << token.name << "                !" << std::endl;
		res.set_content("                     !", "text/plain");
		return;
	}

	string roleDB;
	if (token.roles[1] == "Teacher") {
		roleDB = token.roles[1];
	}
	else if (token.roles[0] == "Student") {
		roleDB = token.roles[0];
	}

	string  response = getData(token.action_code, parameters, token.name, token.group, roleDB);

	res.set_content(response, "text/plain"); //                          ,                                 .
}



//                                                             
void UpdateSchedule(const Request& req, Response& res) {

	string NewSchedule = req.has_param("schedule_json") ? req.get_param_value("schedule_json") : "";

	if (NewSchedule == "") {
		cout << "Empty Schedule\n";
		res.set_content("401", "text/plain");
	}

	mongocxx::client connection{ uri };
	auto BD = connection["Project"];
	auto Collection = BD["Schedule"];

	auto Doc = bsoncxx::from_json(NewSchedule);

	auto find_one_result = Collection.find_one({});
	if (not find_one_result) {
		return;
	}

	Collection.update_one((*find_one_result).view(), make_document(kvp("$set", Doc.view())).view());

	res.set_content("200", "text/plain");
}

//               
int main() {
	mongocxx::instance inst{};

	Server server;
	server.set_payload_max_length(1000000); //            json                  

	server.Get("/getSchedule", getScheduleForTheUser); //                                        (    )
	//server.Get("/ScheduleForTomorrowForTeacher", ScheduleForTomorrowForTeacher);
	//server.Get("/NextLessonForTeacher", NextLessonForTeacher);
	//server.Get("/NextLessonForStudent", NextLessonForStudent);
	//server.Get("/WhereGroup", WhereGroup); //                  
	//server.Get("/WhereTeacher", WhereTeacher); //                   
	//server.Get("/AddCommentary", AddCommentary); //                           
	server.Post("/UpdateSchedule", UpdateSchedule); //                          

	server.listen("0.0.0.0", 8089);
}
