package main

import (
	"fmt"
	"net/http"
	"database/sql"
	_"github.com/lib/pq"
	"github.com/zenazn/goji"
	"github.com/zenazn/goji/web"
	"html/template"
	"encoding/json"
	"os"
	"encoding/base64"
	"time"
	"strings"
	"bytes"
)

type Configuration struct {
	DbName string
	UserName string
}

// db connections
var db *sql.DB
func setupDB() *sql.DB {
	file, _ := os.Open("./config/configuration.json")
	decoder := json.NewDecoder(file)
	configuration := Configuration{}
	decoder.Decode(&configuration)

	userName := configuration.UserName
	dbName := configuration.DbName

	dbinfo := fmt.Sprintf("user=%s dbname=%s sslmode=disable",
		userName, dbName)
	db, err := sql.Open("postgres", dbinfo)
	checkErr(err)
	return db
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

// get questions information
type questionsInformation struct {
	Id string
	Description string
	Sequence string
	Flag int
	Deleted *time.Time
}

// link with child structure
type getAllQuestionsInfo struct {
	QuestionsInfo []questionsInformation
}

// get question information from database using qId
func getQuestionInfoHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	qId := r.FormValue("id")
	stmt1, _ := db.Prepare("select id, description, sequence from questions where id = ($1)")
		rows1, _ := stmt1.Query(qId)
		questions := []questionsInformation{}
		q := questionsInformation{}
		for rows1.Next() {
			err := rows1.Scan(&q.Id, &q.Description, &q.Sequence)
			questions = append(questions, q)
			checkErr(err)
		}
	b, err := json.Marshal(questions)
	if err != nil {
			fmt.Printf("Error: %s", err)
			return;
	}
	//==========================
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(b))//set response...
	fmt.Println(questions)

}

// display only active questions in view...
func allQuestionsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	query :="select id, description, deleted, sequence from questions  order by sequence"
	rows3, _ := db.Query(query)

	getAllQuestionsInfo := getAllQuestionsInfo{}
	questionsInfo := []questionsInformation{}
	q := questionsInformation{}
	for rows3.Next() {
		err := rows3.Scan(&q.Id, &q.Description, &q.Deleted, &q.Sequence)
		if q.Deleted != nil{
			q.Flag = 1
		} else if q.Deleted == nil{
			q.Flag = 0
		}
		questionsInfo = append(questionsInfo, q)
		checkErr(err)
	}
	getAllQuestionsInfo.QuestionsInfo = questionsInfo

		t, _ := template.ParseFiles("./views/questions.html")
		t.Execute(w, getAllQuestionsInfo)
}

// display all questions in view...
func questionsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	query := "select id, description, deleted, sequence from questions where deleted is null order by sequence"
	rows3, _ := db.Query(query)

	getAllQuestionsInfo := getAllQuestionsInfo{}
	questionsInfo := []questionsInformation{}
	q := questionsInformation{}
	for rows3.Next() {
		err := rows3.Scan(&q.Id, &q.Description, &q.Deleted, &q.Sequence)
		if q.Deleted != nil{
			q.Flag = 1
		} else if q.Deleted == nil{
			q.Flag = 0
		}
		questionsInfo = append(questionsInfo, q)
		checkErr(err)
	}
	getAllQuestionsInfo.QuestionsInfo = questionsInfo

	t, _ := template.ParseFiles("./views/questions.html")
	t.Execute(w, getAllQuestionsInfo)
}

// perform edit functionality
func editQuesionHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")
	sequence := r.FormValue("sequence")
	qId := r.FormValue("qId")
	stmt1, _ := db.Prepare("update questions set description = ($1), sequence = ($2) where id = ($3)")
	stmt1.Query(description, sequence, qId)
	http.Redirect(w, r, "questions", 301)
}

