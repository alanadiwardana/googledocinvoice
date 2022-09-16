package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"googledocinvoice/invoice"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/docs/v1"
	"google.golang.org/api/option"
)

func loadInvoiceFile(path string) invoice.Invoice {

	file, err := os.OpenFile(path, os.O_RDONLY, os.ModeExclusive)
	if err != nil {
		log.Fatal("Arquivo de properties: " + path + " Com erro")
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	propsMap := map[string]string{}
	for scanner.Scan() {
		line := scanner.Text()
		propSlice := strings.Split(line, "=")
		propsMap[propSlice[0]] = propSlice[1]
	}
	//fmt.Printf("Full map: %v", propsMap)
	currInv := invoice.Invoice{
		TextTitle:   propsMap["TITLE"],
		PaidTo:      propsMap["PAID_TO"],
		BillTo:      propsMap["BILL_FROM"],
		ServiceDesc: propsMap["SERVICE_DESC"],
		ValuePaid:   propsMap["VALUE_PAID"],
		ValueAdds:   propsMap["VALUE_ADDS"],
		AddsDesc:    propsMap["ADDS_DESC"],
		City:        propsMap["CITY"],
		Locale:      propsMap["LOCALE"],
	}
	return currInv

}

// Retrieves a token, saves the token, then returns the generated client.
func getClient(config *oauth2.Config) *http.Client {
	tokFile := "token.json"
	tok, err := tokenFromFile(tokFile)
	if err != nil {
		tok = getTokenFromWeb(config)
		saveToken(tokFile, tok)
	}
	return config.Client(context.Background(), tok)
}

// Requests a token from the web, then returns the retrieved token.
func getTokenFromWeb(config *oauth2.Config) *oauth2.Token {
	authURL := config.AuthCodeURL("state-token", oauth2.AccessTypeOffline)
	fmt.Printf("Go to the following link in your browser then type the "+
		"authorization code: \n%v\n", authURL)

	var authCode string
	if _, err := fmt.Scan(&authCode); err != nil {
		log.Fatalf("Unable to read authorization code: %v", err)
	}

	tok, err := config.Exchange(oauth2.NoContext, authCode)
	if err != nil {
		log.Fatalf("Unable to retrieve token from web: %v", err)
	}
	return tok
}

// Retrieves a token from a local file.
func tokenFromFile(file string) (*oauth2.Token, error) {
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	tok := &oauth2.Token{}
	err = json.NewDecoder(f).Decode(tok)
	return tok, err
}

// Saves a token to a file path.
func saveToken(path string, token *oauth2.Token) {
	fmt.Printf("Saving credential file to: %s\n", path)
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	defer f.Close()
	if err != nil {
		log.Fatalf("Unable to cache OAuth token: %v", err)
	}
	json.NewEncoder(f).Encode(token)
}

func main() {
	invoice := loadInvoiceFile("info.txt")
	var inputdate = ""
	var currentTime time.Time
	if len(os.Args) > 1 {
		inputdate = os.Args[1]
		var timeErr error
		currentTime, timeErr = time.Parse("20060102", inputdate)
		if timeErr != nil {
			log.Fatal("Invalid Date: ", inputdate)
		}
	} else {
		inputdate = time.Now().Format("20060102")
		currentTime = time.Now()
	}
	invoice.CurrentDate = currentTime
	ctx := context.Background()
	b, err := ioutil.ReadFile("credentials.json")
	if err != nil {
		log.Fatalf("Unable to read client secret file: %v", err)
	}

	// If modifying these scopes, delete your previously saved token.json.
	config, err := google.ConfigFromJSON(b, "https://www.googleapis.com/auth/drive")
	if err != nil {
		log.Fatalf("Unable to parse client secret file to config: %v", err)
	}
	client := getClient(config)

	srv, err := docs.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		log.Fatalf("Unable to retrieve Docs client: %v", err)
	}

	// Prints the title of the requested doc:
	docId := "1LCT-iigKllQbs0KGjnQt1HpbuZiUC6V4dhGgVEK-XXI"
	doc, err := srv.Documents.Get(docId).Do()
	if err != nil {
		log.Fatalf("Unable to retrieve data from document: %v", err)
	}
	//title
	doc.Title = invoice.GetFullDocName()
	ass, newErr := srv.Documents.Create(doc).Do()
	if newErr != nil {
		log.Fatalf("Unable to create doc from template: %v", newErr)
	}
	//request geral
	rb := new(docs.BatchUpdateDocumentRequest)

	//title
	reqTitleText := new(docs.Request)
	//titleText := "RECIBO DE DIARISTA"
	reqTitleStyle := new(docs.Request)
	reqTitleParagraph := new(docs.Request)
	titleOffset := int64(len(invoice.TextTitle))
	SetReqTextMsg(reqTitleText, invoice.TextTitle, 1)
	SetReqTextStyle(reqTitleStyle, reqTitleParagraph, 26, true, "bold,fontSize", "CENTER", 1, titleOffset+1)
	//disclaimer
	disclaimerText := invoice.GetDisclaimerText()
	reqDisclaimerText := new(docs.Request)
	reqDisclaimerStyle := new(docs.Request)
	reqDisclaimerParagraph := new(docs.Request)
	disclaimerOffset := titleOffset + int64(len(disclaimerText))
	fmt.Println(titleOffset, disclaimerOffset)
	SetReqTextMsg(reqDisclaimerText, disclaimerText, titleOffset+1)
	SetReqTextStyle(reqDisclaimerStyle, reqDisclaimerParagraph, 12, false, "bold,fontSize", "JUSTIFIED", titleOffset+2, disclaimerOffset)
	//footerDate
	footerDateText := invoice.GetLocation()
	reqFooterText := new(docs.Request)
	reqFooterStyle := new(docs.Request)
	reqFooterParagraph := new(docs.Request)
	footerOffset := disclaimerOffset + int64(len(footerDateText))
	fmt.Println(titleOffset, disclaimerOffset, footerOffset)
	SetReqTextMsg(reqFooterText, footerDateText, disclaimerOffset-1)
	SetReqTextStyle(reqFooterStyle, reqFooterParagraph, 12, false, "bold,fontSize", "END", disclaimerOffset, footerOffset)
	//sign
	reqSignText := new(docs.Request)
	reqSignStyle := new(docs.Request)
	reqSignParagraph := new(docs.Request)
	signText := invoice.GetSignText()
	signOffset := footerOffset + int64(len(signText))
	fmt.Println(titleOffset, disclaimerOffset, footerOffset, signOffset)
	//footer
	SetReqTextMsg(reqSignText, signText, footerOffset-1)
	SetReqTextStyle(reqSignStyle, reqSignParagraph, 12, true, "bold,fontSize", "CENTER", footerOffset, signOffset)
	//do
	rb.Requests = append(rb.Requests, reqTitleText, reqTitleParagraph, reqTitleStyle, reqDisclaimerText, reqDisclaimerParagraph, reqDisclaimerStyle, reqFooterText, reqFooterParagraph, reqFooterStyle, reqSignText, reqSignParagraph, reqSignStyle)

	_, berr := srv.Documents.BatchUpdate(ass.DocumentId, rb).Do()
	if berr != nil {
		fmt.Println(berr)
	}
	fmt.Printf("The title of the created doc is: %s\n", ass.Title)

}

