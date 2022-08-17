package salonserver

import (
	"bytes"
	"html/template"
)

func ParseEmailTemplate(templateFileName string, data interface{}) (content string, err error) {

	tmpl, err := template.ParseFiles(templateFileName)
	if err != nil {
		return "", err
	}

	buf := new(bytes.Buffer)

	if err := tmpl.Execute(buf, data); err != nil {
		return "", err
	}

	return buf.String(), nil
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
