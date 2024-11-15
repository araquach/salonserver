package salonserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/lib/pq"
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
	name   string
	mobile string
	link   string
}

type UpdateFields struct {
	ID       int64  `json:"id"`
	Notes    string `json:"notes,omitempty"`
	FollowUp string `json:"follow_up,omitempty"`
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

func home(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	// Generate version number for scripts and css
	rand.Seed(time.Now().UnixNano())

	var t, d, i, fn string

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
		DB.Where("salon = ? AND slug = ?", salon, name).First(&m)

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

			DB.Where("salon = ?", salon).Where("stylist LIKE ?", param+" %").First(&r)

			t = param + " recently received this great review!"
			d = r.Review
			i = salonURL + "/dist/img/fb_meta/" + name + ".png"
		}

	} else if dir == "blog" || dir == "blog-info" {
		files, err := ioutil.ReadDir("blog")
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			if strings.Contains(f.Name(), name) {
				fn = f.Name()
			}
		}

		data, err := ioutil.ReadFile("blog/" + fn)
		if err != nil {
			fmt.Println(err)
			return
		}
		lines := strings.Split(string(data), "\n")
		t = lines[0]
		i = lines[4]
		d = lines[6]

	} else {
		page := path.Join(dir, name)

		m := MetaInfo{}
		DB.Where("salon = ?", salon).Where("page = ?", page).First(&m)

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
	DB.Where("salon = ?", salon).Order("position").Find(&team)

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
	DB.Where("salon = ?", salon).Where("slug = ?", param).First(&tm)

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

	if strings.Contains(param, "-") {
		param = strings.Split(param, "-")[0]
	}

	ln := longName(param)

	param = strings.Title(ln)

	if param == "All" {
		DB.Where("salon = ?", salon).Where("rating > 3").Where("review != ''").Where("review != '\"'").Limit(50).Find(&reviews)
	} else {
		DB.Where("salon = ?", salon).Where("stylist LIKE ?", param+" %").Where("rating > 3").Where("review != ''").Where("review != '\"'").Limit(20).Find(&reviews)
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
		salonName = "Jakata Salon"
		address = "info@jakatasalon.co.uk"
	case 2:
		salonName = "Paul Kemp Hairdressing"
		address = "info@paulkemphairdressing.com"
	case 3:
		salonName = "Base Hairdressing"
		address = "info@basehairdressing.com"
	}

	decoder := json.NewDecoder(r.Body)

	var data JoinusApplicant
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	if err := DB.Create(&data).Error; err != nil {
		http.Error(w, "Failed to save data", http.StatusInternalServerError)
		return
	}

	htmlContent, err := ParseEmailTemplate("templates/recruitment.gohtml", data)
	if err != nil {
		http.Error(w, "Failed to parse HTML email template", http.StatusInternalServerError)
		return
	}

	textContent, err := ParseEmailTemplate("templates/recruitment.txt", data)
	if err != nil {
		http.Error(w, "Failed to parse text email template", http.StatusInternalServerError)
		return
	}

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))

	sender := address
	subject := "New " + data.Role + " Applicant for " + salonName
	body := textContent
	recipient := "adam@jakatasalon.co.uk"

	m := mg.NewMessage(sender, subject, body, recipient)

	m.SetHtml(htmlContent)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message	with a 10 second timeout
	resp, id, err := mg.Send(ctx, m)
	if err != nil {
		http.Error(w, "Failed to send email", http.StatusInternalServerError)
		return
	}

	// Send email to applicant

	applicantHtmlContent, err := ParseEmailTemplate("templates/recruitment/initial.gohtml", struct {
		Name      string
		SalonName string
	}{
		Name:      data.Name,
		SalonName: salonName,
	})
	if err != nil {
		http.Error(w, "Failed to parse HTML email template for applicant", http.StatusInternalServerError)
		return
	}
	applicantTextContent, err := ParseEmailTemplate("templates/recruitment/initial.txt", struct {
		Name      string
		SalonName string
	}{
		Name:      data.Name,
		SalonName: salonName,
	})
	if err != nil {
		http.Error(w, "Failed to parse text email template for applicant", http.StatusInternalServerError)
		return
	}

	applicantEmail := data.Email
	applicantSubject := "Thank you for applying to " + salonName
	applicantBody := applicantTextContent
	mApplicant := mg.NewMessage(sender, applicantSubject, applicantBody, applicantEmail)
	mApplicant.SetHtml(applicantHtmlContent)
	respApplicant, idApplicant, err := mg.Send(ctx, mApplicant)
	if err != nil {
		http.Error(w, "Failed to send email to applicant", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "Application submitted successfully. ID: %s Resp: %s. Applicant Email ID: %s Resp: %s\n", id, resp, idApplicant, respApplicant)
}

