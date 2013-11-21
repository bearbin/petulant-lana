package main
// Package main runs the petulant-lana server.

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
	"rand"
	"strconv"
	"strings"
)

// Define the types

type configuration struct {
	Name           string `json: "name"`
	Url            string `json: "url"`
	CallbackSecret string `json: "callbacksecret"`
	BasePrice      int    `json: "baseprice"`
	MinimumPrice   int    `json: "minprice"`
	ApiKey         string `json: "coinbasekey"`
}

type transactionResult struct {
	Success bool `json:"success"`
	Button  struct {
		Code string `json:"code"`
	} `json:"button"`
}

type callbackResult struct {
	Order struct {
		Filename string `json:"custom"`
	} `json:"order"`
}

// Create the configuration
var config = configuration{}

// Do stuff

// Get an appropriate name for the file.
func newFileName(fname string) string {
	// First, remove slashes and spaces, replace with dashes.
	newName := strings.Replace(strings.Replace(fname, "/", "-", -1), " ", "-", -1)

	// Does the current file already exist, in storage.
	if _, err := os.Stat("f/" + newName); os.IsNotExist(err) {
		// Does the current file already exist, in temporary storage.
		if _, err := os.Stat("tmp/" + newName); os.IsNotExist(err) {
			// Don't do anything.
		} else {
			// Add a random number onto the front of the filename.
			// This is not the best method, but it does for now.
			randomLetter := fmt.Sprint(rand.Int())
			newName = newFileName(randomLetter + newName)
		}
	} else {
		// Add a random number onto the front of the filename.
		// This is not the best method, but it does for now.
		randomLetter := fmt.Sprint(rand.Int())
		newName = newFileName(randomLetter + newName)
	}
	return newName
}

// Create a coinbase button.
func createButton(n string, p int) string {
	coinbaserequest := "{ \"button\": {" +
		"\"name\": \"One-Time Hosting Purchase\"," +
		"\"type\": \"buy_now\"," +
		"\"price_string\": \"" + strconv.FormatFloat(float64(p)/float64(100000000), 'f', 8, 64) + "\"," +
		"\"price_currency_iso\": \"BTC\"," +
		"\"custom\": \"" + n + "\"," +
		"\"callback_url\": \"whatever\"," +
		"\"description\": \"Indefinite storage of the provided file. Your file will be available at: http://btcdl.bearbin.net/f/" + n + " when the transaction processes.\"," +
		"\"type\": \"buy_now\"," +
		"\"style\": \"custom_large\"" +
		"} }"
	fmt.Println(coinbaserequest)
	request_body := bytes.NewBuffer([]byte(coinbaserequest))

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://coinbase.com/api/v1/buttons?api_key="+config.ApiKey, request_body)
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
	}

	response_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
	}
	defer resp.Body.Close()
	res := transactionResult{}
	fmt.Println(string(response_body))
	err = json.Unmarshal(response_body, &res)
	return res.Button.Code

}

// hello world, the web server 
func upload(w http.ResponseWriter, req *http.Request) {

	// Get the form file.
	file, header, err := req.FormFile("file")
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get the name for the file.
	fileName := newFileName(header.Filename)
	log.Print("Uploaded new file: ", fileName)

	data, err := ioutil.ReadAll(file)
	if err != nil {
		log.Print(err)
		return
	}
	err = ioutil.WriteFile("tmp/"+fileName, data, 0777)
	if err != nil {
		log.Print(err)
		return
	}

	// Get file size.
	supfil, _ := os.Stat("tmp/" + fileName)
	fileSize := math.Floor(float64(supfil.Size()) / 1024)
	price := int(math.Floor(float64(config.BasePrice) * (fileSize/1024)))
	if price < config.MinimumPrice {
		price = config.MinimumPrice
	}
	// Redirect the user.
	http.Redirect(w, req, "https://coinbase.com/checkouts/"+createButton(fileName, price), 302)

}

func coinbaseCallback(w http.ResponseWriter, req *http.Request) {
	fmt.Println("LELELELE")
	body, _ := ioutil.ReadAll(req.Body)
	res := callbackResult{}
	fmt.Println(body)
	json.Unmarshal([]byte(body), &res)
	fmt.Println(res.Order.Filename)
	os.Rename("tmp/"+res.Order.Filename, "f/"+res.Order.Filename)
}

func MainPage(w http.ResponseWriter, req *http.Request) {
	t, _ := template.ParseFiles("index.html")
	t.Execute(w, "")
}

func main() {
	// Inititalize the config.
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal("Failed to open config: ", err)
	}
	decoder := json.NewDecoder(configFile)

	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("Failed to open config: ", err)
	}

	// Main page
	http.HandleFunc("/", MainPage)
	// Upload page
	http.HandleFunc("/upload", upload)
	// Coinbase callback
	http.HandleFunc("/wheatver", coinbaseCallback)
	// Static files
	http.Handle("/f/", http.FileServer(http.Dir("")))

	// Try and serve port 80.
	err = http.ListenAndServe(":80", nil)
	if err != nil {
		// Failed for some reason, try port 8080
		log.Print("Failed to bind to port 80, trying 8080.")
		err := http.ListenAndServe(":8080", nil)
		if err != nil {
			// Failed.
			log.Fatal("ListenAndServe: ", err)
		}
	}
}