//  delete questions functionality
func deleteQuestionHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	qId := r.FormValue("qid")
	stmt, _ := db.Prepare("select deleted from questions where id = ($1)")
	rows, _ := stmt.Query(qId)
	q := questionsInformation{}
	status := "no"
	for rows.Next() {
		rows.Scan(&q.Deleted)
		if q.Deleted != nil{
			status = "no"
		} else if q.Deleted == nil{
			status = "yes"
		}

	if status == "yes" {
		stmt1, _ := db.Prepare("update questions set deleted = NOW() where id = ($1)")
		stmt1.Query(qId)
	} else if status == "no" {
		stmt1, _ := db.Prepare("update questions set deleted = NULL where id = ($1)")
		stmt1.Query(qId)
	}
	w.Write([]byte(status))
	}
}

// add questions functionality
func addQuestionsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	description := r.FormValue("description")
	sequence := r.FormValue("sequence")
	stmt2, _ := db.Prepare("insert into questions (description, sequence, created) values($1, $2, NOW())")
	stmt2.Query(description, sequence)
	http.Redirect(w, r, "questions", 301)
}

// retrive challenges from database.
func challengesHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	getAllQuestionsInfo := getAllQuestionsInfo{}
	var buffer bytes.Buffer
	buffer.WriteString("select id, description, deleted from challenges where deleted is null order by id")
	rows3, _ := db.Query(buffer.String())

	questionsInfo := []questionsInformation{}
	q := questionsInformation{}
	var encodedChallenge string

	for rows3.Next() {
		err := rows3.Scan(&q.Id, &encodedChallenge, &q.Deleted)
		if q.Deleted != nil{
			q.Flag = 1
		} else if q.Deleted == nil{
			q.Flag = 0
		}

		//Decode the encrypted challenge from database...
		decodedChallenge, err := base64.StdEncoding.DecodeString(encodedChallenge)
		if err != nil {
			fmt.Println("decode error:", err)
			return
		}
		//==================================================

		//convert decrypted challenge to string from byte and store it into structure====================
		var m = map[string]*struct{ challenge string }{
		"foo": {"Challenge"},
		}

		m["foo"].challenge = string(decodedChallenge)[1:150]

		q.Description = m["foo"].challenge
		//======================================================

		questionsInfo = append(questionsInfo, q)
		checkErr(err)
	}

	getAllQuestionsInfo.QuestionsInfo = questionsInfo
	t, _ := template.ParseFiles("./views/programmingtest.html")
	t.Execute(w, getAllQuestionsInfo)
}

func allChallengesHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	getAllQuestionsInfo := getAllQuestionsInfo{}
	var buffer bytes.Buffer
	buffer.WriteString("select id, description, deleted from challenges order by id")
	rows3, _ := db.Query(buffer.String())

	questionsInfo := []questionsInformation{}
	q := questionsInformation{}
	var encodedChallenge string

	for rows3.Next() {
		err := rows3.Scan(&q.Id, &encodedChallenge, &q.Deleted)
		if q.Deleted != nil{
			q.Flag = 1
		} else if q.Deleted == nil{
			q.Flag = 0
		}

		//Decode the encrypted challenge from database...
		decodedChallenge, err := base64.StdEncoding.DecodeString(encodedChallenge)
		if err != nil {
			fmt.Println("decode error:", err)
			return
		}
		//==================================================

		//convert decrypted challenge to string from byte and store it into structure====================
		var m = map[string]*struct{ challenge string }{
		"foo": {"Challenge"},
		}

		m["foo"].challenge = string(decodedChallenge)[1:150]

		q.Description = m["foo"].challenge
		//======================================================

		questionsInfo = append(questionsInfo, q)
		checkErr(err)
	}

	getAllQuestionsInfo.QuestionsInfo = questionsInfo
	t, _ := template.ParseFiles("./views/programmingtest.html")
	t.Execute(w, getAllQuestionsInfo)
}

