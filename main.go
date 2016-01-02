/*
Eddie is a robot that helps kids with their homeworks.

connections:

screen: I2C
button: D2
rotary: A0

*/
package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hybridgroup/gobot"
	"github.com/hybridgroup/gobot/platforms/gpio"
	"github.com/hybridgroup/gobot/platforms/i2c"
	"github.com/mattetti/gobot/platforms/firmata"
)

func init() {
	curWord = randWord()
}

var (
	curWord      string
	specialChars = map[string]int{
		"é": 1,
		"ñ": 2,
		"ó": 3,
		"í": 4,
		"á": 5,
	}

	words = []string{
		"qué",
		"yo",
		"mi",
		"mis",
		"una",
		"un",
		"ver",
		"hacer",
		"jugar",
		"puedo",
		"puedes",
		"tiburón",
		"camello",
		"cebra",
		"hiena",
		"gorilla",
		"muchos",
		"hipopótamo",
		"animales",
		"columpios",
		"carritos",
		"pelota",
		"escondidas",
		"patín",
		"contigo",
		"tobogán",
		"en",
		"a",
		"el",
		"las",
		"los",
		"con",
	}
)

func main() {
	gbot := gobot.NewGobot()

	// board := edison.NewEdisonAdaptor("edison")
	board := firmata.NewFirmataAdaptor("arduino", "/dev/cu.usbmodem1411")

	// button connected to D2
	button := gpio.NewButtonDriver(board, "button", "7")
	// screen connected to any of the I2C ports
	screen := i2c.NewGroveLcdDriver(board, "screen")
	// rotary angle sensor on A0
	rot := gpio.NewGroveRotaryDriver(board, "rotary", "1")
	rand.Seed(time.Now().UnixNano())

	displayQ := make(chan bool)
	quitQ := make(chan bool)

	// catch signals and terminate the robot
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)
	go func() {
		s := <-sigc
		fmt.Println("signal:", s)
		quitQ <- true
	}()

	var displaying bool
	var i int
	var scrollingPos int

	var queue = func() {
		fmt.Println("starting the display queue")
		for {
			select {
			case <-displayQ:
				//if displaying {
				//fmt.Println("off\n\n\n")
				//screen.SetRGB(255, 0, 0)
				//screen.Home()
				//screen.Clear()
				//displaying = false
				//} else {
				screen.SetRGB(rand.Intn(256), rand.Intn(256), rand.Intn(256))
				screen.Home()
				screen.Clear()
				w := curWord
				for w == curWord {
					w = randWord()
				}
				curWord = w
				if err := screen.Write(w); err != nil {
					log.Fatal(err)
				}
				i++
				displaying = true
				//}
			case <-quitQ:
				screen.Clear()
				<-time.After(2 * time.Millisecond)
				screen.SetRGB(255, 255, 255)
				<-time.After(50 * time.Millisecond)
				fmt.Println("ciao!")
				os.Exit(0)
				//case <-time.After(50 * time.Millisecond):
				//fmt.Println(".")
			}
		}
	}
	go queue()

	work := func() {
		screen.SetCustomChar(0, i2c.CustomLCDChars["heart"])
		screen.SetCustomChar(1, i2c.CustomLCDChars["é"])
		screen.SetCustomChar(2, i2c.CustomLCDChars["ñ"])
		screen.SetCustomChar(3, i2c.CustomLCDChars["ó"])
		screen.SetCustomChar(4, i2c.CustomLCDChars["í"])
		screen.SetCustomChar(5, i2c.CustomLCDChars["á"])
		screen.SetCustomChar(6, i2c.CustomLCDChars["smiley"])
		screen.SetCustomChar(7, i2c.CustomLCDChars["frownie"])
		screen.Clear()
		screen.SetRGB(20, 255, 0)
		screen.Write(fmt.Sprintf("Hola Giana %s\nHave fun!  %s", string(byte(0)), string(byte(6))))
		gobot.On(button.Event("push"), func(data interface{}) {
			fmt.Println("Next!")
			displayQ <- true
		})

		gobot.On(rot.Event("data"), func(data interface{}) {
			newPos, ok := data.(int)
			if displaying && ok && newPos != scrollingPos {
				if moved := intAbs(newPos - scrollingPos); moved > 5 {
					toTheRight := newPos > scrollingPos
					// TODO: proportional scrolling
					if err := screen.Scroll(toTheRight); err != nil {
						fmt.Println(err)
					}
					<-time.After(50 * time.Millisecond)
					fmt.Println("Scrolled to right:", toTheRight, data, moved)
					//screen.Write(".")
				}
				scrollingPos = newPos
			}
		})

	}

	robot := gobot.NewRobot("Eddie",
		[]gobot.Connection{board},
		[]gobot.Device{button, screen, rot},
		work,
	)

	gbot.AddRobot(robot)
	if err := gbot.Start(); err != nil {
		quitQ <- true
	}
}

func intAbs(x int) uint {
	switch {
	case x < 0:
		return uint(-x)
	case x == 0:
		return 0 // return correctly abs(-0)
	}
	return uint(x)
}

func randWord() string {
	word := words[rand.Intn(len(words))]
	newWord := []byte{}
	for _, r := range word {
		var b byte
		if pos, ok := specialChars[string(r)]; ok {
			b = byte(pos)
		} else {
			b = byte(r)
		}
		newWord = append(newWord, b)
	}
	return string(newWord)
}
