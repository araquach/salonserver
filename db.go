package salonserver

import (
	"bufio"
	"encoding/csv"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/postgres"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func dbConn() (db *gorm.DB) {
	db, err := gorm.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		panic(err)
	}
	return db
}

func migrate() {
	db := dbConn()
	defer db.Close()
	db.LogMode(true)

	db.DropTableIfExists(&TeamMember{}, &MetaInfo{}, &Service{}, &Level{}, &Salon{})
	db.AutoMigrate(&TeamMember{}, &MetaInfo{}, &JoinusApplicant{}, &ModelApplicant{}, &Review{}, &BookingRequest{}, &Service{}, &Level{}, &Salon{})

	loadSalons()
	loadLevels()
	loadServices()
	loadTeamMembers()
	loadMetaInfo()
}

func loadSalons() {
	var err error
	var salons []Salon
	var files []string

	db := dbConn()
	db.LogMode(true)
	defer db.Close()

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
				Name: line[0],
				Logo: line[1],
				Image: line[2],
				Phone: line[3],
				Bookings: line[4],
			})
		}
	}
	for _, l := range salons {
		db = dbConn()
		db.LogMode(true)
		db.Create(&l)
		if err != nil {
			log.Panic(err)
		}
		db.Close()
	}
}

func loadLevels(){
	var err error
	var levels []Level
	var files []string

	db := dbConn()
	db.LogMode(true)
	defer db.Close()

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

			a, _ := strconv.Atoi(line[1])
			c, _ := strconv.Atoi(line[2])

			levels = append(levels, Level{
				Name: line[0],
				Adapter: a,
				ColAdapter: c,
			})
		}
	}
	for _, l := range levels {
		db = dbConn()
		db.LogMode(true)
		db.Create(&l)
		if err != nil {
			log.Panic(err)
		}
		db.Close()
	}
}

func loadServices(){
	var err error
	var services []Service
	var files []string

	db := dbConn()
	db.LogMode(true)
	defer db.Close()

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
			services = append(services, Service{
				Cat1: uint(c1),
				Cat2: uint(c2),
				Service: line[2],
				Price: p,
			})
		}
	}
	for _, s := range services {
		db = dbConn()
		db.LogMode(true)
		db.Create(&s)
		if err != nil {
			log.Panic(err)
		}
		db.Close()
	}
}

func loadTeamMembers() {
	var err error
	var teamMembers []TeamMember

	db := dbConn()
	db.LogMode(true)
	db.Close()

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
			level, _ := strconv.Atoi(line[3])
			price, _ := strconv.ParseFloat(line[13], 8)
			position, _ := strconv.Atoi(line[14])

			teamMembers = append(teamMembers, TeamMember{
				Salon: uint(salon),
				FirstName: line[1],
				LastName: line[2],
				Level: uint(level),
				LevelName: line[4],
				Image: line[5],
				RemoteImage: line[6],
				RemoteMontage: line[7],
				Para1: line[8],
				Para2: line[9],
				Para3: line[10],
				FavStyle: line[11],
				Product: line[12],
				Price: price,
				Position: uint(position),
				Slug: line[15],
			})
		}
	}

	for _, m := range teamMembers {
		db = dbConn()
		db.LogMode(true)
		db.Create(&m)
		if err != nil {
			log.Println(err)
		}
		db.Close()
	}
}

func loadMetaInfo() {
	var err error
	var metaInfos []MetaInfo

	db := dbConn()
	db.LogMode(true)
	db.Close()

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

			salon, _ := strconv.Atoi(line[4])
			metaInfos = append(metaInfos, MetaInfo{
				Salon: uint(salon),
				Page:  line[0],
				Title: line[1],
				Text:  line[2],
				Image: line[3],
			})
		}
	}

	for _, m := range metaInfos {
		db = dbConn()
		db.LogMode(true)
		db.Create(&m)
		if err != nil {
			log.Println(err)
		}
		db.Close()
	}
}