// mark challenge as a deleted.
func deleteChanllengesHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	qId := r.FormValue("qid")
	stmt, _ := db.Prepare("select deleted from challenges where id = ($1)")
	rows, _ := stmt.Query(qId)
	q := questionsInformation{}
	status := "no"
	for rows.Next() {
		rows.Scan(&q.Deleted)
		if q.Deleted != nil{
			status = "no"
		} else if q.Deleted == nil{
			status = "yes"
		}

	if status == "yes" {
		stmt1, _ := db.Prepare("update challenges set deleted = NOW() where id = ($1)")
		stmt1.Query(qId)
	} else if status == "no" {
		stmt1, _ := db.Prepare("update challenges set deleted = NULL where id = ($1)")
		stmt1.Query(qId)
	}
	w.Write([]byte(status))
	}
}

// edit perticualr challenge.
func editChallengeHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	challengeId := r.FormValue("challengeId")
	description := r.FormValue("challengeDescription")

	//Encrypt the description to store in database with special charecters
	encodeChallenge := base64.StdEncoding.EncodeToString([]byte(description))
	//==================================

	stmt1, _ := db.Prepare("update challenges set description = ($1) where id = ($2)")
	stmt1.Query(encodeChallenge, challengeId)
	http.Redirect(w, r, "programmingtest", 301)
}

type GeneralInfo struct {
	Id string
	Name string
	Contact string
	Degree string
	College string
	YearOfCompletion string
	Email string
	Created time.Time
	Modified time.Time
	ChallengeAttempts string
	DateOnly string
	QuestionsAttended string
}

// display candidates information
func candidateHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	var query = "SELECT c.id, c.name, c.email, c.degree, c.college, c.yearOfCompletion, c.modified, max(c1.attempts)"
		query += " FROM candidates c JOIN sessions s ON c.id = s.candidateid"
		query += " JOIN challenge_answers c1 ON s.id = c1.sessionid"
		query += " where s.status = 0"
		query += " group by c.id"
		query += "  order by c.id asc "

	stmt1 := fmt.Sprintf(  query  )
	rows1, _ := db.Query(stmt1)

	UsersInfo := []GeneralInfo{}
	user := GeneralInfo{}
	for rows1.Next() {

		rows1.Scan(&user.Id, &user.Name, &user.Email, &user.Degree, &user.College, &user.YearOfCompletion, &user.Modified, &user.ChallengeAttempts)

		//extract only date from timestamp========
		str :=&user.Modified
		str1 := str.String()
		s := strings.Split(str1," ")
		user.DateOnly = s[0]
		//================================

		stmt2 := fmt.Sprintf("SELECT count(id) FROM questions_answers WHERE length(answer) > 0 AND  candidateid="+user.Id)
		rows2, _ := db.Query(stmt2)
		for rows2.Next() {
			rows2.Scan(&user.QuestionsAttended)
		}
		UsersInfo = append(UsersInfo, user)
	}
	t, _ := template.ParseFiles("./views/candidates.html")
	t.Execute(w, UsersInfo)
}

type ChallengeInfo struct {
	Id int
}

// add new programming challenges.
func newChallengeHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	desc := r.FormValue("description")

	//Encrypt the description to store in database with special charecters
	description := base64.StdEncoding.EncodeToString([]byte(desc))
	//==================================

	stmt2, _ := db.Prepare("insert into challenges (description, created) values($1, NOW())")
	stmt2.Query(description)

	http.Redirect(w, r, "programmingtest", 301)
}

func addChallengeHandler(c web.C, w http.ResponseWriter, r *http.Request) {

	http.Redirect(w, r, "programmingtest", 301)
}

func testcaseHandler(c web.C, w http.ResponseWriter, r *http.Request) {

	http.Redirect(w, r, "programmingtest", 301)
}

