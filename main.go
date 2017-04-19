package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"github.com/Syfaro/telegram-bot-api"
	"github.com/julienschmidt/httprouter"
	"github.com/nathan-osman/go-rpigpio"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	"net/url"
	"io/ioutil"
	"os/exec"
	"math/rand"
)

var Port string
var Address string
var ConfigFile string
var CurrentPumps Pumps
var isReadyToDo bool
var totalDuration int
var BotToken string
var SpeechKitToken string
var htmlHeader = `<html><head><style>
	* {font-family: Verdana}
	a.button {font-size: 8em;color: #fff;text-decoration: none;user-select: none;background: rgb(76,175,80);padding: .7em 1.5em;outline: none;font-family: Verdana;}
	a.button:hover { background: rgb(232,95,76); }
	a.button:active { background: rgb(152,15,0); }
	.container {display: flex;align-items: center;justify-content: center;height: 100%;}</style>
	<title>NALIVATOR-9000</title>`

type Pumps struct {
	Cname string `json:"cname"`
	Pumps []Pump `json:"pumps"`
}

type Pump struct {
	Name     string
	Pump_pin int
	Led_pin  int
	Duration int
}

func init() {
	flag.StringVar(&Port, "port", "8181", "Listen port")
	flag.StringVar(&Address, "address", "0.0.0.0", "Listen address")
	flag.StringVar(&ConfigFile, "config", "config.json", "Config file")
	flag.StringVar(&BotToken, "bottoken", "", "Telegram bot token")
	flag.StringVar(&SpeechKitToken, "speechkittoken","","Yandex SpeechKit token. Using Yandex SpeechKit Cloud - https://tech.yandex.ru/speechkit/cloud/")
	flag.Parse()

	file, err := os.Open(ConfigFile)
	if err != nil {
		log.Printf("Failed to read config file %s: %v", ConfigFile, err)
		os.Exit(1)
	}
	decoder := json.NewDecoder(file)

	log.Printf("Reading config from file: " + ConfigFile)

	err = decoder.Decode(&CurrentPumps)
	if err != nil {
		log.Printf("Failed to parse JSON config %s: %v", ConfigFile, err)
		os.Exit(1)
	}
	log.Printf("%+v", CurrentPumps)

	totalDuration = 0
	for _, v := range CurrentPumps.Pumps {
		totalDuration += v.Duration
	}
	totalDuration = totalDuration * 1000
	log.Printf("Total duration: %v ms", totalDuration)

	isReadyToDo = true



}

func telegram_bot() {
	log.Printf("Telegram bot token: %s", BotToken)
	bot, err := tgbotapi.NewBotAPI(BotToken)
	if err != nil {
		log.Printf("%v", err)
		log.Print("Can't register bot. Bot will not work")
		return
	}

	log.Printf("Starting telegram bot, @%s", bot.Self.UserName)
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil {
			continue
		}
		reply := ""

		log.Printf("[%s] %s", update.Message.From.UserName, update.Message.Text)

		if strings.Contains(update.Message.Text, CurrentPumps.Cname) {
			if isReadyToDo {
				reply = "Подставляй стакан! Сейчас налью"
				go do_cocktail()
			} else {
				reply = "Прости, я пока занят. Видишь сколько набежало!"
			}
		}

		switch update.Message.Command() {
		case "start":
			reply = "Привет!\nМеня зовут Гоша Наливатор.\nОтец создал меня чтобы наливать людям лучшие коктейли в городе. Пока что я умею только " + CurrentPumps.Cname + ".\n" +
				"Просто напиши мне в телеграмчик: \"Гоша, " + CurrentPumps.Cname + "\" , и я сделаю его для тебя"
		case "help":
			reply = "Я - Гоша. Ты что, забыл? Просто напиши название коктейля который ты хочешь чтобы я сделал"
		case "joke":
			reply = "Сто+ит бармен и протирает барную стойку. Подходит подвыпивший мужик. Смотрит и после недолгого размышления говорит бармену: - Ты в курсе что ты смахиваешь на г+омика? Бармен спокойно и вежливо отвечает:- Извините, мужчина, я буду смахивать в другую сторону. Ха. Ха. Ха. Ха."
		}

		if reply == "" && update.Message.Text != "" {
			reply = "Ты че несешь? Давай накатим?"
		}

		if SpeechKitToken != "" {
			go do_audio(reply)
		}

		msg := tgbotapi.NewMessage(update.Message.Chat.ID, reply)
		bot.Send(msg)
	}
}

func main() {
	r := httprouter.New()

	log.Printf("Starting NALIVATOR-9000 web server on %s:%s..", Address, Port)

	if BotToken != "" {
		go telegram_bot()
	}

	if SpeechKitToken != "" {
		log.Printf("SpeechKit token: %s", SpeechKitToken)
	}

	//startup leds blink
	leds_on(CurrentPumps)
	time.Sleep(time.Second * 2)
	leds_off(CurrentPumps)

	r.GET("/config", ConfigHandler)
	r.GET("/do", DoCocktailHandler)
	r.GET("/", HomeHandler)


	log.Print(http.ListenAndServe(Address+":"+Port, r))
}

