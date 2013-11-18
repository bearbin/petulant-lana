package main

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
	"strconv"
	"strings"
)

type TransactionResult struct {
	Success bool `json:"success"`
	Button  struct {
		Code string `json:"code"`
	} `json:"button"`
}

type CallbackResult struct {
	Order struct {
		Filename string `json:"custom"`
	} `json:"order"`
}

// Get an appropriate name for the file.
func getname(fname string) string {
	result := strings.Replace(strings.Replace(fname, "/", "-", -1), " ", "-", -1)
	if _, err := os.Stat("f/" + result); os.IsNotExist(err) {
		if _, err := os.Stat("tmp/" + result); os.IsNotExist(err) {
			// Don't do anything.
		} else {
			result = getname("p" + result)
		}
	} else {
		result = getname("p" + result)
	}
	return result
}

// Create a coinbase button.
func createbutton(n string, p float64) string {
	coinbaserequest := "{ \"button\": {" +
		"\"name\": \"One-Time Hosting Purchase\"," +
		"\"type\": \"buy_now\"," +
		"\"price_string\": \"" + strconv.FormatFloat(p, 'f', 8, 64) + "\"," +
		"\"price_currency_iso\": \"BTC\"," +
		"\"custom\": \"" + n + "\"," +
		"\"callback_url\": \"whatever\"," +
		"\"description\": \"Indefinite storage of the provided file. Your file will be available at: http://btcdl.bearbin.net/f/" + n + " when the transaction processes.\"," +
		"\"type\": \"buy_now\"," +
		"\"style\": \"custom_large\"" +
		"} }"
	apikey := "InsertKeyHere"
	fmt.Println(coinbaserequest)
	request_body := bytes.NewBuffer([]byte(coinbaserequest))

	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://coinbase.com/api/v1/buttons?api_key="+apikey, request_body)
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
	res := TransactionResult{}
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
	filename := header.Filename
	filename = getname(filename)
	fmt.Println(filename)

	data, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = ioutil.WriteFile("tmp/"+filename, data, 0777)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Get file size.
	supfil, _ := os.Stat("tmp/" + filename)
	filesize := float64(supfil.Size())
	price := math.Floor(math.Floor(filesize/1024)*4.8828125) / 100000000
	if price < 0.000025 {
		price = 0.000025
	}
	fmt.Println(strconv.FormatFloat(price, 'f', 8, 64))

	// Redirect the user.
	http.Redirect(w, req, "https://coinbase.com/checkouts/"+createbutton(filename, price), 302)

}

func coinbaseCallback(w http.ResponseWriter, req *http.Request) {
	fmt.Println("LELELELE")
	body, _ := ioutil.ReadAll(req.Body)
	res := CallbackResult{}
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
	// Main page
	http.HandleFunc("/", MainPage)
	// Upload page
	http.HandleFunc("/upload", upload)
	// Coinbase callback
	http.HandleFunc("/wheatver", coinbaseCallback)
	// Static files
	http.Handle("/f/", http.FileServer(http.Dir("")))

	// Try and serve port 80.
	err := http.ListenAndServe(":80", nil)
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