var Id GeneralInfo// candidate id to use for getting his information ..
//will display personal information of candidates..
func personalInformationHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	Id.Id = r.FormValue("id")

	QuestionsAttended := r.FormValue("queAttempt")
	ChallengeAttempts := r.FormValue("challengeAttmpt")

	stmt2 := fmt.Sprintf("SELECT name, email, contact, degree, college, yearofcompletion from candidates where id ="+Id.Id)
	rows3, _ := db.Query(stmt2)

	UsersInfo := []GeneralInfo{}
	user := GeneralInfo{}

	for rows3.Next() {
		rows3.Scan(&user.Name, &user.Email, &user.Contact, &user.Degree, &user.College, &user.YearOfCompletion)
		user.ChallengeAttempts = ChallengeAttempts
		user.QuestionsAttended = QuestionsAttended
		user.Id = Id.Id
		UsersInfo = append(UsersInfo, user)
	}
	t, _ := template.ParseFiles("./views/personalInformation.html")
	t.Execute(w, UsersInfo)
}

type GetQuestions struct {
	Questions string
	Ans  string
	Created time.Time
	DateTimeOnly string
}
type GeneralDetails struct{
	Id string
	QuestionAttempted string
	ChallengeAttempts string
}
type AllInfo struct{
	GetQuestions []GetQuestions
	GeneralDetails []GeneralDetails
}

//will display questions and answer given by candidates..
func questionDetailsHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	QuestionsAttended := r.FormValue("queAttempt")
	ChallengeAttempts := r.FormValue("challengeAttmpt")

	var query = "SELECT questions.description, questions_answers.answer, questions_answers.Created"
		query += " FROM questions"
		query += " INNER JOIN questions_answers"
		query += " ON questions.id = questions_answers.questionsid"
		query += " where candidateid = "+ Id.Id +""
		query += " ORDER BY questions.sequence"

	rows, _ := db.Query(query)
	questionsInfo := []GetQuestions{}
	qinfo := GetQuestions{}
	GeneralInfo := []GeneralDetails{}
	g := GeneralDetails{}

	for rows.Next() {
		err := rows.Scan(&qinfo.Questions, &qinfo.Ans, &qinfo.Created)

		//extract only date - time from timestamp========
		str :=&qinfo.Created
		str1 := str.String()
		s := strings.Split(str1,".")
		qinfo.DateTimeOnly = s[0]
		//================================

		questionsInfo = append(questionsInfo, qinfo)
		checkErr(err)
	}
	g.QuestionAttempted = QuestionsAttended
	g.ChallengeAttempts = ChallengeAttempts
	g.Id = Id.Id

	GeneralInfo = append(GeneralInfo, g)

	Details := AllInfo{}
	Details.GetQuestions = questionsInfo
	Details.GeneralDetails = GeneralInfo

	t, _ := template.ParseFiles("./views/questionDetails.html")
	t.Execute(w, Details)
}

type GetChallenge struct {
	Challenge string
	QuestionAttempted string
	ChallengeAttempts string
	Id string
}

type LattestAnswer struct{
 	Answer string
 	Lang string
 }

type GetAnswers struct {
	Answer string
	Attempt string
}

type AllDetail struct {
	GetChallenge []GetChallenge
	GetAnswers []GetAnswers
	LattestAnswer []LattestAnswer

}
//will display answer of challenge...
func challengeDetailsHandlers(c web.C, w http.ResponseWriter, r *http.Request) {
	QuestionsAttended := r.FormValue("queAttempt")
	ChallengeAttempts := r.FormValue("challengeAttmpt")

	stmt1, _ := db.Prepare("select description from challenges where id = (select challengeid from sessions where candidateid=($1) AND id =(select MAX(id) from sessions where status = 0 AND candidateid =($2)))")
	rows1, _ := stmt1.Query(Id.Id, Id.Id)
	challenge := []GetChallenge{}
	q := GetChallenge{}
	var encodedChallenge string
	for rows1.Next() {
		err := rows1.Scan(&encodedChallenge)
		checkErr(err)
	}
	//Decode the encrypted challenge from database...
	decodedChallenge, err := base64.StdEncoding.DecodeString(encodedChallenge)
	if err != nil {
		fmt.Println("decode error:", err)
		return
	}
	//==================================================

	//convert decrypted challenge to string from byte and store it into structure====================
	var m = map[string]*struct{ challenge string }{
		"foo": {"Challenge"},
	}

	m["foo"].challenge = string(decodedChallenge)

	q.Challenge = m["foo"].challenge
	q.QuestionAttempted = QuestionsAttended
	q.ChallengeAttempts = ChallengeAttempts
	q.Id = Id.Id
	challenge = append(challenge, q)

	stmt2, _ := db.Prepare("select answer, attempts, language from challenge_answers where sessionid = (select max(id) from sessions where candidateid=($1) AND status = 0) order by attempts")
	rows2, _ := stmt2.Query(Id.Id)
	answer := []GetAnswers{}

	A := GetAnswers{}
	B := LattestAnswer{}
	BB := []LattestAnswer{}

	for rows2.Next() {
		err := rows2.Scan(&B.Answer, &A.Attempt, &B.Lang)
		answer = append(answer, A)
		checkErr(err)
	}
	BB = append(BB, B)
	allDetails := AllDetail{}
	allDetails.GetChallenge = challenge
	allDetails.GetAnswers = answer
	allDetails.LattestAnswer = BB

	t, _ := template.ParseFiles("./views/challengeDetails.html")
	t.Execute(w, allDetails)
}