func apiJoinusApplicants(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	applicants := []JoinusApplicant{}
	DB.Order("id desc").Find(&applicants)

	json, err := json.Marshal(applicants)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiJoinusApplicant(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	vars := mux.Vars(r)
	param := vars["id"]

	applicant := JoinusApplicant{}
	DB.Preload("Notes").Where("id = ?", param).First(&applicant)

	json, err := json.Marshal(applicant)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiJoinUsApplicantUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Define a struct to decode only the fields you expect
	type updateInput struct {
		Notes    pq.StringArray `json:"notes" gorm:"type:text[]"`
		FollowUp string         `json:"follow_up"`
	}

	var input updateInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	vars := mux.Vars(r)
	applicantID := vars["id"] // Assuming you're using gorilla/mux or a similar router

	// Use a map to hold the fields to update to leverage GORM's Updates method
	updateFields := map[string]interface{}{}
	if len(input.Notes) > 0 {
		updateFields["notes"] = input.Notes
	}

	if input.FollowUp != "" {
		updateFields["follow_up"] = input.FollowUp
	}

	if len(updateFields) == 0 {
		http.Error(w, "No valid fields provided for update", http.StatusBadRequest)
		return
	}

	var applicant JoinusApplicant
	if result := DB.First(&applicant, applicantID); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusNotFound)
		return
	}

	if result := DB.Model(&applicant).Updates(updateFields); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(applicant); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func apiJoinUsEmailer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	type RequestBody struct {
		ID            uint   `json:"ID"`
		EmailResponse string `json:"EmailResponse"`
	}

	var requestBody RequestBody
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&requestBody)
	if err != nil {
		http.Error(w, `{"error": "Failed to decode request body", "details": "`+err.Error()+`"}`, http.StatusBadRequest)
		return
	}

	var applicant JoinusApplicant
	if err := DB.First(&applicant, requestBody.ID).Error; err != nil {
		http.Error(w, `{"error": "Applicant not found"}`, http.StatusNotFound)
		return
	}

	var salonName, address, logoLink, logoLinkWhite string
	switch applicant.Salon {
	case 1:
		salonName = "Jakata Salon"
		address = "info@jakatasalon.co.uk"
		logoLink = "https://35fba853288929fc2c78-69cb25cc8c90ed23a3699fb2b2ac841c.ssl.cf5.rackcdn.com/email/jakata.png"
		logoLinkWhite = "https://35fba853288929fc2c78-69cb25cc8c90ed23a3699fb2b2ac841c.ssl.cf5.rackcdn.com/email/jakata-white.png"
	case 2:
		salonName = "Paul Kemp Hairdressing"
		address = "info@paulkemphairdressing.com"
		logoLink = "https://35fba853288929fc2c78-69cb25cc8c90ed23a3699fb2b2ac841c.ssl.cf5.rackcdn.com/email/pk.png"
		logoLinkWhite = "https://35fba853288929fc2c78-69cb25cc8c90ed23a3699fb2b2ac841c.ssl.cf5.rackcdn.com/email/pk-white.png"
	case 3:
		salonName = "Base Hairdressing"
		address = "info@basehairdressing.com"
		logoLink = "https://35fba853288929fc2c78-69cb25cc8c90ed23a3699fb2b2ac841c.ssl.cf5.rackcdn.com/email/base.png"
		logoLinkWhite = "https://35fba853288929fc2c78-69cb25cc8c90ed23a3699fb2b2ac841c.ssl.cf5.rackcdn.com/email/base-white.png"
	default:
		http.Error(w, `{"error": "Invalid salon ID"}`, http.StatusBadRequest)
		return
	}

	var htmlTemplatePath, textTemplatePath string
	switch requestBody.EmailResponse { // Assuming FollowUp is used for email status
	case "unsuccessful":
		htmlTemplatePath = "templates/recruitment/unsuccessful.gohtml"
		textTemplatePath = "templates/recruitment/unsuccessful.txt"
	case "maybe":
		htmlTemplatePath = "templates/recruitment/maybe.gohtml"
		textTemplatePath = "templates/recruitment/maybe.txt"
	case "successful":
		htmlTemplatePath = "templates/recruitment/successful.gohtml"
		textTemplatePath = "templates/recruitment/successful.txt"
	default:
		debugInfo := map[string]string{
			"error":         "Invalid status",
			"emailResponse": applicant.EmailResponse,
			"followUp":      applicant.FollowUp,
		}

		log.Printf("Invalid status received: '%s'. Applicant FollowUp: '%s'", applicant.EmailResponse, applicant.FollowUp)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)

		if err := json.NewEncoder(w).Encode(debugInfo); err != nil {
			log.Printf("Failed to write error response: %v", err)
			http.Error(w, `{"error": "Failed to write error response"}`, http.StatusInternalServerError)
		}
		return
	}

	// Include additional data for the templates
	templateData := struct {
		JoinusApplicant
		SalonName     string
		LogoLink      string
		LogoLinkWhite string
	}{
		JoinusApplicant: applicant,
		SalonName:       salonName,
		LogoLink:        logoLink,
		LogoLinkWhite:   logoLinkWhite,
	}

	htmlContent, err := ParseEmailTemplate(htmlTemplatePath, templateData)
	if err != nil {
		http.Error(w, `{"error": "Failed to parse HTML email template", "details": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	textContent, err := ParseEmailTemplate(textTemplatePath, templateData)
	if err != nil {
		http.Error(w, `{"error": "Failed to parse text email template", "details": "`+err.Error()+`"}`, http.StatusInternalServerError)
		return
	}

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))
	sender := address
	subject := "Follow up on your application to " + salonName
	body := textContent
	recipient := applicant.Email
	m := mg.NewMessage(sender, subject, body, recipient)
	m.SetHtml(htmlContent)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	resp, id, err := mg.Send(ctx, m)
	if err != nil {
		http.Error(w, `{"error": "Failed to send email"}`, http.StatusInternalServerError)
		return
	}

	if err := DB.Model(&applicant).Update("EmailResponse", requestBody.EmailResponse).Error; err != nil {
		http.Error(w, `{"error": "Failed to update EmailResponse"}`, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"message": "Email Successfully sent", "ID": "%s", "Resp": "%s"}`, id, resp)
}

func apiJoinusUpdateRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}
	// Extract applicant ID from the URL parameters (using gorilla/mux)
	vars := mux.Vars(r)
	applicantID := vars["id"]
	// Find the applicant record by ID
	var applicant JoinusApplicant
	if result := DB.First(&applicant, applicantID); result.Error != nil {
		http.Error(w, "Applicant not found", http.StatusNotFound)
		return
	}
	// Update the role to "Saturday/Evening"
	if result := DB.Model(&applicant).Update("role", "Saturday"); result.Error != nil {
		http.Error(w, result.Error.Error(), http.StatusInternalServerError)
		return
	}

	// Respond with the updated applicant record
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(applicant); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

func apiModel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data ModelApplicant
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	DB.Create(&data)

	return
}

func apiBookingRequest(w http.ResponseWriter, r *http.Request) {
	var data BookingRequest
	var br BookingRequest
	var dbResponse Response

	json.NewDecoder(r.Body).Decode(&data)

	DB.Where("mobile", data.Mobile).First(&br)

	if data.Mobile == br.Mobile {
		dbResponse.Message = "You've already registered! We'll be in touch soon"
		responseJSON(w, dbResponse)
		return
	}
	DB.Create(&data)

	sendSms(data.Stylist)
	return
}

func apiBlogPost(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var blog Blog
	var fn string

	params := mux.Vars(r)

	files, err := ioutil.ReadDir("blog")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		if strings.Contains(f.Name(), params["slug"]) {
			fn = f.Name()
		}
	}

	data, err := ioutil.ReadFile("blog/" + fn)
	if err != nil {
		fmt.Println("File reading error", err)
		return
	}
	lines := strings.Split(string(data), "\n")
	title := lines[0]
	date := lines[1]
	author := lines[2]
	image := lines[3]
	intro := lines[6]
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

	files, err := ioutil.ReadDir("blog")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		today := time.Now()
		startFrom := today.Add(-8760 * time.Hour)
		split := strings.Split(f.Name(), " ")
		date, err := time.Parse("2006-01-02", split[0])
		if err != nil {
			panic(err)
		}
		if date.After(startFrom) && date.Before(today) {
			data, err := ioutil.ReadFile("blog/" + f.Name())
			if err != nil {
				fmt.Println("File reading error", err)
				return
			}
			slug := strings.Split(split[1], ".")[0]
			lines := strings.Split(string(data), "\n")
			title := lines[0]
			bDate := lines[1]
			author := lines[2]
			image := lines[3]
			intro := lines[6]
			text := strings.Join(lines[6:8], "\n")
			body := blackfriday.MarkdownBasic([]byte(text))

			blogs = append(blogs, Blog{Slug: slug, Date: bDate, Title: title, Image: image, Intro: intro, Author: author, Body: string(body)})
		}
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

	files, err := ioutil.ReadDir("blog")
	if err != nil {
		log.Fatal(err)
	}

	for _, f := range files {
		today := time.Now()
		startFrom := today.Add(-8760 * time.Hour)
		split := strings.Split(f.Name(), " ")
		date, err := time.Parse("2006-01-02", split[0])
		if err != nil {
			panic(err)
		}
		if date.After(startFrom) && date.Before(today) {
			data, err := ioutil.ReadFile("blog/" + f.Name())
			if err != nil {
				fmt.Println("File reading error", err)
				return
			}
			slug := strings.Split(split[1], ".")[0]
			lines := strings.Split(string(data), "\n")
			title := lines[0]
			image := lines[3]
			bDate := lines[1]
			text := lines[6]
			body := strings.Split(text, ".")

			blogs = append(blogs, Blog{Slug: slug, Date: bDate, Title: title, Image: image, Body: body[0]})
		}
	}
	sort.SliceStable(blogs, func(i, j int) bool { return blogs[i].Date > blogs[j].Date })

	if len(blogs) > 4 {
		blogs = blogs[:4]
	}

	json, err := json.Marshal(blogs)
	if err != nil {
		log.Panic(err)
	}
	w.Write(json)
}

