package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/headzoo/surf"
)

const cs304URL = "http://www.ugrad.cs.ubc.ca/~cs304/2016W1/schedule.html"

func fetchCS304() (string, error) {
	doc, err := goquery.NewDocument(cs304URL)
	if err != nil {
		return "", err
	}
	return goquery.OuterHtml(doc.Find("table"))
}

const cs311URL = "https://www.ugrad.cs.ubc.ca/~cs311/2016W1/_homework.php"

func fetchCS311() (string, error) {
	doc, err := goquery.NewDocument(cs311URL)
	if err != nil {
		return "", err
	}
	return goquery.OuterHtml(doc.Find("table[rules]"))
}

const piazzaLoginURL = `https://piazza.com/account/login`
const cs313URL = "https://piazza.com/ubc.ca/winterterm12016/cpsc313/resources"

func fetchCS313() (string, error) {
	bow := surf.NewBrowser()
	if err := bow.Open(piazzaLoginURL); err != nil {
		return "", err
	}

	// Log in to the site.
	fm, _ := bow.Form("form#login-form")
	fm.Input("email", *piazzaUser)
	fm.Input("password", *piazzaPass)
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
	html += `</tbody></body>`
	return html, nil
}

var port = flag.Int("port", 80, "the port to listen on")
var piazzaUser = flag.String("piazzauser", "", "piazza username")
var piazzaPass = flag.String("piazzapass", "", "piazza password")

func main() {
	flag.Parse()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(solarizedDark))
		w.Write([]byte("<h1>Class Lists</h1>"))

		funcs := []struct {
			title string
			fetch func() (string, error)
		}{
			{"cs304", fetchCS304},
			{"cs311", fetchCS311},
			{"cs313", fetchCS313},
		}

		for _, f := range funcs {
			fmt.Fprintf(w, "<h2>%s</h2>", f.title)
			body, err := f.fetch()
			if err != nil {
				fmt.Fprintf(w, "<p>Error: %s</p>", err)
			}
			w.Write([]byte(body))
		}
	})

	log.Println("Running...")
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", *port), nil))
}

