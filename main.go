package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/julienschmidt/httprouter"
	"github.com/microcosm-cc/bluemonday"
	"github.com/russross/blackfriday/v2"
)

var header, footer *template.Template
var idRe = regexp.MustCompile("[a-z]{5}")

func main() {
	hB, err := ioutil.ReadFile("templates/header.html")
	if err != nil {
		panic(err)
	}
	header, err = template.New("header").Parse(string(hB))
	if err != nil {
		panic(err)
	}
	fB, err := ioutil.ReadFile("templates/footer.html")
	if err != nil {
		panic(err)
	}
	footer, err = template.New("footer").Parse(string(fB))
	if err != nil {
		panic(err)
	}

	router := httprouter.New()
	router.GET("/us", index)
	router.GET("/", index)
	router.GET("/sys/:system", otherSystem)
	router.GET("/simple/:system", simple)
	router.ServeFiles("/static/*filepath", http.Dir("static"))

	log.Fatal(http.ListenAndServe(":5000", router))
}

func otherSystem(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if p.ByName("system") == "" {
		fmt.Fprintf(w, "System ID was empty")
		return
	}
	fronter(w, r, p.ByName("system"))
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	fronter(w, r, "qvzbz")
}

func simple(w http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if p.ByName("system") == "" {
		fmt.Fprintf(w, "System ID was empty")
		return
	}
	if !idRe.MatchString(p.ByName("system")) {
		fmt.Fprintf(w, "System ID wasn't a valid 5-letter ID")
		return
	}
	id := p.ByName("system")

	tB, err := ioutil.ReadFile("templates/frontsimple.html")
	if err != nil {
		fmt.Fprintf(w, "Error when parsing template frontsimple.html: %v", err)
		log.Printf("Error when parsing template frontsimple.html: %v", err)
		return
	}
	tmpl, err := template.New("front").Parse(string(tB))
	if err != nil {
		fmt.Fprintf(w, "Error when parsing template frontsimple.html: %v", err)
		log.Printf("Error when parsing template frontsimple.html: %v", err)
		return
	}

	resp, err := http.Get("https://api.pluralkit.me/v1/s/" + id + "/fronters")
	if err != nil {
		fmt.Fprintf(w, "Error getting the current fronter: %v", err)
		log.Printf("Error getting the current fronter: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Fprintf(w, "Error when getting the current fronter: %v", resp.Status)
		log.Printf("Error when getting the current fronter: %v", resp.Status)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "Error reading the fronter info: %v", err)
		log.Printf("Error reading the fronter info: %v", err)
		return
	}
	var f front
	if err := json.Unmarshal(b, &f); err != nil {
		fmt.Fprintf(w, "Error unmarshaling fronter info: %v", err)
		log.Printf("Error unmarshaling fronter info: %v", err)
		return
	}

	info := struct {
		Out       bool
		Name      string
		Colour    string
		Pronouns  string
		Since     time.Time
		Others    string
		AvatarURL string
		Created   time.Time
	}{Since: f.Since}

	if len(f.Members) > 0 {
		info.Name = f.Members[0].Name
		info.Colour = f.Members[0].Color
		info.Pronouns = f.Members[0].Pronouns
		info.Created = f.Members[0].Created
		if info.Pronouns == "" {
			info.Pronouns = "unknown/not specified"
		}

		info.AvatarURL = f.Members[0].AvatarURL
		if info.AvatarURL == "" {
			info.AvatarURL = "https://fakeimg.pl/512x512/36393f/?text=%20"
		}

		if len(f.Members) > 1 {
			var members []string
			for _, m := range f.Members[1:] {
				members = append(members, m.Name)
			}
			info.Others = strings.Join(members, ", ")
		} else {
			info.Others = "None"
		}
	} else {
		info.Out = true
		info.AvatarURL = "https://fakeimg.pl/512x512/36393f/?text=%20"
	}

	tmpl.Execute(w, info)
}

func fronter(w http.ResponseWriter, r *http.Request, id string) {
	// get the template
	tB, err := ioutil.ReadFile("templates/fronter.html")
	if err != nil {
		fmt.Fprintf(w, "Error when parsing template fronter.html: %v", err)
		log.Printf("Error when parsing template fronter.html: %v", err)
		return
	}
	tmpl, err := template.New("footer").Parse(string(tB))
	if err != nil {
		fmt.Fprintf(w, "Error when parsing template fronter.html: %v", err)
		log.Printf("Error when parsing template fronter.html: %v", err)
		return
	}

	resp, err := http.Get("https://api.pluralkit.me/v1/s/" + id)
	if err != nil {
		fmt.Fprintf(w, "Error getting system: %v", err)
		log.Printf("Error getting system: %v", err)
		return
	}
	if resp.StatusCode != 200 {
		fmt.Fprintf(w, "Error getting system: %v", resp.Status)
		log.Printf("Error getting system: %v", resp.Status)
		return
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "Error reading system info: %v", err)
		log.Printf("Error reading system info: %v", err)
		return
	}
	var s system
	if err := json.Unmarshal(b, &s); err != nil {
		fmt.Fprintf(w, "Error unmarshaling system: %v", err)
		log.Printf("Error unmarshaling system: %v", err)
		return
	}
	var systemName string
	if s.Name != "" {
		systemName = s.Name
	} else {
		systemName = fmt.Sprintf("[no name] (ID: %v)", s.ID)
	}

	resp, err = http.Get("https://api.pluralkit.me/v1/s/" + id + "/fronters")
	if err != nil {
		fmt.Fprintf(w, "Error getting the current fronter: %v", err)
		log.Printf("Error getting the current fronter: %v", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Fprintf(w, "Error when getting the current fronter: %v", resp.Status)
		log.Printf("Error when getting the current fronter: %v", resp.Status)
		return
	}
	b, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(w, "Error reading the fronter info: %v", err)
		log.Printf("Error reading the fronter info: %v", err)
		return
	}
	var f front
	if err := json.Unmarshal(b, &f); err != nil {
		fmt.Fprintf(w, "Error unmarshaling fronter info: %v", err)
		log.Printf("Error unmarshaling fronter info: %v", err)
		return
	}
	info := struct {
		Page     pageInfo
		SysName  string
		SysID    string
		Fronters []member
	}{
		Page: pageInfo{
			PageTitle: "Currently fronting",
		},
		SysName: systemName,
		SysID:   s.ID,
	}
	if len(f.Members) > 0 {
		for _, m := range f.Members {
			if m.Birthday != "" {
				bd, err := time.Parse("2006-01-02", m.Birthday)
				if err != nil {
					fmt.Fprintf(w, "Error parsing birthday: %v", err)
					log.Printf("Error parsing birthday: %v", err)
					return
				}
				m.TimeBirthday = bd
			}
			m.HTMLDesc = template.HTML(
				bluemonday.UGCPolicy().SanitizeBytes(
					blackfriday.Run(
						[]byte(m.Description),
						blackfriday.WithExtensions(blackfriday.Autolink|blackfriday.Strikethrough|blackfriday.HardLineBreak))))
			info.Fronters = append(info.Fronters, m)
		}
	}
	header.Execute(w, info)
	tmpl.Execute(w, info)
	footer.Execute(w, info)
}
