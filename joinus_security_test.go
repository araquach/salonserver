package salonserver

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func genuineApplication() JoinusApplicant {
	return JoinusApplicant{Name: "Zoë O'Neill", Mobile: "+44 (0)7700 900123", Email: "zoe@example.com", Role: "Apprentice", Position: "at school/college", About: "I am finishing school.", WhyHair: "I enjoy styling hair.", WhyUs: "I like your training.", Salon: 3}
}

func TestRecruitmentValidation(t *testing.T) {
	if !validRecruitment(genuineApplication()) {
		t.Fatal("genuine applicant rejected")
	}
	cases := map[string]func(*JoinusApplicant){
		"default role":     func(d *JoinusApplicant) { d.Role = "default" },
		"default position": func(d *JoinusApplicant) { d.Position = "default" },
		"unknown role":     func(d *JoinusApplicant) { d.Role = "admin" },
		"numeric answer":   func(d *JoinusApplicant) { d.About = "5426154862" },
		"short answer":     func(d *JoinusApplicant) { d.WhyUs = "yes" },
		"long answer":      func(d *JoinusApplicant) { d.WhyHair = strings.Repeat("a", 5001) },
		"bad email":        func(d *JoinusApplicant) { d.Email = "not-an-email" },
		"header injection": func(d *JoinusApplicant) { d.Name = "Alice\r\nBcc: spam@example.com" },
		"short phone":      func(d *JoinusApplicant) { d.Mobile = "123" },
		"bad salon":        func(d *JoinusApplicant) { d.Salon = 0 },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			d := genuineApplication()
			mutate(&d)
			if validRecruitment(d) {
				t.Fatal("invalid application accepted")
			}
		})
	}
}

// Calling the actual handler with DB unset proves rejected requests stop before
// database access, template rendering and either notification email.
func TestRecruitmentRejections(t *testing.T) {
	cases := []struct {
		method, body string
		status       int
	}{
		{"GET", "", 405}, {"POST", "{", 400}, {"POST", "{} {}", 400},
		{"POST", `{"role":"default","position":"default"}`, 422},
		{"POST", `{"website":"https://spam.example"}`, 422},
		{"POST", strings.Repeat(" ", 65537), 400},
	}
	for _, c := range cases {
		recruitmentLimiter = &joinusLimiter{attempts: make(map[string]joinusAttempt)}
		r := httptest.NewRequest(c.method, "/api/joinus", strings.NewReader(c.body))
		w := httptest.NewRecorder()
		apiJoinus(w, r)
		if w.Code != c.status {
			t.Fatalf("got %d want %d", w.Code, c.status)
		}
	}
}

func TestRecruitmentRateLimit(t *testing.T) {
	l := &joinusLimiter{attempts: make(map[string]joinusAttempt)}
	now := time.Now()
	for i := 0; i < 5; i++ {
		if ok, _ := l.allow("ip", now); !ok {
			t.Fatal("allowed attempt rejected")
		}
	}
	if ok, retry := l.allow("ip", now); ok || retry < 900 {
		t.Fatal("limit not enforced")
	}
	if ok, _ := l.allow("other", now); !ok {
		t.Fatal("different IP rejected")
	}
	if ok, _ := l.allow("ip", now.Add(15*time.Minute)); !ok {
		t.Fatal("window did not reset")
	}
	recruitmentLimiter = l
	r := httptest.NewRequest("POST", "/api/joinus", strings.NewReader("{}"))
	r.RemoteAddr = "192.0.2.1:1234"
	for i := 0; i < 5; i++ {
		l.allow("192.0.2.1", now)
	}
	w := httptest.NewRecorder()
	apiJoinus(w, r)
	if w.Code != 429 || w.Header().Get("Retry-After") == "" {
		t.Fatal("missing HTTP rate limit response")
	}
}

func TestRecruitmentProxyTrust(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/joinus", nil)
	r.RemoteAddr = "127.0.0.1:1234"
	r.Header.Set("X-Forwarded-For", "198.51.100.2, 10.0.0.1")
	t.Setenv("JOINUS_TRUSTED_PROXY_CIDRS", "")
	if recruitmentIP(r) != "127.0.0.1" {
		t.Fatal("untrusted forwarding accepted")
	}
	t.Setenv("JOINUS_TRUSTED_PROXY_CIDRS", "127.0.0.0/8,10.0.0.0/8")
	if recruitmentIP(r) != "198.51.100.2" {
		t.Fatal("client IP not resolved")
	}
}

