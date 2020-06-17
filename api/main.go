package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/pat"
	"github.com/satori/uuid"
	"github.com/trotsek/groups"
)

func main() {
	a := &app{
		groupByID: map[string]groups.Group{},
	}

	//v1 API:
	p := "/groups/v1"
	r := pat.New()
	r.Get("/metrics", a.metrics)

	r.Delete(p+"/{id}", a.delGroup)
	r.Get(p+"/{id}", a.getGroup)
	r.Post(p, a.newGroup)
	r.Get(p, a.getGroups)
	r.Put(p, a.updGroup)

	r.Options("/", a.options)
	a.r = setContentType(r)

	http.ListenAndServe("localhost:12345", a.r)
}

func setContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")
		h.ServeHTTP(res, req)

		fmt.Printf("Response Headers:\n")
		for h, v := range res.Header() {
			fmt.Printf("  %s: %+v\n", h, v)
		}

	})
} //setContentType

type app struct {
	sync.Mutex
	r         http.Handler
	groupByID map[string]groups.Group
}

func (app *app) getGroups(res http.ResponseWriter, req *http.Request) {
	app.Lock()
	defer app.Unlock()
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, err.Error(), http.StatusMethodNotAllowed)
		return
	}

	nameFilter := strings.ToUpper(req.URL.Query().Get("name"))
	res.Header().Set("Content-Type", "application/json")
	l := []groups.Group{}
	for _, g := range app.groupByID {
		if nameFilter == "" || strings.Index(strings.ToUpper(g.Name), nameFilter) >= 0 {
			l = append(l, g)
		}
	}
	jsonList, _ := json.Marshal(l)
	res.Write(jsonList)
}

func (app *app) newGroup(res http.ResponseWriter, req *http.Request) {
	app.Lock()
	defer app.Unlock()
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	ct := req.Header.Get("Content-Type")

	var g groups.Group
	parsedContent := false
	if strings.Index(ct, "application/json") >= 0 {
		if err := json.NewDecoder(req.Body).Decode(&g); err != nil {
			http.Error(res, fmt.Sprintf("cannot unmarshal JSON content: %v", err), http.StatusBadRequest)
			return
		}
		parsedContent = true
	}
	if !parsedContent {
		http.Error(res, "expecting Content-Type application/json", http.StatusBadRequest)
		return
	}

	if err := g.Validate(); err != nil {
		http.Error(res, fmt.Sprintf("invalid group: %v", err), http.StatusBadRequest)
		return
	}
	g.ID = uuid.NewV1().String()
	app.groupByID[g.ID] = g
	jsonGroup, _ := json.Marshal(g)
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonGroup)
	return
}

func (app *app) getGroup(res http.ResponseWriter, req *http.Request) {
	app.Lock()
	defer app.Unlock()
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := req.URL.Query().Get(":id")
	if g, ok := app.groupByID[id]; ok {
		jsonGroup, _ := json.Marshal(g)
		res.Write(jsonGroup)
		return
	}
	http.Error(res, "unknown id", http.StatusNotFound)
}

func (app *app) updGroup(res http.ResponseWriter, req *http.Request) {
	app.Lock()
	defer app.Unlock()
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	ct := req.Header.Get("Content-Type")
	var g groups.Group
	parsedContent := false
	if strings.Index(ct, "application/json") >= 0 {
		if err := json.NewDecoder(req.Body).Decode(&g); err != nil {
			http.Error(res, fmt.Sprintf("cannot unmarshal JSON content: %v", err), http.StatusBadRequest)
			return
		}
		parsedContent = true
	}
	if !parsedContent {
		http.Error(res, "expecting Content-Type application/json", http.StatusBadRequest)
		return
	}

	if err := g.Validate(); err != nil {
		http.Error(res, fmt.Sprintf("invalid group: %v", err), http.StatusBadRequest)
		return
	}

	if _, ok := app.groupByID[g.ID]; ok {
		app.groupByID[g.ID] = g //replace with new data
		jsonGroup, _ := json.Marshal(g)
		res.Write(jsonGroup)
		return
	}
	http.Error(res, "unknown id", http.StatusNotFound)
}

func (app *app) delGroup(res http.ResponseWriter, req *http.Request) {
	app.Lock()
	defer app.Unlock()
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := req.URL.Query().Get(":id")
	if _, ok := app.groupByID[id]; ok {
		delete(app.groupByID, id)
		return
	}
	http.Error(res, "unknown id", http.StatusNotFound)
}

func (app *app) unknown(res http.ResponseWriter, req *http.Request) {
	fmt.Printf("ERROR: HTTP %s %s\n", req.Method, req.URL.Path)
}

func (app *app) metrics(res http.ResponseWriter, req *http.Request) {
	//ignore
}

//preflight OPTIONS handling before PUT/DEL
func (app *app) options(res http.ResponseWriter, req *http.Request) {
	log(req)
	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	// HTTP OPTIONS /groups/v1
	// Origin: [http://localhost:4200]
	// Sec-Fetch-Dest: [empty]
	// User-Agent: [Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/83.0.4103.97 Safari/537.36]
	// Accept-Encoding: [gzip, deflate, br]
	// Connection: [keep-alive]
	// Accept: [*/*]
	// Access-Control-Request-Method: [POST]
	// Accept-Language: [en-GB,en-US;q=0.9,en;q=0.8]
	// Access-Control-Request-Headers: [content-type]
	// Sec-Fetch-Mode: [cors]
	// Sec-Fetch-Site: [same-site]
	// Referer: [http://localhost:4200/groups]
}

//check for simple GET/POST requests
func (app *app) checkRequest(res http.ResponseWriter, req *http.Request) error {
	if origin := req.Header.Get("Origin"); origin == "http://localhost:4200" || origin == "" {
		//res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Origin", origin)
		fmt.Printf("Allowing origin!\n")
	} else {
		fmt.Printf("***** NOT Allowing origin! *****\n")
		return fmt.Errorf("origin:\"%s\" not allowed", origin)
	}

	method := req.Header.Get("Access-Control-Request-Method")
	switch method {
	case "": //no method requested
	case "OPTIONS", "POST", "PUT", "DELETE", "GET":
		res.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, GET, PUT, DELETE")
	default:
		fmt.Printf("***** NOT Allowing method %s *****\n", method)
		return fmt.Errorf("method:%s not allowed", method)
	}

	reqHeaders := strings.Split(req.Header.Get("Access-Control-Request-Headers"), ",")
	resHeaders := ""
	for _, h := range reqHeaders {
		if strings.ToLower(h) == "content-type" {
			resHeaders += ", Content-Type"
		}
	}
	if len(resHeaders) > 2 {
		//res.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding")
		res.Header().Set("Access-Control-Allow-Headers", resHeaders[2:])
	}
	return nil
}

func log(req *http.Request) {
	fmt.Printf("HTTP %s %s\n", req.Method, req.URL.Path)
	for h, v := range req.Header {
		fmt.Printf("  %s: %+v\n", h, v)
	}
}
