package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/pat"
	"github.com/jansemmelink/groups"
	mongogroups "github.com/jansemmelink/groups/mongo"
)

func main() {
	mongoURIFlag := flag.String("mongo", "mongodb://localhost:27017", "Mongo address")
	dbNameFlag := flag.String("db", "trotsek", "Mongo database name")
	addrFlag := flag.String("addr", "localhost:12345", "Server address")
	flag.Parse()

	g, err := mongogroups.Groups(*mongoURIFlag, *dbNameFlag)
	if err != nil {
		panic(err)
	}
	a := &app{
		g: g,
		//groupByID: map[string]groups.Group{},
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

	http.ListenAndServe(*addrFlag, a.r)
}

func setContentType(h http.Handler) http.Handler {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		res.Header().Add("Content-Type", "application/json")
		h.ServeHTTP(res, req)

		// fmt.Printf("Response Headers:\n")
		// for h, v := range res.Header() {
		// 	fmt.Printf("  %s: %+v\n", h, v)
		// }

	})
} //setContentType

type app struct {
	r http.Handler
	g groups.IGroups
}

func (app *app) getGroups(res http.ResponseWriter, req *http.Request) {
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, err.Error(), http.StatusMethodNotAllowed)
		return
	}

	filter := map[string]interface{}{}
	name := strings.ToUpper(req.URL.Query().Get("name"))
	if name != "" {
		filter["name"] = name
	}
	size := req.URL.Query().Get("size")
	sizeLimit := 10
	if size != "" {
		var err error
		sizeLimit, err = strconv.Atoi(size)
		if err != nil || sizeLimit <= 0 {
			sizeLimit = 10
		}
	}
	l := app.g.List(filter, sizeLimit, []string{"name"})
	jsonList, _ := json.Marshal(l)
	res.Write(jsonList)
}

func (app *app) newGroup(res http.ResponseWriter, req *http.Request) {
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

	g, err := app.g.New(g)
	if err != nil {
		http.Error(res, fmt.Sprintf("failed to create: %v", err), http.StatusBadRequest)
		return
	}

	jsonGroup, _ := json.Marshal(g)
	res.Header().Set("Content-Type", "application/json")
	res.Write(jsonGroup)
	return
}

func (app *app) getGroup(res http.ResponseWriter, req *http.Request) {
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := req.URL.Query().Get(":id")
	if g, err := app.g.Get(id); err == nil && g != nil {
		jsonGroup, _ := json.Marshal(*g)
		res.Write(jsonGroup)
		return
	}
	http.Error(res, "unknown id", http.StatusNotFound)
}

func (app *app) updGroup(res http.ResponseWriter, req *http.Request) {
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

	if err := app.g.Upd(g); err != nil {
		http.Error(res, "update failed: "+err.Error(), http.StatusNotFound)
		return
	}
	jsonGroup, _ := json.Marshal(g)
	res.Write(jsonGroup)
}

func (app *app) delGroup(res http.ResponseWriter, req *http.Request) {
	log(req)

	if err := app.checkRequest(res, req); err != nil {
		http.Error(res, "not allowed", http.StatusMethodNotAllowed)
		return
	}

	id := req.URL.Query().Get(":id")
	if err := app.g.Del(id); err != nil {
		http.Error(res, "unknown id", http.StatusNotFound)
		return
	}
	return
}

func (app *app) unknown(res http.ResponseWriter, req *http.Request) {
	fmt.Printf("Unknown: HTTP %s %s\n", req.Method, req.URL.Path)
	http.Error(res, "unknown request", http.StatusNotFound)
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
}

//check for simple GET/POST requests
func (app *app) checkRequest(res http.ResponseWriter, req *http.Request) error {
	if origin := req.Header.Get("Origin"); origin == "http://localhost:4200" || origin == "" {
		//res.Header().Set("Access-Control-Allow-Origin", "*")
		res.Header().Set("Access-Control-Allow-Origin", origin)
	} else {
		return fmt.Errorf("origin:\"%s\" not allowed", origin)
	}

	method := req.Header.Get("Access-Control-Request-Method")
	switch method {
	case "": //no method requested
	case "OPTIONS", "POST", "PUT", "DELETE", "GET":
		res.Header().Set("Access-Control-Allow-Methods", "OPTIONS, POST, GET, PUT, DELETE")
	default:
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
