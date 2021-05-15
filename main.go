package main

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"github.com/umahmood/haversine"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type Location struct {
	Lat float64 `json:"lat"`
	Lon float64 `json:"lon"`
}

type City struct {
	Airports    []string      `json:"airports"`
	Con         int           `json:"con"`
	ContId      string        `json:"contId"`
	CountryId   string        `json:"countryId"`
	CountryName string        `json:"countryName"`
	Dest        string        `json:"dest"`
	Iata        string        `json:"iata"`
	Id          string        `json:"id"`
	Images      []interface{} `json:"images"`
	Location    Location 	`json:"location"`
	Name       string      `json:"name"`
	Popularity float64     `json:"popularity"`
	Rank       int         `json:"rank"`
	RegId      string      `json:"regId"`
	SubId      interface{} `json:"subId"`
	TerId      interface{} `json:"terId"`
}


func homePage(w http.ResponseWriter, r *http.Request){
	fmt.Fprintf(w, "Welcome to the HomePage!")
	fmt.Println("Endpoint Hit: homePage")
}

func handleRequests() {
	myRouter := mux.NewRouter().StrictSlash(true)
	myRouter.HandleFunc("/", homePage)
	myRouter.HandleFunc("/get-paths", getTravelPoints).Methods("POST")
	log.Fatal(http.ListenAndServe(":10000", myRouter))
}

type Request struct {
	Origin string `json:"origin"`
}

func contains(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

func getTravelPoints(w http.ResponseWriter, r *http.Request) {
	// get the body of our POST request
	// unmarshal this into a new Request struct
	reqBody, _ := ioutil.ReadAll(r.Body)
	var request Request
	json.Unmarshal(reqBody, &request)

	//get from url
	//url := "https://s3.us-west-2.amazonaws.com/secure.notion-static.com/4be05480-e7fc-4b41-b642-fb26dcaa4c39/cities.json?X-Amz-Algorithm=AWS4-HMAC-SHA256&X-Amz-Credential=AKIAT73L2G45O3KS52Y5%2F20210515%2Fus-west-2%2Fs3%2Faws4_request&X-Amz-Date=20210515T064252Z&X-Amz-Expires=86400&X-Amz-Signature=4d16b51a35124ab41cdba086d4ad7ea60628f4bffa9162df8bcc258a23703757&X-Amz-SignedHeaders=host&response-content-disposition=filename%20%3D%22cities.json%22"
	//req, err := http.NewRequest("GET", url, nil)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//
	//client := &http.Client{}
	//resp, err := client.Do(req)
	//if err != nil {
	//	log.Fatalln(err)
	//}
	//body, err := ioutil.ReadAll(resp.Body)
	//if err != nil {
	//	panic(err.Error())
	//}

	//get from file
	body, _ := ioutil.ReadFile("cities.json")
	var result map[string]interface{}
	json.Unmarshal(body, &result)

	//Get starting city
	b, _ := json.Marshal(result[request.Origin])
	startCity := City{}
	json.Unmarshal(b, &startCity)

	//get 6 unique continents from response
	s := make([]string, 0)
	for k := range result {
		b, err := json.Marshal(result[k])
		if err != nil {
			panic(err)
		}
		city := City{}
		json.Unmarshal(b, &city)
		if !contains(s, city.ContId) && city.ContId != startCity.ContId {
			s = append(s, city.ContId)
		}
		//break
	}
	//fmt.Println(s)

	//Get city from each continent asynchronously with go routine
	c := make(chan cityPoint)
	for _, cont := range s {
		go getCityFromCont(cont, result, c)
	}
	points := make([]cityPoint, len(s))
	var response = make([]string, 0)
	response = append(response, request.Origin + ", (" + startCity.RegId + ")")

	var lat1 = startCity.Location.Lat
	var lon1 = startCity.Location.Lon
	var distance =0.0
	//Loop through each continent result
	for i, _ := range points {
		points[i] = <-c
		response = append(response, points[i].city + ", (" + points[i].region + ")")
		distance += getDistanceFromLatLonInKm( lat1,lon1, points[i].lat, points[i].long)
		//fmt.Println(points[i].city, " ", points[i].region)
		lat1 = points[i].lat
		lon1 = points[i].long
	}
	distance+=getDistanceFromLatLonInKm(lat1, lon1, startCity.Location.Lat, startCity.Location.Lon)
	response = append(response, request.Origin + ", (" + startCity.RegId + ")")

	//json.NewEncoder(w).Encode(strings.Join(response[:], ",") + "" + fmt.Sprintf("%f", distance))
	json.NewEncoder(w).Encode(Response{
		Path:     strings.Join(response[:], " -- "),
		Distance: distance,
	})
	//close channel to return
	close(c)
}


func main() {
	handleRequests()
}

func getCityFromCont(contId string, result map[string]interface{}, c chan cityPoint){
	for k := range result {
		b, _ := json.Marshal(result[k])
		city := City{}
		json.Unmarshal(b, &city)
		if contId==city.ContId {
			//fmt.Printf("%v", city)
			c <- cityPoint{k, city.Name +" "+ city.CountryName, city.Location.Lat, city.Location.Lon}
			break
		}
	}
}

type cityPoint struct {
	city    string
	region string
	lat float64
	long float64
}

type Response struct {
	Path string `json:"path"`
	Distance float64 `json:"distanceInKM"`
}

func getDistanceFromLatLonInKm(lat1 float64,lon1 float64,lat2 float64,lon2 float64) float64 {
	l1 := haversine.Coord{Lat: lat1, Lon: lon1}
	l2  := haversine.Coord{Lat: lat2, Lon: lon2}
	_, km := haversine.Distance(l1,l2)
	return km
}