type ChallengeCases struct{
	Id int
	Input string
	Output string
	Default bool
	Challenge string
	Flag int
}
type ChallengeDesc struct{
	Challenge string
}
type AllDetails struct{
	ChallengeCases []ChallengeCases
	ChallengeDesc ChallengeDesc
}

// add testcases for perticular challenge.
var challengeId string
func addTestCase(c web.C, w http.ResponseWriter, r *http.Request) {
	var qId string = r.URL.Query().Get("qid")
	if qId != "" {
		challengeId = qId
		Challenge := ChallengeDesc{}
		var encodedChallenge string
		err := db.QueryRow("select description from challenges where id = $1", challengeId).Scan(&encodedChallenge)
		checkErr(err)

		//Decode the encrypted challenge from database...
		decodedChallenge, err := base64.StdEncoding.DecodeString(encodedChallenge)
		if err != nil {
			fmt.Println("decode error:", err)
			return
		}

		//convert decrypted challenge to string from byte and store it into structure====================
		var m = map[string]*struct{ challenge string }{
			"foo": {"Challenge"},
		}

		m["foo"].challenge = string(decodedChallenge)
		Challenge.Challenge = m["foo"].challenge

		stmt1, _ := db.Prepare("select id, input, output, defaultcase from challenge_cases where challengeid = ($1) order by id")
		rows1, _ := stmt1.Query(challengeId)

		challengeCases := []ChallengeCases{}
		q := ChallengeCases{}

		for rows1.Next() {
			err := rows1.Scan(&q.Id, &q.Input, &q.Output, &q.Default)
			checkErr(err)
			if q.Default == true{
				q.Flag = 0
			} else if q.Default == false{
				q.Flag = 1
			}

			challengeCases = append(challengeCases, q)
		}
		allDetails := AllDetails{}
		allDetails.ChallengeCases = challengeCases
		allDetails.ChallengeDesc = Challenge

		t, _ := template.ParseFiles("./views/addTestCases.html")
		t.Execute(w, allDetails)

	} else {
		input := r.FormValue("input")
		output := r.FormValue("output")

		stmt1, _ := db.Prepare("insert into challenge_cases(challengeid, input, output, defaultCase, created) values ($1, $2, $3, $4, NOW());")
		stmt1.Query(challengeId, input, output, false)

		http.Redirect(w, r,  "addTestCases?qid=" + challengeId , 301)
	}
}

// retrive testcase input output
func getTestCaseHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	testCaseId := r.FormValue("testCaseId")
	challengeId := r.FormValue("challengeId")
	stmt,_ :=db.Prepare("select input, output from challenge_cases where challengeId = ($1) AND id = ($2)")
	rows, _ := stmt.Query(challengeId, testCaseId)
	challengeCases := []ChallengeCases{}
		q := ChallengeCases{}
		for rows.Next() {
			err := rows.Scan(&q.Input, &q.Output)
			checkErr(err)
			challengeCases = append(challengeCases, q)
		}
		b, err := json.Marshal(challengeCases)
		if err != nil {
			fmt.Printf("Error: %s", err)
			return;
		}
	//==========================
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(b))//set response...
}

