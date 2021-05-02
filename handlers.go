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

type tm struct {
	name string
	mobile string
	link string
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

	var team []TeamMember
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

	sendSms(data.Stylist)
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

func apiSaveQuoteDetails(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var salonURL string
	var salonName string
	var data QuoteRespondent

	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	db.Create(&data)

	sID := data.SalonID

	switch sID {
	case 1:
		salonName = "Jakata Salon"
		salonURL = "https://www.jakatasalon.co.uk/"
	case 2:
		salonName = "Paul Kemp Hairdressing"
		salonURL = "https://www.paulkemphairdressing.com/"
	case 3:
		salonName = "Base Hairdressing"
		salonURL = "https://www.basehairdressing.com/"
	}

	client := textmagic.NewClient(os.Getenv("TEXT_MAGIC_USERNAME"), os.Getenv("TEXT_MAGIC_TOKEN"))
	name := strings.Split(data.Name, " ")[0]
	mobile := data.Mobile
	link := data.Link
	t := "Hi " + name + ", Here's a link to your quote at " + salonName + ": " + salonURL + "quote/" + link

	params := map[string]string{
		"phones": "+44" + mobile[1:],
		"text":   t,
	}

	message, err := client.CreateMessage(params)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(message.ID)
	}
}

func apiGetQuoteDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var services QuoteRespondent

	vars := mux.Vars(r)
	param := vars["link"]

	db.Where("link", param).First(&services)

	json, err := json.Marshal(services)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

//func apiSendQuoteDetails(w http.ResponseWriter, r *http.Request) {
//	decoder := json.NewDecoder(r.Body)
//
//	var data QuoteDetails
//	err := decoder.Decode(&data)
//	if err != nil {
//		panic(err)
//	}
//
//	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))
//
//	sender := "info@basehairdressing.co.uk"
//	subject := "New Message for Base"
//	body := data.Info
//	recipient := data.Email
//
//	// The message object allows you to add attachments and Bcc recipients
//	message := mg.NewMessage(sender, subject, body, recipient)
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
//	defer cancel()
//
//	// Send the message	with a 10 second timeout
//	resp, id, err := mg.Send(ctx, message)
//
//	if err != nil {
//		log.Fatal(err)
//	}
//
//	fmt.Printf("ID: %s Resp: %s\n", id, resp)
//
//	return
//}

func sendSms(n string) {
	var name, mobile, link string

	client := textmagic.NewClient(os.Getenv("TEXT_MAGIC_USERNAME"), os.Getenv("TEXT_MAGIC_TOKEN"))

	tm1 := tm{"Nat", "+447975690833", "12564"}
	tm2 := tm{"Georgia", "+447713452458", "13789"}
	tm3 := tm{"Matt", "+447378420023", "58839"}
	tm4 := tm{"Abbi", "+447960892938", "28509"}
	tm5 := tm{"Lauren-T", "+447534193140", "08348"}
	tm6 := tm{"Vikki", "+447833248653", "46748"}
	tm7 := tm{"Layla", "+447494187775", "35688"}
	tm8 := tm{"Laura", "+447989786237", "48938"}
	tm9 := tm{"Kellie", "+447805093942", "24389"}
	tm10 := tm{"Izzy", "+447817722920", "88453"}
	tm11 := tm{"Jo", "+447710408166", "23675"}
	tm12 := tm{"Abi", "+447388033659", "46347"}
	tm13 := tm{"Brad", "+447762329249", "34765"}
	tm14 := tm{"David", "+447539685042", "73834"}
	tm15 := tm{"Michelle", "+447714263500", "35278"}
	tm16 := tm{"Adam", "+447921806884", "45765"}
	tm17 := tm{"Jimmy", "+447939011951", "57833"}
	tm18 := tm{"Lauren-W", "+447984334430", "87648"}
	tm19 := tm{"Lucy", "+447432522388", "78598"}
	tm20 := tm{"Sophie", "+447793046731", "45748"}
	tm21 := tm{"Beth", "+447432094293", "68388"}
	tm22 := tm{"Ruby", "+447808034791", "32673"}
	tm23 := tm{"Jak Not Sure", "+447921806884", "98761"}
	tm24 := tm{"PK Not Sure", "+447921806884", "98762"}
	tm25 := tm{"B Not Sure", "+447921806884", "98763"}

	tms := []tm{tm1, tm2, tm3, tm4, tm5, tm6, tm7, tm8, tm9, tm10, tm11, tm12, tm13, tm14, tm15, tm16, tm17, tm18, tm19, tm20, tm21, tm22, tm23, tm24, tm25}

	for _, v := range tms {
		if n == v.name {
			name = v.name
			mobile = v.mobile
			link = v.link
		}
	}

	m := mobile
	t := "Hi " + name + ", a new client has registered with you! https://fast-basin-93128.herokuapp.com/" + link

	params := map[string]string{
		"phones": m,
		"text":   t,
	}
	message, err := client.CreateMessage(params)
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(message.ID)
	}
}