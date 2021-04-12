package salonserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/mailgun/mailgun-go/v3"
	"github.com/russross/blackfriday"
	"github.com/textmagic/textmagic-rest-go"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path"
	"sort"
	"strings"
	"time"
)

type Response struct {
	Message string `json:"message"`
}

func forceSsl(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("GO_ENV") == "production" {
			if r.Header.Get("x-forwarded-proto") != "https" {
				sslUrl := "https://" + r.Host + r.RequestURI
				http.Redirect(w, r, sslUrl, http.StatusTemporaryRedirect)
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

func responseJSON(w http.ResponseWriter, data interface{}) {
	json.NewEncoder(w).Encode(data)
}

func longName(n string) (l string) {
	if n == "brad" {
		n = "bradley"
	}
	if n == "nat" {
		n = "natalie"
	}
	if n == "matt" {
		n = "matthew"
	}
	return n
}

func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// Generate version number for scripts and css
	rand.Seed(time.Now().UnixNano())

	var t, d, i string

	var salonURL string

	switch salon {
	case 1:
		salonURL = "https://www.jakatasalon.co.uk"
	case 2:
		salonURL = "https://www.paulkemphairdressing.com"
	case 3:
		salonURL = "https://www.basehairdressing.com"
	}

	vars := mux.Vars(r)
	dir := vars["category"]
	name := vars["name"]

	if dir == "" && name == "" {
		name = "home"
	}

	if dir == "team" || dir == "team-info" && len(name) > 0 {
		m := TeamMember{}
		db.Where("salon = ? AND slug = ?", salon, name).First(&m)

		t = m.FirstName + " " + m.LastName
		d = m.Para1 + " " + m.Para2
		i = salonURL + "/dist/img/fb_meta/" + m.Slug + ".png"

	} else if dir == "reviews" {
		if name == "" || name == "all" {
			t = "Recent Reviews from our happy customers"
			d = "The team receives consistently great reviews. Check them out here. You can filter by stylist too"
			i = salonURL + "/dist/img/fb_meta/reviews.png"

		} else {
			r := Review{}
			ln := longName(name)
			param := strings.Title(ln)

			db.Where("salon = ?", salon).Where("stylist LIKE ?", "Staff: "+param+" %").First(&r)

			t = param + " recently received this great review!"
			d = r.Review
			i = salonURL + "/dist/img/fb_meta/" + name + ".png"
		}

	} else if dir == "blog" || dir == "blog-info" {
		path := path.Join(dir, name)

		data, err := ioutil.ReadFile(path + ".txt")
		if err != nil {
			fmt.Println(err)
			return
		}
		lines := strings.Split(string(data), "\n")
		t = string(lines[0])
		i = string(lines[4])
		d = string(lines[6])

	} else {
		split := strings.Split(name, "-")[0]

		page := path.Join(dir, split)

		m := MetaInfo{}
		db.Where("salon = ?", salon).Where("page = ?", page).First(&m)

		if m.Title != "" {
			t = m.Title
		} else {
			switch salon {
			case 1:
				t = "Fashion Forward Hairdressing"
			case 2:
				t = "A New Standard Of Hairdressing"
			case 3:
				t = "Academy for the next generation of super skilled stylists"
			}
		}

		if m.Text != "" {
			d = m.Text
		} else {
			switch salon {
			case 1:
				d = "Jakata is a fashion forward salon in Warrington Town Centre"
			case 2:
				d = "Paul Kemp Hairdressing is a luxurious hair salon right in the heart of Warrington town centre. Sister salon to the award winning Jakata Hair and Beauty team, the stunning Salon opened back in June 2011 with the aim to offer an ultra relaxing atmosphere, first class customer service, alongside the highest level of hairdressing expertise. The salon's talented hairdressers are all trained to the highest level in cutting, colouring and styling hair, with specialists in technical colour, hair straightening, wedding hair and hair extensions. The team has a wealth of experience in all aspects of hairdressing"
			case 3:
				d = "Base Hairdressing is an Academy for the next generation of super-skilled stylists"
			}
		}

		if m.Title != "" {
			i = salonURL + "/dist/img/fb_meta/" + m.Image + ".png"
		} else {
			i = salonURL + "/dist/img/fb_meta/home.png"
		}
	}

	path := path.Join(dir, name)

	v := string(rand.Intn(30))

	meta := map[string]string{
		"ogTitle":       t,
		"ogDescription": d,
		"ogImage":       i,
		"ogImageWidth":  "1200",
		"ogImageHeight": "628",
		"ogUrl":         salonURL + "/" + path,
		"version":       v,
	}

	if err := tpl.Execute(w, meta); err != nil {
		panic(err)
	}
}

// api

func apiTeam(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	team := []TeamMember{}
	db.Where("salon = ?", salon).Order("position").Find(&team)

	json, err := json.Marshal(team)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiTeamMember(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	param := vars["slug"]

	tm := TeamMember{}
	db.Where("salon = ?", salon).Where("slug = ?", param).First(&tm)

	json, err := json.Marshal(tm)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiReviews(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	reviews := []Review{}
	vars := mux.Vars(r)
	param := vars["tm"]

	ln := longName(param)

	param = strings.Title(ln)

	if param == "All" {
		db.Where("salon = ?", salon).Find(&reviews)
	} else {
		db.Where("salon = ?", salon).Where("stylist LIKE ?", "Staff: "+param+" %").Find(&reviews)
	}

	json, err := json.Marshal(reviews)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiSendMessage(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data ContactMessage
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))

	sender := "info@basehairdressing.co.uk"
	subject := "New Message for Base"
	body := data.Message
	recipient := "adam@jakatasalon.co.uk"

	// The message object allows you to add attachments and Bcc recipients
	message := mg.NewMessage(sender, subject, body, recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message	with a 10 second timeout
	resp, id, err := mg.Send(ctx, message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)

	return
}

func apiJoinus(w http.ResponseWriter, r *http.Request) {
	var salonName, address string

	switch salon {
	case 1:
		salonName = "Jakata"
	case 2:
		salonName = "PK"
	case 3:
		salonName = "Base"
	}

	switch salon {
	case 1:
		address = "info@jakatasalon.co.uk"
	case 2:
		address = "info@paulkemphairdressing.com"
	case 3:
		address = "info@basehairdressing.com"
	}

	decoder := json.NewDecoder(r.Body)

	var data JoinusApplicant
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	db.Create(&data)

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))

	sender := address
	subject := "New " + data.Role + " Applicant for " + salonName
	body := data.Info
	recipient := "adam@jakatasalon.co.uk"

	// The message object allows you to add attachments and Bcc recipients
	message := mg.NewMessage(sender, subject, body, recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message	with a 10 second timeout
	resp, id, err := mg.Send(ctx, message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)

	return
}

func apiModel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data ModelApplicant
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	db.Create(&data)

	return
}

func apiBookingRequest(w http.ResponseWriter, r *http.Request) {
	var data BookingRequest
	var br BookingRequest
	var dbResponse Response

	json.NewDecoder(r.Body).Decode(&data)

	db.Where("mobile", data.Mobile).First(&br)

	if data.Mobile == br.Mobile {
		dbResponse.Message = "You've already registered! We'll be in touch soon"
		responseJSON(w, dbResponse)
		return
	}
	db.Create(&data)

	sm := getStaffMobile()

	sendSms(sm[data.Stylist])
	return
}

func apiBlogPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var blog Blog

	params := mux.Vars(r)

	data, err := ioutil.ReadFile("blog/" + params["slug"] + ".txt")
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	title := string(lines[0])
	date := string(lines[1])
	author := string(lines[2])
	image := string(lines[3])
	intro := string(lines[6])
	text := strings.Join(lines[6:], "\n")
	body := blackfriday.MarkdownBasic([]byte(text))
	slug := params["slug"]

	blog = Blog{Slug: slug, Date: date, Title: title, Image: image, Intro: intro, Author: author, Body: string(body)}

	json, err := json.Marshal(blog)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiBlogPosts(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var blogs []Blog

	files, err := ioutil.ReadDir("./blog")
	if err != nil {
		log.Fatal(err)
	}

	for _, fi := range files {
		data, err := ioutil.ReadFile("./blog/" + fi.Name())
		if err != nil {
			fmt.Println("File reading error", err)
			return
		}
		slug := strings.Split(fi.Name(), ".")
		lines := strings.Split(string(data), "\n")
		title := string(lines[0])
		date := string(lines[1])
		author := string(lines[2])
		image := string(lines[3])
		intro := string(lines[6])
		text := strings.Join(lines[6:8], "\n")
		body := blackfriday.MarkdownBasic([]byte(text))

		blogs = append(blogs, Blog{Slug: slug[0], Date: date, Title: title, Image: image, Intro: intro, Author: author, Body: string(body)})
	}
	sort.SliceStable(blogs, func(i, j int) bool { return blogs[i].Date > blogs[j].Date })

	json, err := json.Marshal(blogs)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiNewsItems(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var blogs []Blog

	files, err := ioutil.ReadDir("./blog")
	if err != nil {
		log.Fatal(err)
	}

	for _, fi := range files {
		data, err := ioutil.ReadFile("./blog/" + fi.Name())
		if err != nil {
			fmt.Println("File reading error", err)
			return
		}
		slug := strings.Split(fi.Name(), ".")
		lines := strings.Split(string(data), "\n")
		title := string(lines[0])
		image := string(lines[3])
		date := string(lines[1])
		text := string(lines[6])
		body := strings.Split(text, ".")

		blogs = append(blogs, Blog{Slug: slug[0], Date: date, Title: title, Image: image, Body: body[0]})
	}
	sort.SliceStable(blogs, func(i, j int) bool { return blogs[i].Date > blogs[j].Date })

	json, err := json.Marshal(blogs[0:4])
	if err != nil {
		log.Panic(err)
	}
	w.Write(json)
}

func apiServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var p []Service

	db.Find(&p)

	json, err := json.Marshal(p)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiStylists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var s []TeamMember

	db.Find(&s)

	json, err := json.Marshal(s)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiLevels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var l []Level

	db.Find(&l)

	json, err := json.Marshal(l)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiSalons(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var s []Salon

	db.Find(&s)

	json, err := json.Marshal(s)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiSendQuoteDetails(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data QuoteDetails
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))

	sender := "info@basehairdressing.co.uk"
	subject := "New Message for Base"
	body := data.Info
	recipient := data.Email

	// The message object allows you to add attachments and Bcc recipients
	message := mg.NewMessage(sender, subject, body, recipient)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message	with a 10 second timeout
	resp, id, err := mg.Send(ctx, message)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)

	return
}

func getStaffMobile() map[string]string {
	s := make(map[string]string)

	s["Nat"] = "+447975690833"
	s["Georgia"] = "+447713452458"
	s["Matt"] = "+447378420023"
	s["Abbi"] = "+447960892938"
	s["Lauren-T"] = "+447534193140"
	s["Vikki"] = "+447833248653"
	s["Layla"] = "+447494187775"
	s["Laura"] = "+447989786237"
	s["Kellie"] = "+447805093942"
	s["Izzy"] = "+447817722920"
	s["Jo"] = "+447710408166"
	s["Abi"] = "+447388033659"
	s["Brad"] = "+447762329249"
	s["David"] = "+447539685042"
	s["Michelle"] = "+447714263500"
	s["Adam"] = "+447921806884"
	s["Jimmy"] = "+447939011951"
	s["Lauren-W"] = "+447984334430"
	s["Lucy"] = "+447432522388"
	s["Sophie"] = "+447793046731"
	s["Beth"] = "+447432094293"
	s["Ruby"] = "+447808034791"
	s["Jak Not Sure"] = "+447921806884"
	s["PK Not Sure"] = "+447921806884"
	s["B Not Sure"] = "+447921806884"

	return s
}

func sendSms(n string) {
	client := textmagic.NewClient("adamcarter3", "s66tZKiQuVk2BkrdAcPuSFIYGSUMae")

	params := map[string]string{
		"phones": n,
		"text":   "A new client has registered with you!",
	}
	message, err := client.CreateMessage(params)

	if err != nil {
		log.Print(err)
	} else {
		log.Print(message.ID)
	}
}