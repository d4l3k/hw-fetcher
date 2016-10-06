package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
	"sync"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
	"github.com/microcosm-cc/bluemonday"
	"github.com/pkg/errors"
)

type Assignment struct {
	Name, Comment string
	Due, Late     string
}

type course struct {
	Course     string
	CourseURL  string
	CoursePage template.HTML
	Handin     []Assignment
	Errors     []error
}

const handinURL = "https://www.ugrad.cs.ubc.ca/~q7w9a/handin.cgi"

func fetchHandin() (map[string][]Assignment, error) {
	resp, err := http.Get(handinURL)
	m := make(map[string][]Assignment)
	if err != nil {
		return m, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&m); err != nil {
		return m, err
	}
	return m, nil
}

func getClasses() ([]string, error) {
	m, err := fetchHandin()
	if err != nil {
		return nil, err
	}
	for courseCode := range classFuncs {
		m[courseCode] = nil
	}
	var classes []string
	for c := range m {
		classes = append(classes, c)
	}
	sort.Strings(classes)
	return classes, nil
}

const cs304URL = "http://www.ugrad.cs.ubc.ca/~cs304/2016W1/schedule.html"

func fetchCS304() (string, error) {
	doc, err := goquery.NewDocument(cs304URL)
	if err != nil {
		return "", err
	}
	dom := doc.Find("table")
	makeAbsolute(dom, cs340URL)
	return goquery.OuterHtml(dom)
}

const cs340URL = "https://www.cs.ubc.ca/~schmidtm/Courses/340-F16/"

func fetchCS340() (string, error) {
	doc, err := goquery.NewDocument(cs340URL)
	if err != nil {
		return "", err
	}
	dom := doc.Find("table")
	makeAbsolute(dom, cs340URL)
	return goquery.OuterHtml(dom)
}

const cs311URL = "https://www.ugrad.cs.ubc.ca/~cs311/2016W1/_homework.php"

func fetchCS311() (string, error) {
	doc, err := goquery.NewDocument(cs311URL)
	if err != nil {
		return "", err
	}

	dom := doc.Find("table[rules]")
	if err := makeAbsolute(dom, cs311URL); err != nil {
		return "", err
	}
	return goquery.OuterHtml(dom)
}

const piazzaLoginURL = `https://piazza.com/account/login`
const cs313URL = "https://piazza.com/ubc.ca/winterterm12016/cpsc313/resources"

func fetchCS313() (string, error) {
	bow := surf.NewBrowser()
	if err := bow.Open(piazzaLoginURL); err != nil {
		return "", err
	}

	// Log in to the site.
	fm, err := bow.Form("form#login-form")
	if err != nil {
		return "", err
	}
	if err := fm.Input("email", *piazzaUser); err != nil {
		return "", err
	}
	if err := fm.Input("password", *piazzaPass); err != nil {
		return "", err
	}
	if err := fm.Submit(); err != nil {
		return "", err
	}
	if err := bow.Open(cs313URL); err != nil {
		return "", err
	}
	body := ""
	bow.Find("script").Each(func(_ int, s *goquery.Selection) {
		text := s.Text()
		parts := strings.Split(text, "this.resource_data        = ")
		if len(parts) != 2 {
			return
		}
		body = strings.Split(parts[1], ";\n")[0]
	})
	/*
	   {
	   	"content":"https://www.facebook.com/notes/facebook-engineering/the-full-stack-part-i/461505383919",
	   	"subject":"Reading Sep 8: The Full Stack Part 1",
	   	"created":"2016-09-06T20:32:57Z",
	   	"id":"isrxno834nx6x2",
	   	"config":{
	   		"resource_type":"link",
	   		"section":"general",
	   		"date":""
	   	}
	   }
	*/
	data := []struct {
		Content string `json:"content"`
		Subject string `json:"subject"`
		Created string `json:"created"`
		ID      string `json:"id"`
		Config  struct {
			ResourceType string `json:"resource_type"`
			Section      string `json:"section"`
			Date         string `json:"date"`
		} `json:"config"`
	}{}
	if err := json.Unmarshal([]byte(body), &data); err != nil {
		return "", err
	}
	html := `<table>
<thead>
<tr>
<th>Assignment</th>
<th>Out</th>
<th>Due</th>
</tr>
</thead>
<tbody>`
	for _, resource := range data {
		if resource.Config.Section != "homework" {
			continue
		}
		html += fmt.Sprintf(`<tr>
<td><a href="%s">%s</a></td>
<td>%s</td>
<td>%s</td>`, resource.Content, resource.Subject, resource.Created, resource.Config.Date)
	}
	html += `</tbody></table>`
	return html, nil
}

const cs322URL = "https://connect.ubc.ca/webapps/blackboard/content/listContent.jsp?course_id=_82806_1&content_id=_3510707_1"

