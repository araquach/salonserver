package salonserver

import (
	"bufio"
	"encoding/csv"
	"io"
	"log"
	"os"
	"path/filepath"
	"strconv"
)

func Migrate() {
	DB.Migrator().DropTable(&TeamMember{}, &MetaInfo{}, &Level{}, &Salon{})
	DB.AutoMigrate(&TeamMember{}, &MetaInfo{}, &JoinusApplicant{}, &ModelApplicant{}, &Review{}, &BookingRequest{}, &Service{}, &Level{}, &Salon{})

	loadSalons()
	loadLevels()
	// loadServices()
	loadTeamMembers()
	loadMetaInfo()
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

			a, _ := strconv.Atoi(line[1])
			c, _ := strconv.Atoi(line[2])

			levels = append(levels, Level{
				Name:       line[0],
				Adapter:    a,
				ColAdapter: c,
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
			services = append(services, Service{
				Cat1:    uint(c1),
				Cat2:    uint(c2),
				Service: line[2],
				Price:   p,
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