func SetReqTextMsg(txtRequest *docs.Request, msg string, index int64) {
	drt := new(docs.InsertTextRequest)
	drtl := new(docs.Location)
	drt.Text = msg
	drtl.Index = index
	drt.Location = drtl
	txtRequest.InsertText = drt
}

func SetReqTextStyle(
	request *docs.Request,
	parRequest *docs.Request,
	fontsize float64,
	bold bool,
	fields string,
	alignment string,
	start int64, end int64) {

	//text style
	tstyle := new(docs.TextStyle)
	tstyle.Bold = bold
	dim := new(docs.Dimension)
	dim.Magnitude = fontsize
	dim.Unit = "PT"
	tstyle.FontSize = dim

	//request text style
	utsr := new(docs.UpdateTextStyleRequest)
	utsr.TextStyle = tstyle
	rtsr := new(docs.Range)
	rtsr.StartIndex = start
	rtsr.EndIndex = end
	utsr.Range = rtsr
	utsr.Fields = fields
	request.UpdateTextStyle = utsr

	//request parag style
	upsr := new(docs.UpdateParagraphStyleRequest)
	rupsr := new(docs.Range)
	rupsr.StartIndex = start
	rupsr.EndIndex = end
	upsr.Range = rupsr
	upsr.Fields = "alignment"
	parStyle := new(docs.ParagraphStyle)
	parStyle.Alignment = alignment
	upsr.ParagraphStyle = parStyle
	parRequest.UpdateParagraphStyle = upsr

}