func do_audio(texttospeech string) {
	log.Printf("Text to speech: %s", texttospeech)
	audio_tmp := "/tmp/audio_tmp.wav"
	hexurl := url.QueryEscape(texttospeech)

	resp, err := http.Get("https://tts.voicetech.yandex.net/generate?key="+SpeechKitToken+"&format=wav&quality=hi&lang=ru-RU&speaker=ermil&text=" + hexurl)
	if err != nil {
		log.Print("Something gone wrong with SpeechAPI")
	}
	defer resp.Body.Close()
	log.Printf("Response from Yandex SpeechKit API: %s", resp.Status)
	body, err := ioutil.ReadAll(resp.Body)

	//calculating duration of wav file
	log.Printf("Length of api response=%d",len(body))
	wav_duration := ( ( len(body) / 88000 ) * 1000 ) - 500
	log.Printf("Wav duration: %d ms", wav_duration)

	//save speechapi result in file
	err = ioutil.WriteFile(audio_tmp, body, 0644)

	//start blinking led
	go do_led(wav_duration)

	//play speech file
	out, err := exec.Command("sh","-c","aplay "+audio_tmp).Output()
	log.Print(out)
}

func leds_on(P Pumps) {
	log.Print("LED - Leds on")
	for _, v := range P.Pumps {
		p, err := rpi.OpenPin(v.Led_pin, rpi.OUT)
		if err != nil {
			log.Print("LED - Can't set LED pin to output")
		}
		p.Write(rpi.HIGH)
	}
}

func leds_off (P Pumps) {
	log.Print("LED - Leds off")
	for _, v := range CurrentPumps.Pumps {
		p, err := rpi.OpenPin(v.Led_pin, rpi.OUT)
		if err != nil {
			log.Print("LED - Can't set LED pin to output")
		}
		p.Write(rpi.LOW)

	}
}

func do_led(duration int) {
	inc := 0
	i := 0
	log.Printf("==== LED - Doing led, duration: %d ms ====", duration)
	for i < duration {

		//leds on
		leds_on(CurrentPumps)

		//sleep rand ms
		inc = rand.Intn(600)
		log.Printf("LED - Sleep: %d ms", inc)
		time.Sleep(time.Millisecond * time.Duration(inc))

		i = i + inc

		//leds off
		leds_off(CurrentPumps)

		//sleep rand ms
		inc = rand.Intn(300)
		log.Printf("LED - Sleep: %d ms", inc)
		time.Sleep(time.Millisecond * time.Duration(inc))

		i = i + inc

	}

	log.Print("==== LED - Done ====")
}

func do_cocktail() {
	isReadyToDo = false
	time.Sleep(time.Second * 5)
	log.Print("==== Start coocking ====")
	for _, v := range CurrentPumps.Pumps {
		log.Printf("Nalivaem %s ;duration = %v; GPIO = %v", v.Name, v.Duration, v.Pump_pin)
		//pump pin open
		p, err := rpi.OpenPin(v.Pump_pin, rpi.OUT)
		if err != nil {
			panic(err)
		}
		defer p.Close()

		//led pin open
		l, l_err := rpi.OpenPin(v.Led_pin, rpi.OUT)
		if l_err != nil {
			log.Print("LED - Can't set LED pin to output")
		}
		defer l.Close()

		// pump on
		p.Write(rpi.HIGH)

		//led on
		l.Write(rpi.HIGH)

		time.Sleep(time.Second * time.Duration(v.Duration))

		// pump off
		p.Write(rpi.LOW)

		//led off
		l.Write(rpi.LOW)
	}
	do_audio("Твой коктейль готов!")

	log.Print("==== Done ====")
	isReadyToDo = true
}

func DoCocktailHandler(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	if isReadyToDo == true {
		go do_cocktail()
		fmt.Fprint(rw, htmlHeader)
		fmt.Fprintf(rw, "<script type=\"text/JavaScript\">setTimeout(\"location.href = '/';\", %v);</script></head><body><div class=\"container\"><div><h1>Doing</h1></div></div></body></html>", totalDuration)

	} else {
		fmt.Fprint(rw, htmlHeader+"<script type=\"text/JavaScript\">setTimeout(\"location.href = '/';\",5000);</script></head><body><div class=\"container\"><div><h1>Sorry, i'm busy. Please try again</h1></div></div></body></html>")
	}
}

func ConfigHandler(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprint(rw, CurrentPumps)
}

func HomeHandler(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	fmt.Fprint(rw, htmlHeader)
	fmt.Fprint(rw, "</head><body><div class=\"container\"><div><a href=\"/do\" class=\"button\">Налить</a></div></div></body></html>")
}