func apiServices(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var p []Service

	DB.Find(&p)

	json, err := json.Marshal(p)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiStylists(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var s []TeamMember

	DB.Find(&s)

	json, err := json.Marshal(s)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiLevels(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var l []Level

	DB.Find(&l)

	json, err := json.Marshal(l)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiSalons(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var s []Salon

	DB.Find(&s)

	json, err := json.Marshal(s)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

func apiSaveQuoteDetails(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var salonURL, salonName, salonEmail, tplFolder string
	var data QuoteRespondent
	var quote QuoteInfo

	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	DB.Create(&data)

	sID := data.StylistSalonID

	switch sID {
	case 1:
		salonName = "Jakata Salon"
		salonURL = "https://www.jakatasalon.co.uk/"
		salonEmail = "info@jakatasalon.co.uk"
		tplFolder = "jakata"
	case 2:
		salonName = "Paul Kemp Hairdressing"
		salonURL = "https://www.paulkemphairdressing.com/"
		salonEmail = "info@paulkemphairdressing.com"
		tplFolder = "pk"
	case 3:
		salonName = "Base Hairdressing"
		salonURL = "https://www.basehairdressing.com/"
		salonEmail = "info@basehairdressing.com"
		tplFolder = "base"
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

	err = json.Unmarshal(data.Quote, &quote)
	if err != nil {
		log.Fatalln(err)
	}

	now := time.Now()
	month := now.AddDate(0, 0, 7*4)
	formatted := month.Format("02/01/2006")

	quote.Discount = quote.Total * .8
	quote.Expires = formatted

	htmlContent, err := ParseEmailTemplate("templates/"+tplFolder+"/quote.gohtml", quote)
	if err != nil {
		log.Fatalln(err)
	}

	textContent, err := ParseEmailTemplate("templates/"+tplFolder+"/quote.txt", quote)
	if err != nil {
		log.Fatalln(err)
	}

	mg := mailgun.NewMailgun(os.Getenv("MAILGUN_DOMAIN"), os.Getenv("MAILGUN_KEY"))

	sender := salonEmail
	subject := "Your quote for " + salonName
	body := textContent
	recipient := data.Email

	m := mg.NewMessage(sender, subject, body, recipient)

	m.SetHtml(htmlContent)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	// Send the message	with a 10 second timeout
	resp, id, err := mg.Send(ctx, m)

	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("ID: %s Resp: %s\n", id, resp)
}

func apiGetQuoteDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var services QuoteRespondent

	vars := mux.Vars(r)
	param := vars["link"]

	DB.Where("link", param).First(&services)

	json, err := json.Marshal(services)
	if err != nil {
		log.Println(err)
	}
	w.Write(json)
}

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

func apiOpenEvening(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data OpenEveningApplicant
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	DB.Create(&data)

	return
}

func apiFeedbackResult(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)

	var data FeedbackResult
	err := decoder.Decode(&data)
	if err != nil {
		panic(err)
	}

	DB.Create(&data)

	return
}

func apiStoreData(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	var storeData struct {
		Tiles   []OnlineStoreTile   `json:"tiles"`
		Banners []OnlineStoreBanner `json:"banners"`
	}

	var tiles []OnlineStoreTile
	var banners []OnlineStoreBanner

	if err := DB.Find(&tiles).Error; err != nil {
		http.Error(w, "Failed to retrieve tiles", http.StatusInternalServerError)
		log.Println("Error retrieving tiles:", err)
		return
	}

	if err := DB.Find(&banners).Error; err != nil {
		http.Error(w, "Failed to retrieve banners", http.StatusInternalServerError)
		log.Println("Error retrieving banners:", err)
		return
	}

	storeData.Tiles = tiles
	storeData.Banners = banners

	respData, err := json.Marshal(storeData)
	if err != nil {
		http.Error(w, "Failed to marshal data", http.StatusInternalServerError)
		log.Println("Error marshalling data:", err)
		return
	}
	w.Write(respData)
}
