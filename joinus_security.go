package salonserver

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/mail"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"
)

// Only public application fields are decoded; administrative fields cannot be set.
type joinusRequest struct {
	Name     string `json:"name"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Position string `json:"position"`
	About    string `json:"about"`
	WhyHair  string `json:"why_hair"`
	WhyUs    string `json:"why_us"`
	Salon    uint   `json:"salon"`
	Website  string `json:"website"`
}

type joinusAttempt struct {
	count int
	until time.Time
}
type joinusLimiter struct {
	mu       sync.Mutex
	attempts map[string]joinusAttempt
}

var recruitmentLimiter = &joinusLimiter{attempts: make(map[string]joinusAttempt)}

// Fixed windows: five attempts per IP per 15 minutes, including rejected requests.
// Storage is bounded and expired entries are removed on each request.
func (l *joinusLimiter) allow(key string, now time.Time) (bool, int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	for k, v := range l.attempts {
		if !now.Before(v.until) {
			delete(l.attempts, k)
		}
	}
	v, exists := l.attempts[key]
	if !exists {
		if len(l.attempts) >= 10000 {
			return false, 900
		}
		v.until = now.Add(15 * time.Minute)
	}
	if v.count >= 5 {
		return false, int(v.until.Sub(now).Seconds()) + 1
	}
	v.count++
	l.attempts[key] = v
	return true, 0
}

func recruitmentIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	peer := net.ParseIP(host)
	// Forwarded headers are trusted only from explicitly configured proxy networks.
	for _, cidr := range strings.Split(os.Getenv("JOINUS_TRUSTED_PROXY_CIDRS"), ",") {
		_, network, err := net.ParseCIDR(strings.TrimSpace(cidr))
		if err != nil || peer == nil || !network.Contains(peer) {
			continue
		}
		chain := strings.Split(r.Header.Get("X-Forwarded-For"), ",")
		for i := len(chain) - 1; i >= 0; i-- {
			ip := net.ParseIP(strings.TrimSpace(chain[i]))
			if ip == nil {
				return host
			}
			trusted := false
			for _, entry := range strings.Split(os.Getenv("JOINUS_TRUSTED_PROXY_CIDRS"), ",") {
				_, n, e := net.ParseCIDR(strings.TrimSpace(entry))
				if e == nil && n.Contains(ip) {
					trusted = true
					break
				}
			}
			if !trusted {
				return ip.String()
			}
		}
		return host
	}
	return host
}

func recruitmentError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(Response{Message: message})
}

func readRecruitment(w http.ResponseWriter, r *http.Request) (JoinusApplicant, bool) {
	var empty JoinusApplicant
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", "POST")
		recruitmentError(w, 405, "Please submit the application using POST.")
		return empty, false
	}
	if allowed, retry := recruitmentLimiter.allow(recruitmentIP(r), time.Now()); !allowed {
		w.Header().Set("Retry-After", strconv.Itoa(retry))
		recruitmentError(w, 429, "Too many attempts. Please wait before trying again.")
		return empty, false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	decoder := json.NewDecoder(r.Body)
	var request joinusRequest
	if err := decoder.Decode(&request); err != nil {
		recruitmentError(w, 400, "Please submit a valid application.")
		return empty, false
	}
	if decoder.Decode(&struct{}{}) != io.EOF {
		recruitmentError(w, 400, "Please submit a single application.")
		return empty, false
	}
	data := JoinusApplicant{Name: strings.TrimSpace(request.Name), Mobile: strings.TrimSpace(request.Mobile), Email: strings.TrimSpace(request.Email), Role: strings.TrimSpace(request.Role), Position: strings.TrimSpace(request.Position), About: strings.TrimSpace(request.About), WhyHair: strings.TrimSpace(request.WhyHair), WhyUs: strings.TrimSpace(request.WhyUs), Salon: request.Salon}
	if request.Website != "" || !validRecruitment(data) {
		recruitmentError(w, 422, "Please check your name, phone and email, check your application details, and provide written answers of 10 to 5,000 characters.")
		return empty, false
	}
	return data, true
}

func validRecruitment(d JoinusApplicant) bool {
	length := func(s string, min, max int) bool {
		n := utf8.RuneCountInString(s)
		return utf8.ValidString(s) && n >= min && n <= max
	}
	hasLetter := func(s string) bool {
		for _, r := range s {
			if unicode.IsLetter(r) {
				return true
			}
		}
		return false
	}
	if !length(d.Name, 2, 100) || !hasLetter(d.Name) || strings.ContainsAny(d.Name+d.Email, "\r\n") {
		return false
	}
	address, err := mail.ParseAddress(d.Email)
	if err != nil || address.Address != d.Email || !length(d.Email, 3, 254) || !strings.Contains(strings.SplitN(d.Email, "@", 2)[1], ".") {
		return false
	}
	digits := 0
	for _, r := range d.Mobile {
		if r >= '0' && r <= '9' {
			digits++
		} else if !strings.ContainsRune("+() .-", r) {
			return false
		}
	}
	if digits < 10 || digits > 15 || !length(d.Mobile, 10, 30) {
		return false
	}
	if !validRecruitmentSelection(d.Salon, d.Role, d.Position) {
		return false
	}
	for _, answer := range []string{d.About, d.WhyHair, d.WhyUs} {
		if !length(answer, 10, 5000) || !hasLetter(answer) {
			return false
		}
	}
	return true
}

// Base has independent role and position dropdowns. The other sites use
// separate role-specific forms; Saturday submits a fixed position with no dropdown.
func validRecruitmentSelection(salonID uint, role, position string) bool {
	contains := func(value string, options ...string) bool {
		for _, option := range options {
			if value == option {
				return true
			}
		}
		return false
	}
	if salonID == 3 {
		return contains(role, "Stylist", "Apprentice", "Saturday", "Other") &&
			contains(position, "at school/college", "new to hairdressing", "partway through", "already qualified", "other")
	}
	if salonID != 1 && salonID != 2 {
		return false
	}
	switch role {
	case "Stylist":
		return contains(position, "less than one year", "1 - 3 years", "3 - 5 years", "5 plus years")
	case "Apprentice":
		return contains(position, "new to hairdressing", "partway through", "already qualified", "other")
	case "Saturday":
		return position == "new to hairdressing"
	default:
		return false
	}
}
