package salonserver

import (
	"github.com/jinzhu/gorm"
	"gorm.io/datatypes"
	"time"
)

type Salon struct {
	Id       uint   `json:"id" gorm:"primaryKey"`
	Name     string `json:"name"`
	Logo     string `json:"logo"`
	Image    string `json:"image"`
	Phone    string `json:"phone"`
	Bookings string `json:"bookings"`
}

type Service struct {
	Id           uint    `json:"id" gorm:"primary_key"`
	Cat1         uint    `json:"cat1"`
	Cat2         uint    `json:"cat2"`
	Service      string  `json:"service"`
	Price        float64 `json:"price"`
	ProductPrice float64 `json:"product_price"`
}

type Level struct {
	Id      uint   `json:"id" gorm:"primary_key"`
	Name    string `json:"name"`
	Adapter int    `json:"adapter"`
}

type ContactMessage struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Message string `json:"message"`
}

type JoinusApplicant struct {
	gorm.Model
	Salon    uint   `json:"salon"`
	Role     string `json:"role"`
	Name     string `json:"name"`
	Mobile   string `json:"mobile"`
	Email    string `json:"email"`
	Position string `json:"position"`
	About    string `json:"about"`
	WhyHair  string `json:"why_hair"`
	WhyUs    string `json:"why_us"`
}

type ModelApplicant struct {
	gorm.Model
	Name   string `json:"name"`
	Mobile string `json:"mobile"`
	Info   string `json:"info"`
}

type TeamMember struct {
	ID            uint    `json:"id" gorm:"primary_key"`
	Salon         uint    `json:"salon"`
	StaffId       uint    `json:"staff_id"`
	FirstName     string  `json:"first_name"`
	LastName      string  `json:"last_name"`
	Level         uint    `json:"level"`
	LevelName     string  `json:"level_name"`
	Image         string  `json:"image"`
	RemoteImage   string  `json:"remote_image"`
	RemoteMontage string  `json:"remote_montage"`
	Para1         string  `json:"para_1"`
	Para2         string  `json:"para_2"`
	Para3         string  `json:"para_3"`
	FavStyle      string  `json:"fav_style"`
	Product       string  `json:"product"`
	Price         float64 `json:"price"`
	Position      uint    `json:"position"`
	Slug          string  `json:"slug"`
}

type Review struct {
	ID      uint      `json:"id" gorm:"primary_key"`
	Date    time.Time `json:"date"`
	Salon   uint      `json:"salon"`
	Review  string    `json:"review"`
	Client  string    `json:"client"`
	Stylist string    `json:"stylist"`
}

type MetaInfo struct {
	ID    uint   `json:"id" gorm:"primary_key"`
	Salon uint   `json:"salon"`
	Page  string `json:"page"`
	Title string `json:"title"`
	Text  string `json:"text"`
	Image string `json:"image"`
}

type BookingRequest struct {
	ID        uint   `json:"id" gorm:"primary_key"`
	Salon     uint   `json:"salon"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Mobile    string `json:"mobile"`
	Stylist   string `json:"stylist"`
	TimeSlot  string `json:"time_slot"`
}

type Blog struct {
	Slug   string `json:"slug"`
	Date   string `json:"date"`
	Title  string `json:"title"`
	Image  string `json:"image"`
	Intro  string `json:"intro"`
	Body   string `json:"body"`
	Author string `json:"author"`
}

type QuoteRespondent struct {
	ID             uint           `json:"id" gorm:"primaryKey"`
	Link           string         `json:"link"`
	CreatedAt      time.Time      `json:"created_at"`
	SalonID        uint           `json:"salon_id"`
	StylistSalonID uint           `json:"stylist_salon_id"`
	Name           string         `json:"name"`
	Mobile         string         `json:"mobile"`
	Email          string         `json:"email"`
	Referral       string         `json:"referral"`
	Regular        bool           `json:"regular"`
	Selector       string         `json:"selector"`
	Quote          datatypes.JSON `json:"quote"`
}

type QuoteInfo struct {
	Services []struct {
		Price   float64 `json:"price"`
		Service string  `json:"service"`
	} `json:"services"`
	Stylist struct {
		Image string `json:"image"`
		Name  string `json:"name"`
	} `json:"stylist"`
	Total    float64 `json:"total"`
	Discount float64 `json:"discount"`
	Regular  bool    `json:"regular"`
	Expires  string  `json:"expires"`
	Status   int     `json:"status"`
}
