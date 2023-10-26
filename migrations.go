package salonserver

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

var tiles []OnlineStoreTile
var banners []OnlineStoreBanner

func Migrate() {
	err := DB.Migrator().DropTable(&TeamMember{}, &MetaInfo{}, &Level{}, &Salon{}, &Service{}, &Review{}, &OnlineStoreBanner{}, &OnlineStoreTile{})
	if err != nil {
		log.Fatalf("Failed to seed the data: %v", err)
		return
	}
	err = DB.AutoMigrate(&TeamMember{}, &MetaInfo{}, &JoinusApplicant{}, &ModelApplicant{}, &Review{}, &BookingRequest{}, &Service{}, &Level{}, &Salon{}, &QuoteRespondent{}, &OpenEveningApplicant{}, &FeedbackResult{}, &Review{}, &OnlineStoreBanner{}, &OnlineStoreTile{})
	if err != nil {
		log.Fatalf("Failed to seed the data: %v", err)
	}

	loadSalons()
	loadLevels()
	loadServices()
	loadTeamMembers()
	loadMetaInfo()
	loadReviews()
	loadTiles()
	loadBanners()
}

func loadReviews() {
	var files []string

	root := "data/csv/reviews"

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".csv" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, file := range files {

		reviews, err := divideByBlock(file)
		if err != nil {
			fmt.Println("Error:", err)
			return
		}

		for _, review := range reviews {
			s := filepath.Base(file)
			salon := strings.Split(s, "_")[0]
			salonId, err := strconv.Atoi(salon)
			if err != nil {
				log.Fatalf("Couldnt convert to integer: %v", err)
			}

			//fmt.Printf("Salon: %v\n", salonId)
			//fmt.Printf("Rating: %d\n", review.Rating)
			//fmt.Printf("Date: %s\n", review.Date)
			//fmt.Printf("Stylist: %s\n", review.Stylist)
			//fmt.Printf("Client: %s\n", review.Client)
			//fmt.Printf("Review: %s\n", review.Review)
			//fmt.Println("")

			review.Salon = uint(salonId)

			DB.Create(&review)
		}
	}
}

func parseBlock(block []string) (*Review, error) {
	headerPattern := regexp.MustCompile(`^([1-5]),,(\d{2}/\d{2}/\d{2}),,([^,]+),,`)
	footerPattern := regexp.MustCompile(`^- (.*)`)
	reviewPattern := regexp.MustCompile(`"""(.*?)"""`)

	var review Review

	// Parse header (first line of block)
	headerSubmatch := headerPattern.FindStringSubmatch(block[0])
	if len(headerSubmatch) < 4 {
		return nil, fmt.Errorf("unable to parse header: %s", block[0])
	}

	// Rating
	rating := headerSubmatch[1]
	review.Rating, _ = strconv.Atoi(rating)

	// Date
	review.Date = headerSubmatch[2]

	// Stylist
	review.Stylist = headerSubmatch[3]

	// Parse footer (last line of block)
	footerSubmatch := footerPattern.FindStringSubmatch(block[len(block)-1])
	if len(footerSubmatch) < 2 {
		return nil, fmt.Errorf("unable to parse footer: %s", block[len(block)-1])
	}

	// Client
	review.Client = strings.TrimRight(footerSubmatch[1], ",")

	// Join all lines except header and footer in the block, then use regex to find and extract review text
	blockText := strings.Join(block[1:len(block)-1], "\n")
	reviewMatch := reviewPattern.FindStringSubmatch(blockText)

	if len(reviewMatch) > 0 {
		// If a triple-quote delimited review was found, use it
		review.Review = reviewMatch[1]
	} else {
		// Split by commas and newlines, then rejoin only the parts within triple quotes
		fields := strings.FieldsFunc(blockText, func(r rune) bool {
			return r == ',' || r == '\n'
		})
		parts := []string{}
		for _, field := range fields {
			// Only keep parts within triple quotes
			if strings.HasPrefix(field, `"`) && strings.HasSuffix(field, `"`) {
				parts = append(parts, field)
			}
		}
		// Join parts back into a single string, excluding the quotation marks
		joinedParts := strings.Join(parts, " ")
		if len(joinedParts) > 0 {
			review.Review = joinedParts[1 : len(joinedParts)-1]
		} else {
			review.Review = ""
		}
	}

	return &review, nil
}

func divideByBlock(filename string) ([]Review, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	endPattern := regexp.MustCompile(`^- .*`)

	var reviews []Review
	block := make([]string, 0)

	for scanner.Scan() {
		line := scanner.Text()

		// Append the line to the current block
		block = append(block, line)

		// If the line matches the end pattern, it's the end of a block
		if endPattern.MatchString(line) {
			// Parse the block and add the review to the list
			review, err := parseBlock(block)
			if err != nil {
				return nil, err
			}
			reviews = append(reviews, *review)

			// Start a new block
			block = make([]string, 0)
		}
	}

	// Handle scan error
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return reviews, nil
}