// edit test case .
func editTestCaseHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	testCaseId := r.FormValue("testCaseId")
	challengeId := r.FormValue("challengeId")
	input := r.FormValue("input")
	output := r.FormValue("output")
	stmt,_ :=db.Prepare("update challenge_cases set input = ($1), output = ($2) where id = ($3) AND challengeId = ($4)")
	stmt.Query(input, output, testCaseId, challengeId)
	http.Redirect(w, r,  "addTestCases?qid=" + challengeId , 301)
}

//Searching will perform on candidates information...
func searchHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	//data comes from admin side...
	name := r.FormValue("name")
	degree := r.FormValue("degree")
	college := r.FormValue("college")
	year := r.FormValue("year")

	// default query for search...
	var query ="SELECT c.id,c.name, c.email, c.degree, c.college, c.yearOfCompletion, c.modified, max(c1.attempts)"
	query += " FROM candidates c"
	query += " JOIN sessions s ON c.id = s.candidateid"
	query += " JOIN challenge_answers c1 ON s.id = c1.sessionid"
	query += " where s.status=0 "

	var stmt1 string

	// =======================making query for search =================================

if(year =="All"){//will search for all the year passing out candidates..
	if(name ==""){
		if(degree == "" && college == ""){//search for all the field..
			stmt1 = fmt.Sprintf(query+" group by c.id order by c.id asc ")
			} else if(degree == ""){//will search for college only..
				stmt1 = fmt.Sprintf(query+" AND (c.college ILIKE '%%%s%%')  group by c.id order by c.id asc ",college)
				}else if(college == ""){//will search for degree only..
					stmt1 = fmt.Sprintf(query+" AND (c.degree ILIKE '%%%s%%') group by c.id order by c.id asc ",degree)
					}else{
						stmt1 = fmt.Sprintf(query+" AND ((c.degree ILIKE '%%%s%%') AND (c.college ILIKE '%%%s%%') ) group by c.id order by c.id asc ",degree,college)
					}

	} else if(degree == ""){
		 if(degree == "" && college == ""){//will search for name only..
				stmt1 = fmt.Sprintf(query+" AND ((c.name ILIKE '%%%s%%') OR (c.email LIKE '%%%s%%')) group by c.id order by c.id asc ",name,name)
				} else if(degree == ""){// will search for both name and college fields...
					stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email ILIKE '%%%s%%')) AND (c.college ILIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,college)
						}

	} else if(college == ""){//will search for name and degree both field....
		stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email ILIKE '%%%s%%')) AND (c.degree ILIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,degree)
		} else {//will search for all the fields..
			stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email ILIKE '%%%s%%')) AND (c.college ILIKE '%%%s%%') AND (c.degree ILIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,college,degree)
			}

} else {//will search for specific year passing out candidates..
	if(name ==""){
		if(degree == "" && college == ""){//search for all the field with specific year..
			stmt1 = fmt.Sprintf(query+" AND (c.yearOfCompletion::text LIKE '%%%s%%')group by c.id order by c.id asc ",year)
			} else if(degree == ""){//will search for college only with specific year..
				stmt1 = fmt.Sprintf(query+" AND ((c.college ILIKE '%%%s%%') AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",college,year)
				}else if(college == ""){//will search for degree only with specific year..
					stmt1 = fmt.Sprintf(query+" AND ((c.degree ILIKE '%%%s%%') AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",degree,year)
					}else{
						stmt1 = fmt.Sprintf(query+" AND ((c.degree ILIKE '%%%s%%') AND (c.college ILIKE '%%%s%%') AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",degree,college,year)
					}

	} else if(degree == ""){
		 if(degree == "" && college == ""){//will search for name only with specific year..
				stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email LIKE '%%%s%%')) AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,year)
				} else if(degree == ""){// will search for both name and college fields with specific year...
					stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email ILIKE '%%%s%%')) AND (c.college ILIKE '%%%s%%') AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,college,year)
						}

	} else if(college == ""){//will search for name and degree both field with specific year....
		stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email ILIKE '%%%s%%')) AND (c.degree ILIKE '%%%s%%') AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,degree,year)
		} else {//will search for all the fields with specific year..
			stmt1 = fmt.Sprintf(query+" AND (((c.name ILIKE '%%%s%%') OR (c.email ILIKE '%%%s%%')) AND (c.college ILIKE '%%%s%%') AND (c.degree ILIKE '%%%s%%') AND (c.yearOfCompletion::text LIKE '%%%s%%')) group by c.id order by c.id asc ",name,name,college,degree,year)
			}
}
	//==============================================================================================================================================

	rows1, err := db.Query(stmt1)
	if(err != nil){
		panic (err)
	}
	UsersInfo := []GeneralInfo{}
	user := GeneralInfo{}

	for rows1.Next() {

		rows1.Scan(&user.Id, &user.Name, &user.Email, &user.Degree, &user.College, &user.YearOfCompletion, &user.Modified, &user.ChallengeAttempts)

		//extract only date from timestamp========
		str :=&user.Modified
		str1 := str.String()
		s := strings.Split(str1," ")
		user.DateOnly = s[0]
		//================================

		//=========will count no of attended questions========
		stmt2 := fmt.Sprintf("SELECT count(id) FROM questions_answers WHERE length(answer) > 0 AND  candidateid="+user.Id)
		rows2, _ := db.Query(stmt2)
		for rows2.Next() {
			rows2.Scan(&user.QuestionsAttended)
		}
		UsersInfo = append(UsersInfo, user)
	}
	//================================================

	//========to convert response to JSON ==========
	b, err := json.Marshal(UsersInfo)
	if err != nil {
			fmt.Printf("Error: %s", err)
			return;
	}
	//==========================

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(b))//set response...
}

