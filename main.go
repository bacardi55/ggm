package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"net/smtp"
	"net/url"

	"git.sr.ht/~adnano/go-gemini"
	"github.com/pelletier/go-toml"
)

const invalidCertErrorMsg = "Certificate is unvalid"

// Nota: Gemini servers might have max execution time for cgi scripts.
// Eg: Gemserv has a 5s maximum policy before killing the request.
const maxRequestTime = 4

type GGMConfig struct {
	CapsuleRootAddress string
	MaxMentions        int
	Contact            string
	Log                string
	From               string
	SmtpServer         string
	Port               int
	Login              string
	Password           string
}

func main() {
	if os.Getenv("QUERY_STRING") == "" {
		fmt.Println("10\ttext/gemtext\r\n")
		fmt.Println("Enter the URL containing mentions: ")
		os.Exit(0)
	}

	// Retrieve configuration.
	// Default location is /etc/gogeminimention.toml
	// This can be override with GGM_CONFIG_PATH environment variable.
	configFilePath := "/etc/gogeminimention.toml"
	if ggmConfig := os.Getenv("GGM_CONFIG_PATH"); ggmConfig != "" {
		configFilePath = ggmConfig
	}

	configContent, err := getConfig(configFilePath)
	if err != nil {
		log.Println("Error loading configuration file:\n", err.Error())
		fmt.Println("An error occurred, please retry later…")
		endResponse()
	}

	config := GGMConfig{}
	toml.Unmarshal(configContent, &config)

	// Starting the response:
	fmt.Println("20\ttext/gemini\r\n")

	err = configureLogs(config.Log)
	if err != nil {
		// TODO: Should we stop if log file isn't found?
		log.Println("Log file not found!")
	}

	remoteUrl := os.Getenv("QUERY_STRING")
	log.Println("Received Gemini Mention:", remoteUrl)

	u, e := validateUrl(remoteUrl)
	if e != nil {
		fmt.Printf("Url is not valid.\r\n")
		log.Println("Received an unvalid url:", remoteUrl)
		endResponse()
	}

	response, err := fetchGeminiPage(u)

	if err != nil || response.Status.Class() != gemini.StatusSuccess {
		fmt.Printf("Error retrieving content from %v\n%s", u, err)
		log.Printf("Error retrieving content from", u, err)
		endResponse()
	}

	if respCert := response.TLS().PeerCertificates; len(respCert) > 0 && time.Now().After(respCert[0].NotAfter) {
		fmt.Printf("Invalid certificate for capsule:", u, "\n Link is ignored.")
		log.Println("Ignored url (invalid certificate for capsule):", u)
		endResponse()
	}

	// If all good, let's find gemini mentions link inside.
	var content []byte
	content, err = io.ReadAll(response.Body)
	defer response.Body.Close()

	if err != nil {
		fmt.Printf("Couldn't retrieve the provided URL content\n")
		log.Println("Couldn't retrieve the provided URL content:", err.Error())
		endResponse()
	}
	links := findMentionLinks(string(content), config.CapsuleRootAddress)
	if len(links) < 1 {
		fmt.Printf("No mention found in submitted link, ignoring.\n")
		log.Printf("Recieved a link with no mention:", u)
		endResponse()
	}

	// We have links, now let's check if they point to an existing page of our capsule:
	if len(links) < config.MaxMentions {
		config.MaxMentions = len(links)
	}
	//mentions := make([]string, config.MaxMentions)
	mentions := []string{}
	for i := 0; i < config.MaxMentions; i++ {
		l := extractLinkFromMention(links[i], config.CapsuleRootAddress)
		// Checking if it is a real existing link:
		resp, e := fetchGeminiPage(l)
		if e != nil || resp.Status.Class() != gemini.StatusSuccess {
			fmt.Println("Url mentioned doesn't exist on this capsule\n", l)
			continue
		}

		mentions = append(mentions, l)
	}
	if len(mentions) == 0 {
		fmt.Println("The mention link found in the remote url doesn't match any valid page of this capsule, no notification will be sent")
		endResponse()
	}

	err = notifyOwner(config, u, mentions)
	if err != nil {
		fmt.Println("An error occurred when sending the notification, sorry T_T.")
		log.Println("An error occurred when sending the notification", err)
		endResponse()
	}

	fmt.Println("The notification has successfully been sent, thank you for sharing!\n")
	fmt.Println("Your URL:\n=>", u)
	fmt.Println("You mentioned:")
	for _, v := range mentions {
		fmt.Println("=>", v)
	}
	fmt.Println("\n\n")
	endResponse()
}