const solarizedDark = `<style>
table {width: 100% !important;}
/*
 * Drop the below regex, after a comma, just before the opening curly bracket
 * above, to exclude websites from solarization:
  * ,regexp("https?://(www\.)?(?!(userstyles\.org|docs\.google|github)\..*).*")
	 */


	 /* Firefox Scrollbars */
	 scrollbar {opacity: .75 !important;}

	 /*Vars
	 base03    #002b36 rgba(0,43,54,1);
	 base02    #073642 rgba(7,54,66,1);
	 base01    #586e75 rgba(88,110,117,1);
	 base00    #657b83 rgba(101,123,131,1);
	 base0     #839496 rgba(131,148,150,1);
	 base1     #93a1a1 rgba(147,161,161,1);
	 base2     #eee8d5 rgba(238,232,213,1);
	 base3     #fdf6e3 rgba(253,246,227,1);
	 yellow    #b58900 rgba(181,137,0,1);
	 orange    #cb4b16 rgba(203,75,22,1);
	 red       #dc322f rgba(220,50,47,1);
	 magenta   #d33682 rgba(211,54,130,1);
	 violet    #6c71c4 rgba(108,113,196,1);
	 blue      #268bd2 rgba(38,139,210,1);
	 cyan      #2aa198 rgba(42,161,152,1);
	 green     #859900 rgba(133,153,0,1);
	 */

	 /* Base */
	 *, ::before, ::after {
		 color: #93a1a1 !important;
		 border-color: #073642 !important;
		 outline-color: #073642 !important;
		 text-shadow: none !important;
		 box-shadow: none !important;
		 /*-moz-box-shadow: none !important;*/
		 background-color: transparent !important;
	 }

	 html * {
		 color: inherit !important;
	 }

	 p::first-letter,
	 h1::first-letter,
	 h2::first-letter,
	 p::first-line {

		 color: inherit !important;
		 background: none !important;
	 }

	 /* :: Give solid BG :: */

	 /* element */
	 b,i,u,strong{color:#859900}


	 html,
	 body,
	 li ul,
	 ul li,
	 table,
	 header,
	 article,
	 section,
	 nav,
	 menu,
	 aside,

	 /* common */

	 [class*="nav"],
	 [class*="open"],
	 [id*="ropdown"], /*dropdown*/
	 [class*="ropdown"],
	 div[class*="menu"],
	 [class*="tooltip"],
	 div[class*="popup"],
	 div[id*="popup"],

	 /* Notes, details, etc.  Maybe useful */
	 div[id*="detail"],div[class*="detail"],
	 div[class*="note"], span[class*="note"],
	 div[class*="description"],

	 /* Also common */
	 div[class*="content"], div[class*="container"],

	 /* Popup divs that use visibility: hidden and display: none */
	 div[style*="display: block"],
	 div[style*="visibility: visible"] {
		 background-color: #002b36 !important
	 }



	 /*: No BG :*/
	 *:not(:empty):not(span):not([class="html5-volume-slider html5-draggable"]):not([class="html5-player-chrome html5-stop-propagation"]), *::before, *::after,
	 td:empty, p:empty, div:empty:not([role]):not([style*="flashblock"]):not([class^="html5"]):not([class*="noscriptPlaceholder"]) {
		 background-image: none !important;
	 }

	 /*: Filter non-icons :*/
	 span:not(:empty):not([class*="icon"]):not([id*="icon"]):not([class*="star"]):not([id*="star"]):not([id*="rating"]):not([class*="rating"]):not([class*="prite"]) {
		 background-image: none !important;
		 text-indent: 0 !important;
	 }

	 /*: Image opacity :*/
	 img, svg             {opacity: .75 !important;}
	 img:hover, svg:hover {opacity: 1 !important;}

	 /* Highlight */
	 ::-moz-selection {
		 background-color: #eee8d5 !important;
		 color: #586e75 !important;
	 }

	 /* ::: anchor/links ::: */

	 a {
		 color: #2aa198 !important;
		 background-color: #002b36 !important;
		 opacity: 1 !important;
		 text-indent: 0 !important;
	 }

	 a:link         {color: #268bd2 !important;} /* hyperlink */
	 a:visited      {color: #6c71c4 !important;}
	 a:hover        {color: #b58900 !important; background-color: #073642 !important;}
	 a:active       {color: #cb4b16 !important;}

	 /* "Top level" div */

	 body > div {background-color: inherit !important;}

	 /* :::::: Text Presentation :::::: */

	 summary, details                   {background-color: inherit !important}
	 kbd, time, label, .date            {color: #859900 !important}
	 acronym, abbr                      {border-bottom: 1px dotted !important; cursor: help !important;}
	 mark000000       {background-color: #dc322f !important}


	 /* :::::: Headings :::::: */

	 h1,h2,h3,h4,h5,h6  {

		 background-image: none !important;
		 border-radius: 5px !important;
		 /*-moz-border-radius: 5px !important;*/
		 -webkit-border-radius: 5px !important;
		 text-indent: 0 !important;
	 }

	 h1,h2,h3,h4,h5,h6 {background-color: #073642 !important}


	 h1,h2{color:#859900!important}
	 h3,h4{color:#b58900!important}
	 h5,h6{color:#cb4b16!important}

	 /* :::::: Tables, cells :::::: */

	 table table {background: #073642 !important;}
	 th, caption {background: #002b36 !important;}

	 /* ::: Inputs, textareas ::: */

	 input, textarea, button,
	 select,option,optgroup{

		 color: #586e75 !important;
		 background: none #073642 !important;
		 -moz-appearance: none !important;
		 -webkit-appearance: none !important;
	 }

	 input,
	 textarea,
	 button {
			 border-color: #586e75 !important;
		 border-width: 1px !important;
	 }

	 /* :::::: Button styling :::::: */

	 input[type="button"],
	 input[type="submit"],
	 input[type="reset"],
	 button {
		 background: #073642 !important;
	 }

	 input[type="button"]:hover,
	 input[type="submit"]:hover,
	 input[type="reset"]:hover,
	 button:hover {
		 color: #586e75 !important;
		 background: #eee8d5 !important;
	 }

	 input[type="image"] {opacity: .85 !important}
	 input[type="image"]:hover {opacity: .95 !important}

	 /* Lightbox fix */
	 html [id*="lightbox"] * {background-color: transparent !important;}
	 html [id*="lightbox"] img {opacity: 1 !important;}

	 /* Youtube Annotation */
	 #movie_player-html5 .annotation {background: #073642 !important}

	 /* Mozilla addons shrink/expand sections */
	 .expando a {background: none transparent  !important;}

	 .reading  {color: #088A29 !important}
	 .homework {color: #FF6600 !important}
	 .project  {color: #3333FF !important}
	 .special  {color: #CC0033 !important}
	 .tutorial {color: #990099 !important}
	 .peerwise {color: #E67E22 !important}
</style>`
