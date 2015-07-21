package main

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

var cookieeJar Cookiee

type Cookiee struct {
	cookies []*http.Cookie
}

func (this *Cookiee) String() string {
	var str string
	for _, cookiee := range this.cookies {
		str = str + fmt.Sprintf("%s=%s;", cookiee.Name, cookiee.Value)
	}
	return str
}

func (this *Cookiee) Set(cookiees string) error {
	for _, item := range strings.Split(cookiees, ";") {
		index := strings.Index(item, "=")
		if index <= 0 || index == (len(item)-1) {
			return fmt.Errorf("Invalid Cookiee")
		}
		newCookiee := &http.Cookie{
			Name:  item[:index],
			Value: item[index+1:],
			Path:  "/",
		}
		this.cookies = append(this.cookies, newCookiee)
	}
	return nil
}

func (this Cookiee) SetCookies(u *url.URL, cookies []*http.Cookie) {
	this.cookies = cookies
}

func (this Cookiee) Cookies(u *url.URL) []*http.Cookie {
	return this.cookies
}