type ChallengeAttemptsDetails struct{
	Source string
	Language string
}

//will return the perticullar attempted challenge source code
func challengeAttemptHandler(c web.C, w http.ResponseWriter, r *http.Request) {

	candidateId := r.FormValue("candidateID")
	attemptNo := r.FormValue("attemptNo")

	attemptDetails := ChallengeAttemptsDetails{}

	stmt,_ :=db.Prepare("SELECT answer, language FROM challenge_answers WHERE attempts=($1) AND sessionid = (select MAX(id) from sessions where candidateid=($2) AND status= 0 )")
	rows, _ := stmt.Query(attemptNo, candidateId)

	for rows.Next() {
			err := rows.Scan(&attemptDetails.Source, &attemptDetails.Language)
			checkErr(err)
		}
	//========to convert response to JSON ==========
	b, err := json.Marshal(attemptDetails)
	if err != nil {
			fmt.Printf("Error: %s", err)
			return;
	}
	//==========================
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(b))//set response...
}

func deleteTestcaseHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	challengeId := r.FormValue("challengeId")
	testCaseId := r.FormValue("testCaseId")

	stmt,_ :=db.Prepare("DELETE from challenge_cases WHERE challengeid = ($1) AND id = ($2)")
	_, err := stmt.Query(challengeId, testCaseId)
	checkErr(err)
}

func DefaultTestcaseHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	testCaseId := r.FormValue("testCaseId")
	challengeId := r.FormValue("challengeId")

	// err := db.QueryRow("select id from challenge_cases where challengeid = ($1) AND defaultcase = true", challengeId).Scan(&defaultCase)
	db.Query("update challenge_cases set defaultcase = false where challengeId = ($1)", challengeId)

	stmt1,_ :=db.Prepare("UPDATE challenge_cases SET defaultcase = true WHERE id = ($1) AND challengeid = ($2)")
	_, errr := stmt1.Query(testCaseId, challengeId)
	checkErr(errr)

	rows, _ := db.Query("select id, defaultcase from challenge_cases where challengeid = ($1)", challengeId)
	challengeCases := []ChallengeCases{}
	q := ChallengeCases{}
	for rows.Next() {
			err := rows.Scan(&q.Id, &q.Default)
			checkErr(err)
	}
	challengeCases = append(challengeCases, q)

	//========to convert response to JSON ==========
	b, err := json.Marshal(challengeCases)
	if err != nil {
			fmt.Printf("Error: %s", err)
			return;
	}
	//==========================
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(b))//set response...
}