func TestRecruitmentHoneypotAndPublicFields(t *testing.T) {
	payload, _ := json.Marshal(genuineApplication())
	for _, honeypot := range []bool{false, true} {
		recruitmentLimiter = &joinusLimiter{attempts: make(map[string]joinusAttempt)}
		var fields map[string]interface{}
		json.Unmarshal(payload, &fields)
		fields["id"] = 999
		fields["email_response"] = "injected"
		if honeypot {
			fields["website"] = "bot-filled"
		}
		body, _ := json.Marshal(fields)
		r := httptest.NewRequest("POST", "/api/joinus", strings.NewReader(string(body)))
		w := httptest.NewRecorder()
		if honeypot {
			apiJoinus(w, r)
			if w.Code != 422 {
				t.Fatal("honeypot accepted")
			}
		} else {
			data, valid := readRecruitment(w, r)
			if !valid || data.ID != 0 || data.EmailResponse != "" {
				t.Fatal("public field restriction failed")
			}
		}
	}
}

func TestOtherSitesRecruitmentOptions(t *testing.T) {
	for _, salonID := range []uint{1, 2} {
		for _, position := range []string{"less than one year", "1 - 3 years", "3 - 5 years", "5 plus years"} {
			d := genuineApplication()
			d.Salon = salonID
			d.Role = "Stylist"
			d.Position = position
			if !validRecruitment(d) {
				t.Fatalf("salon %d stylist position %q rejected", salonID, position)
			}
		}
		d := genuineApplication()
		d.Salon = salonID
		d.Role = "Saturday"
		d.Position = "new to hairdressing"
		if !validRecruitment(d) {
			t.Fatal("Saturday application rejected")
		}
	}
}

func TestRecruitmentFormPatterns(t *testing.T) {
	basePositions := []string{"at school/college", "new to hairdressing", "partway through", "already qualified", "other"}
	for _, role := range []string{"Stylist", "Apprentice", "Saturday", "Other"} {
		for _, position := range basePositions {
			d := genuineApplication()
			d.Role = role
			d.Position = position
			if !validRecruitment(d) {
				t.Fatalf("Base combination rejected: %s / %s", role, position)
			}
		}
	}
	for _, salonID := range []uint{1, 2} {
		for _, position := range []string{"new to hairdressing", "partway through", "already qualified", "other"} {
			d := genuineApplication()
			d.Salon = salonID
			d.Position = position
			if !validRecruitment(d) {
				t.Fatalf("apprentice position rejected: %d / %s", salonID, position)
			}
		}
	}
	for _, salonID := range []uint{1, 2, 3} {
		for _, role := range []string{"Stylist", "Apprentice", "Saturday"} {
			for _, position := range []string{"", "default", "invented"} {
				d := genuineApplication()
				d.Salon = salonID
				d.Role = role
				d.Position = position
				if validRecruitment(d) {
					t.Fatalf("invalid position accepted: %d / %s / %s", salonID, role, position)
				}
			}
		}
	}
	invalid := []JoinusApplicant{
		{Salon: 1, Role: "Stylist", Position: "partway through"},
		{Salon: 2, Role: "Apprentice", Position: "5 plus years"},
		{Salon: 1, Role: "Saturday", Position: "already qualified"},
		{Salon: 2, Role: "Other", Position: "other"},
		{Salon: 3, Role: "Stylist", Position: "5 plus years"},
	}
	for _, d := range invalid {
		if validRecruitmentSelection(d.Salon, d.Role, d.Position) {
			t.Fatalf("wrong form combination accepted: %+v", d)
		}
	}
}

func TestRecruitmentHerokuProxy(t *testing.T) {
	t.Setenv("JOINUS_PROXY_MODE", "heroku")
	t.Setenv("JOINUS_TRUSTED_PROXY_CIDRS", "")
	for _, tc := range []struct {
		name    string
		headers []string
		want    string
	}{
		{"client", []string{"198.51.100.10"}, "198.51.100.10"},
		{"spoofed prefix ignored", []string{"203.0.113.99, 198.51.100.10"}, "198.51.100.10"},
		{"duplicate headers", []string{"203.0.113.99", "198.51.100.10"}, "198.51.100.10"},
		{"IPv6", []string{"203.0.113.99, 2001:db8::10"}, "2001:db8::10"},
		{"missing header", nil, "10.1.2.3"},
		{"malformed final IP", []string{"198.51.100.10, invalid"}, "10.1.2.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			r := httptest.NewRequest("POST", "/api/joinus", nil)
			r.RemoteAddr = "10.1.2.3:4321"
			for _, value := range tc.headers {
				r.Header.Add("X-Forwarded-For", value)
			}
			if got := recruitmentIP(r); got != tc.want {
				t.Fatalf("got %s, want %s", got, tc.want)
			}
		})
	}
	t.Setenv("JOINUS_PROXY_MODE", "")
	r := httptest.NewRequest("POST", "/api/joinus", nil)
	r.RemoteAddr = "127.0.0.1:4321"
	r.Header.Set("X-Forwarded-For", "198.51.100.10")
	if recruitmentIP(r) != "127.0.0.1" {
		t.Fatal("local requests trusted forwarded header without opt-in")
	}
}