func loadSalons() {
	var err error
	var salons []Salon
	var files []string

	root := "data/csv/salons"

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".csv" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		csvFile, _ := os.Open(file)
		reader := csv.NewReader(bufio.NewReader(csvFile))
		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}
			salons = append(salons, Salon{
				Name:     line[0],
				Logo:     line[1],
				Image:    line[2],
				Phone:    line[3],
				Bookings: line[4],
			})
		}
	}
	for _, l := range salons {
		DB.Create(&l)
	}
}

func loadLevels() {
	var err error
	var levels []Level
	var files []string

	root := "data/csv/levels"

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".csv" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		csvFile, _ := os.Open(file)
		reader := csv.NewReader(bufio.NewReader(csvFile))
		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}

			a, _ := strconv.ParseFloat(line[1], 8)

			levels = append(levels, Level{
				Name:    line[0],
				Adapter: a,
			})
		}
	}
	for _, l := range levels {
		DB.Create(&l)
	}
}

func loadServices() {
	var err error
	var services []Service
	var files []string

	root := "data/csv/services"

	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		if filepath.Ext(path) != ".csv" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, file := range files {
		csvFile, _ := os.Open(file)
		reader := csv.NewReader(bufio.NewReader(csvFile))
		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}
			c1, _ := strconv.Atoi(line[0])
			c2, _ := strconv.Atoi(line[1])
			p, _ := strconv.ParseFloat(line[3], 8)
			pp, _ := strconv.ParseFloat(line[4], 8)
			services = append(services, Service{
				Cat1:         uint(c1),
				Cat2:         uint(c2),
				Service:      line[2],
				Price:        p,
				ProductPrice: pp,
			})
		}
	}
	for _, s := range services {
		DB.Create(&s)
	}
}

func loadTeamMembers() {
	var err error
	var teamMembers []TeamMember

	var files []string

	root := "data/csv/team_members"
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})

	if err != nil {
		panic(err)
	}
	for _, file := range files {
		csvFile, _ := os.Open(file)

		reader := csv.NewReader(bufio.NewReader(csvFile))
		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}
			salon, _ := strconv.Atoi(line[0])
			staffId, _ := strconv.Atoi(line[1])
			level, _ := strconv.Atoi(line[4])
			price, _ := strconv.ParseFloat(line[14], 8)
			position, _ := strconv.Atoi(line[15])

			teamMembers = append(teamMembers, TeamMember{
				Salon:         uint(salon),
				StaffId:       uint(staffId),
				FirstName:     line[2],
				LastName:      line[3],
				Level:         uint(level),
				LevelName:     line[5],
				Image:         line[6],
				RemoteImage:   line[7],
				RemoteMontage: line[8],
				Para1:         line[9],
				Para2:         line[10],
				Para3:         line[11],
				FavStyle:      line[12],
				Product:       line[13],
				Price:         price,
				Position:      uint(position),
				Slug:          line[16],
			})
		}
	}

	for _, m := range teamMembers {
		DB.Create(&m)
	}
}

func loadMetaInfo() {
	var err error
	var metaInfos []MetaInfo

	var files []string

	root := "data/csv/meta_infos"
	err = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		files = append(files, path)
		return nil
	})

	if err != nil {
		panic(err)
	}
	for _, file := range files {
		csvFile, _ := os.Open(file)

		reader := csv.NewReader(bufio.NewReader(csvFile))
		for {
			line, error := reader.Read()
			if error == io.EOF {
				break
			} else if error != nil {
				log.Fatal(error)
			}

			salon, _ := strconv.Atoi(line[0])
			metaInfos = append(metaInfos, MetaInfo{
				Salon: uint(salon),
				Page:  line[1],
				Title: line[2],
				Text:  line[3],
				Image: line[4],
			})
		}
	}
	for _, m := range metaInfos {
		DB.Create(&m)
	}
}

func loadTiles() {
	file := "data/csv/store/tiles-Table 1.csv"
	csvFile, _ := os.Open(file)

	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}

		tiles = append(tiles, OnlineStoreTile{
			Brand:       line[0],
			Description: line[1],
			Image:       line[2],
			Url:         line[3],
		})
	}
	DB.Create(&tiles)
}

func loadBanners() {
	file := "data/csv/store/banners-Table 1.csv"
	csvFile, _ := os.Open(file)

	reader := csv.NewReader(bufio.NewReader(csvFile))
	for {
		line, error := reader.Read()
		if error == io.EOF {
			break
		} else if error != nil {
			log.Fatal(error)
		}

		banners = append(banners, OnlineStoreBanner{
			Image: line[0],
			Url:   line[1],
		})
	}
	DB.Create(&banners)
}