type getChallengeDescription struct {
	Description string
}

// get challenge information using challenge id.
func getChallengeInfoHandler(c web.C, w http.ResponseWriter, r *http.Request) {
	challengeId := r.FormValue("challengeId")

	stmt,_ := db.Prepare("select description from challenges where id = ($1)")
	rows, _ := stmt.Query(challengeId)
	challenge := getChallengeDescription{}
	for rows.Next() {
		err := rows.Scan(&challenge.Description)
		checkErr(err)
	}
	decodedChallenge, err := base64.StdEncoding.DecodeString(challenge.Description)
	if err != nil {
		fmt.Println("decode error:", err)
		return
	}
	//==================================================

	//convert decrypted challenge to string from byte and store it into structure====================
	var m = map[string]*struct{ challenge string }{
		"foo": {"Challenge"},
	}

	m["foo"].challenge = string(decodedChallenge)
	challenge.Description = m["foo"].challenge;
	b, err := json.Marshal(challenge)
	if err != nil {
			fmt.Printf("Error: %s", err)
			return;
	}
	//==========================
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(b))//set response...

}


func main() {
	db = setupDB()
	defer db.Close()

	goji.Handle("/", candidateHandler)
	goji.Handle("/search", searchHandler)
	goji.Handle("/candidates", candidateHandler)
	goji.Handle("/allQuestions", allQuestionsHandler)
	goji.Handle("/questions", questionsHandler)
	goji.Handle("/addQuestions", addQuestionsHandler)
	goji.Handle("/editquestion", editQuesionHandler)
	goji.Handle("/deleteQuestion", deleteQuestionHandler)
	goji.Handle("/deleteChallenge", deleteChanllengesHandler)
	goji.Handle("/personalInformation", personalInformationHandler)
	goji.Handle("/questionDetails", questionDetailsHandler)
	goji.Handle("/challengeDetails", challengeDetailsHandlers)
	goji.Handle("/addchallenge", addChallengeHandler)
	goji.Handle("/testcase", testcaseHandler)
	goji.Handle("/addTestCases", addTestCase)
	goji.Handle("/programmingtest", challengesHandler)
	goji.Handle("/allChallenges", allChallengesHandler)
	goji.Handle("/editchallenge", editChallengeHandler)
	goji.Handle("/newChallenge", newChallengeHandler)
	goji.Handle("/challengeAttempt", challengeAttemptHandler)
	goji.Post("/deleteTestCase", deleteTestcaseHandler)
	goji.Post("/setDefaultTestcase", DefaultTestcaseHandler)
	goji.Post("/getQuestionInfo", getQuestionInfoHandler);
	goji.Post("/getTestCase", getTestCaseHandler)
	goji.Handle("/editTestCase", editTestCaseHandler)
	goji.Post("/getChallengeInfo", getChallengeInfoHandler)

	http.Handle("/assets/css/", http.StripPrefix("/assets/css/", http.FileServer(http.Dir("assets/css"))))
	http.Handle("/assets/jquery/", http.StripPrefix("/assets/jquery/", http.FileServer(http.Dir("assets/jquery"))))
	http.Handle("/assets/js/", http.StripPrefix("/assets/js/", http.FileServer(http.Dir("assets/js"))))
	http.Handle("/assets/img/", http.StripPrefix("/assets/img/", http.FileServer(http.Dir("assets/img"))))
	http.Handle("/assets/fonts/", http.StripPrefix("/assets/fonts/", http.FileServer(http.Dir("assets/fonts"))))
	goji.Serve()
}
