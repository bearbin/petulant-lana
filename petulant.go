package main

// Package main runs the petulant-lana server.

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

// Define the types

type configuration struct {
	Name           string `json:"name"`
	Url            string `json:"url"`
	CallbackSecret string `json:"callbacksecret"`
	BasePrice      int    `json:"baseprice"`
	MinimumPrice   int    `json:"minprice"`
	ApiKey         string `json:"coinbasekey"`
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

type ButtonType struct {
	Name        string `json:"name"`
	Price       string `json:"price_string"`
	Currency    string `json:"price_currency_iso"`
	Filename    string `json:"custom"`
	CallbackUrl string `json:"callback_url"`
	Description string `json:"description"`
	Type        string `json:"type"`
	Style       string `json:"style"`
}

type coinbaseRequest struct {
	Button ButtonType `json:"button"`
}

// Create the configuration
var config = configuration{}

func init() {
	// Seed the RNG. Only needs doing once at startup.
	rand.Seed(time.Now().UTC().UnixNano())

	// Open config file.
	configFile, err := os.Open("config.json")
	if err != nil {
		log.Fatal("failed to open config: ", err)
	}
	defer configFile.Close()

	// Decode config.
	decoder := json.NewDecoder(configFile)
	err = decoder.Decode(&config)
	if err != nil {
		log.Fatal("failed to decode config: ", err)
	}
}

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
			randomLetter := fmt.Sprint(rand.Intn(10))
			newName = newFileName(randomLetter + newName)
		}
	} else {
		// Add a random number onto the front of the filename.
		// This is not the best method, but it does for now.
		randomLetter := fmt.Sprint(rand.Intn(10))
		newName = newFileName(randomLetter + newName)
	}
	return newName
}

// Create a coinbase button.
func createButton(n string, p int) string {
	buttonCode := ButtonType {
		Name:        "One-Time Hosting Purchase",
		Price:       strconv.FormatFloat(float64(p)/float64(100000000), 'f', 8, 64),
		Currency:    "BTC",
		Filename:    n,
		CallbackUrl: fmt.Sprintf("%s/%s", config.Url, config.CallbackSecret),
		Description: fmt.Sprintf("Indefinite storage of the provided file. Your file will be available at: %s/%s when the transaction processes.", config.Url, n),
		Type:        "buy_now",
		Style:       "custom_large",
	}
	coinbaseRequest := coinbaseRequest{Button: buttonCode}

	data, err := json.Marshal(coinbaseRequest)
	if err != nil {
		log.Println("creating button: ", err)
	}
	request_body := bytes.NewBuffer(data)

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://coinbase.com/api/v1/buttons?api_key="+config.ApiKey, request_body)
	if err != nil {
		log.Println("creating coinbase request: ", err)
	}

	req.Header.Add("content-type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		log.Println("completing coinbase request: ", err)
	}

	response_body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Println("reading coinbase requst: ", err)
	}
	defer resp.Body.Close()
	
	res := transactionResult{}
	err = json.Unmarshal(response_body, &res)
	if err != nil {
		log.Println("decoding coinbase response: ", err)
	}
	return res.Button.Code

}

// hello world, the web server 
func upload(w http.ResponseWriter, req *http.Request) {

	// Get the form file.
	file, header, err := req.FormFile("file")
	if err != nil {
		log.Println("form file: ", err)
		return
	}

	// Get the name for the file.
	fileName := newFileName(header.Filename)
	log.Print("uploaded new file: ", fileName)

	dataFile, err := os.Create("tmp/"+fileName)
	if err != nil {
		log.Println("opening file for writing: ", err)
	}
	defer dataFile.Close()

	io.Copy(dataFile, file)


	// Get file size.
	fileInfo, _ := os.Stat("tmp/" + fileName)
	fileSize := math.Floor(float64(fileInfo.Size()) / 1024)
	price := int(math.Floor(float64(config.BasePrice) * (fileSize / 1024)))
	if price < config.MinimumPrice {
		price = config.MinimumPrice
	}

	// Put info on the page.
	buttonCode := createButton(fileName, price)
	fileCode := fmt.Sprintf("%s/f/%s", config.Url, fileName)
	pageSource := fmt.Sprintf(
		`
		<html>
		<head>
			<title>Upload Finished</title>
		</head>
		<body>
			<p>Your upload has finished, now all you need to do is pay!</p>
			<a class="coinbase-button" data-code="%s" data-button-style="custom_large" data-button-text="Checkout with Bitcoin" href="#">Checkout With Bitcoin</a>
			<script src="https://coinbase.com/assets/button.js" type="text/javascript"></script>
			<p>Your file will be available at <a href="%s">%s</a>. Don't forget this as it's very hard to find out which file you uploaded.</p>
		</body>
		</html>
		`, buttonCode, fileCode, fileCode)
	io.WriteString(w, pageSource)
}

func coinbaseCallback(w http.ResponseWriter, req *http.Request) {
	body, _ := ioutil.ReadAll(req.Body)
	res := callbackResult{}
	fmt.Println(body)
	json.Unmarshal([]byte(body), &res)
	fmt.Println(res.Order.Filename)
	os.Rename("tmp/"+res.Order.Filename, "f/"+res.Order.Filename)
}

func MainPage(w http.ResponseWriter, req *http.Request) {
	t, _ := template.ParseFiles("index.html")
	err := t.Execute(w, config)
	if err != nil {
		log.Println("Error loading main page: ", err)
	}
}

func main() {
	bindAddr := flag.String("port", "8080", "Server port.")
	iface := flag.String("iface", "0.0.0.0", "Interface to bind to.")
	flag.Parse()
	// Main page
	http.HandleFunc("/", MainPage)
	// Upload page
	http.HandleFunc("/upload", upload)
	// Coinbase callback
	http.HandleFunc("/"+config.CallbackSecret, coinbaseCallback)
	// Static files
	http.Handle("/f/", http.FileServer(http.Dir("")))

	log.Println("Binding to port", *bindAddr)
	log.Fatal(http.ListenAndServe(*iface + ":" + *bindAddr, nil))
}