func fetchCS322() (string, error) {
	bow := surf.NewBrowser()
	if err := bow.Open(cs322URL); err != nil {
		return "", err
	}

	// Follow redirects
	redirectLink := bow.Find("a").AttrOr("href", "")
	if err := bow.Open(redirectLink); err != nil {
		return "", err
	}
	redirectLink = strings.Split(strings.Split(bow.Find("noscript").Text(), "href=\"")[1], "\"")[0]
	if err := bow.Open(redirectLink); err != nil {
		return "", err
	}
	log.Printf("third %q\n%q\n%q", bow.Title(), bow.Url(), redirectLink)

	{
		// Log in to the site.
		fm, err := bow.Form("form")
		if err != nil {
			return "", err
		}

		if err := fm.Input("username", *cwlUser); err != nil {
			if err := fm.Input("j_username", *cwlUser); err != nil {
				return "", errors.Wrap(err, "no username or j_username elem")
			}
		}
		if err := fm.Input("password", *cwlPass); err != nil {
			if err := fm.Input("j_password", *cwlPass); err != nil {
				return "", errors.Wrap(err, "no password or j_password elem")
			}
		}
		if err := fm.Submit(); err != nil {
			return "", err
		}
	}
	{
		// SAML submit
		fm, err := bow.Form("form")
		if err != nil {
			return "", err
		}
		if err := fm.Submit(); err != nil {
			return "", err
		}
	}

	log.Printf("submit %q\n%q", bow.Title(), bow.Url())
	log.Println(bow.Dom().Html())
	if err := bow.Open(cs322URL); err != nil {
		return "", err
	}
	log.Printf("login %q\n%q", bow.Title(), bow.Url())

	dom := bow.Find("ul#content_listContainer")
	dom.Find("img").Remove()
	if err := makeAbsolute(dom, cs322URL); err != nil {
		return "", err
	}
	return goquery.OuterHtml(dom)
}

func makeAbsolute(sel *goquery.Selection, basePath string) error {
	base, err := url.Parse(basePath)
	if err != nil {
		return err
	}
	sel.Find("a").Each(func(_ int, s *goquery.Selection) {
		href, ok := s.Attr("href")
		if !ok {
			return
		}
		parsed, err := url.Parse(href)
		if err != nil {
			log.Printf("err parsing href %q: %s", href, err)
			return
		}
		s.SetAttr("href", base.ResolveReference(parsed).String())
	})
	tables := sel.Find("table")
	for _, attr := range []string{"border", "cellspacing", "cellpadding", "width", "rules"} {
		sel.RemoveAttr(attr)
		tables.RemoveAttr(attr)
	}
	return nil
}

var (
	port       = flag.Int("port", 80, "the port to listen on")
	piazzaUser = flag.String("piazzauser", "", "piazza username")
	piazzaPass = flag.String("piazzapass", "", "piazza password")
	cwlUser    = flag.String("cwluser", "", "cwl username")
	cwlPass    = flag.String("cwlpass", "", "cwl password")
)

var classFuncs = map[string]struct {
	fetch func() (string, error)
	url   string
}{
	"cs304": {fetchCS304, "https://www.ugrad.cs.ubc.ca/~cs304/2016W1/"},
	"cs311": {fetchCS311, "https://www.ugrad.cs.ubc.ca/~cs311/2016W1/"},
	"cs313": {fetchCS313, "https://piazza.com/class/isrvn2xyq3t69a"},
	"cs322": {fetchCS322, "https://connect.ubc.ca/webapps/blackboard/execute/content/blankPage?cmd=view&content_id=_3755785_1&course_id=_82806_1"},
	"cs340": {fetchCS340, "https://www.cs.ubc.ca/~schmidtm/Courses/340-F16/"},
}

var tmpls = template.Must(template.ParseFiles("index.html", "layout.html", "classes.html"))

func main() {
	flag.Parse()

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	sanitize := bluemonday.UGCPolicy()
	sanitize.AllowStyling()
	sanitize.AllowElements("font")

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		base := path.Base(r.URL.Path)
		if len(base) == 0 || base == "/" {
			classes, err := getClasses()
			if err != nil {
				http.Error(w, err.Error(), 500)
				return
			}
			if err := tmpls.Lookup("index.html").Execute(w, classes); err != nil {
				http.Error(w, err.Error(), 500)
			}
			return
		}

		dispCourses := strings.Split(strings.ToLower(base), ",")
		sort.Strings(dispCourses)

		courses := make([]course, len(dispCourses))
		var respsWG sync.WaitGroup
		var handin map[string][]Assignment
		var handinWG sync.WaitGroup
		handinWG.Add(1)
		go func() {
			var err error
			handin, err = fetchHandin()
			if err != nil {
				fmt.Fprintf(w, "<p>Handin Error: %s</p>", err)
			}
			handinWG.Done()
		}()
		for i, courseTitle := range dispCourses {
			courseTitle := courseTitle
			respsWG.Add(1)
			c := &courses[i]
			c.Course = courseTitle
			f, ok := classFuncs[courseTitle]
			go func() {
				if ok {
					c.CourseURL = f.url
					body, err := f.fetch()
					if err != nil {
						c.Errors = append(c.Errors, err)
					}
					c.CoursePage = template.HTML(sanitize.Sanitize(body))
				}
				handinWG.Wait()
				if assns, ok := handin[courseTitle]; ok {
					for _, resource := range assns {
						c.Handin = append(c.Handin, resource)
					}
				}
				respsWG.Done()
			}()
		}
		respsWG.Wait()
		if err := tmpls.Lookup("classes.html").Execute(w, courses); err != nil {
			http.Error(w, err.Error(), 500)
		}
	})

	log.Println(os.Args)

	addr := fmt.Sprintf(":%d", *port)
	log.Printf("Listening %s...", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}