func fetchGeminiPage(remoteUrl string) (*gemini.Response, error) {
	gemclient := &gemini.Client{}
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(maxRequestTime)*time.Second)
	response, err := gemclient.Get(ctx, remoteUrl)

	if err != nil {
		return response, err
	}

	if respCert := response.TLS().PeerCertificates; len(respCert) > 0 && time.Now().After(respCert[0].NotAfter) {
		return response, fmt.Errorf(invalidCertErrorMsg)
	}

	return response, nil
}

func validateUrl(remoteUrl string) (string, error) {
	remote, e := url.QueryUnescape(remoteUrl)
	if e != nil {
		return "", fmt.Errorf("Provided URL is not a good URL: %s", e)
	}
	remote = strings.Replace(remote, "..", "", -1)

	u, err := url.Parse(remote)

	if err != nil {
		return "", fmt.Errorf("Provided URL is not a good URL: %s", err)
	} else if u.Scheme != "gemini" && u.Scheme != "" {
		return "", fmt.Errorf("Only gemini url are supported for now.")
	} else {
		return "gemini://" + u.Host + u.Path, nil
	}
}

func endResponse() {
	fmt.Printf("=> /index.gmi Return to homepage\r\n")
	fmt.Printf("=> /.well-known/mentions Send another mention\r\n")
	os.Exit(0)
}

func findMentionLinks(content string, capsuleRootAddress string) []string {
	exp := "(?im)^(=>)[ ]?gemini://" + capsuleRootAddress + "[^ ]+[ ]RE:[ ]?(.*)$"
	re := regexp.MustCompile(exp)
	return re.FindAllString(content, -1)
}

func extractLinkFromMention(mention string, capsuleRootAddress string) string {
	exp := "(?im)^=>[ ]?(?P<mentionUrl>gemini://" + capsuleRootAddress + "[^ ]+)[ ]RE:[ ]?.*$"
	re := regexp.MustCompile(exp)
	matches := re.FindStringSubmatch(mention)
	index := re.SubexpIndex("mentionUrl")
	return matches[index]
}

func notifyOwner(config GGMConfig, responseLink string, mentionedLinks []string) error {
	mailContent := "To: " + config.Contact + "\r\n"
	mailContent += "Subject: New Gemini Mention!\r\n"
	mailContent += "\r\n"
	mailContent += "A fellow geminaut responded to " + strconv.Itoa(len(mentionedLinks)) + " of your post(s).\n"
	mailContent += "The following link(s) are mentioned within the response:\n"
	for _, v := range mentionedLinks {
		mailContent += v + "\n"
	}
	mailContent += "\nThe response link is:\n" + responseLink
	mailContent += "\r\n\r\n"

	auth := smtp.PlainAuth("", config.Login, config.Password, config.SmtpServer)
	to := []string{config.Contact}
	add := config.SmtpServer + ":" + strconv.Itoa(config.Port)

	err := smtp.SendMail(add, auth, config.From, to, []byte(mailContent))

	return err
}

// Configure Log file.
func configureLogs(logPath string) error {
	var logFile string
	if logPath != "" {
		logFile = logPath
	} else {
		logFile = "./ggm.log"
	}

	file, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0666)
	if err != nil {
		return err
	}
	log.SetOutput(file)

	return nil
}

func getConfig(configFile string) ([]byte, error) {
	log.Println("Trying to load provided configuration file:", configFile)
	content, err := os.ReadFile(configFile)
	if err != nil {
		log.Println("Couldn't load configuration file.\n", err.Error())
		return nil, err
	}

	return content, nil
}
