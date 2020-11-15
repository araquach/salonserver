package main

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

	db.DropTableIfExists(&TeamMember{}, &MetaInfo{})
	db.AutoMigrate(&TeamMember{}, &MetaInfo{}, &JoinusApplicant{}, &ModelApplicant{}, &Review{}, &BookingRequest{})

	loadTeamMembers()
	loadMetaInfo()
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
			price, _ := strconv.ParseFloat(line[12], 8)
			position, _ := strconv.Atoi(line[13])

			teamMembers = append(teamMembers, TeamMember{
				Salon: uint(salon),
				FirstName: line[1],
				LastName: line[2],
				Level: uint(level),
				LevelName: line[4],
				Image: line[5],
				RemoteImage: line[6],
				Para1: line[7],
				Para2: line[8],
				Para3: line[9],
				FavStyle: line[10],
				Product: line[11],
				Price: price,
				Position: uint(position),
				Slug: line[14],
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
