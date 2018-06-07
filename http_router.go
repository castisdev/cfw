package main

import (
	"net/http"

	"github.com/gorilla/mux"
)

// Route is struct for http route
type Route struct {
	Name        string
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// Routes is slice of Route struct
type Routes []Route

// NewRouter is constructor of Route struct
func NewRouter() *mux.Router {

	router := mux.NewRouter().StrictSlash(true)

	for _, route := range routes {

		var handler http.Handler
		handler = route.HandlerFunc

		// logger middle ware
		//handler = Logger(handler, route.Name)

		router.
			Methods(route.Method).
			Path(route.Pattern).
			Name(route.Name).
			Handler(handler)
	}

	return router
}

var routes = Routes{
	Route{
		Name:        "ls",
		Method:      "GET",
		Pattern:     "/files",
		HandlerFunc: GetFileList,
	},
	Route{
		Name:        "df",
		Method:      "GET",
		Pattern:     "/df",
		HandlerFunc: GetDiskUsage,
	},
	Route{
		Name:        "rm",
		Method:      "DELETE",
		Pattern:     "/files/{fileName}",
		HandlerFunc: DeleteFile,
	},
}
