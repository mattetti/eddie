/*
Eddie is a robot that helps kids with their homeworks.

connections:

screen: I2C
button: D2
rotary: A0

*/
package main

import (
	"flag"
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
	levelFlag     = flag.Int("level", 0, "word/sentence level")
	curWord       string
	lastFailed    string
	lastFailedPos int
	wCounter      int
	specialChars  = map[string]int{
		"é": 1,
		"ñ": 2,
		"ó": 3,
		"í": 4,
		"á": 5,
		"ú": 7,
	}
)

func main() {
	flag.Parse()
	gbot := gobot.NewGobot()

	// board := edison.NewEdisonAdaptor("edison")
	board := firmata.NewFirmataAdaptor("arduino", "/dev/cu.usbmodem1411")

	// button connected to D7
	button := gpio.NewButtonDriver(board, "button", "7")
	// touch button D2
	touch := gpio.NewButtonDriver(board, "touch", "2")
	// screen connected to any of the I2C ports
	screen := i2c.NewGroveLcdDriver(board, "screen")
	// rotary angle sensor on A0
	rot := gpio.NewGroveRotaryDriver(board, "rotary", "2")
	rand.Seed(time.Now().UnixNano())

	redLed := gpio.NewLedDriver(board, "red", "8")

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
				if lastFailed != "" && wCounter == lastFailedPos {
					if err := screen.Write(lastFailed); err != nil {
						log.Fatal(err)
					}
					lastFailedPos = 0
					lastFailed = ""
				} else if wCounter > 10 {
					wCounter = 0
					sentence := randSentence()
					if err := screen.Write(sentence); err != nil {
						log.Fatal(err)
					}
				} else {
					w := randWord()
					for w == curWord {
						w = randWord()
					}
					curWord = w
					if err := screen.Write(w); err != nil {
						log.Fatal(err)
					}
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
		screen.SetCustomChar(7, i2c.CustomLCDChars["ú"])
		screen.Clear()
		screen.SetRGB(20, 255, 0)
		screen.Write(fmt.Sprintf("Hola Giana %s\nHave fun!  %s", string(byte(0)), string(byte(6))))
		gobot.On(button.Event("push"), func(data interface{}) {
			wCounter++
			fmt.Printf("Position: %d", wCounter)
			if lastFailed == "" {
				fmt.Println()
			} else {
				fmt.Printf(" last failed: %s [%d]\n", lastFailed, lastFailedPos)
			}
			displayQ <- true
			redLed.Off()
		})

		gobot.On(touch.Event("push"), func(data interface{}) {
			lastFailed = curWord
			lastFailedPos = wCounter
			screen.SetRGB(rand.Intn(256), rand.Intn(256), rand.Intn(256))
			redLed.On()
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
		[]gobot.Device{button, screen, touch, redLed, rot},
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

func randSentence() string {
	sentence := sentences[rand.Intn(len(sentences))]
	newSentence := []byte{}
	for _, r := range sentence {
		var b byte
		if pos, ok := specialChars[string(r)]; ok {
			b = byte(pos)
		} else {
			b = byte(r)
		}
		newSentence = append(newSentence, b)
	}
	return string(newSentence)
}

var words = []string{
	"a",
	"animales",
	"amarillo",
	"amarilla",
	"ama",
	"amo",
	"anaranjada",
	"anaranjado",
	"abajo",
	"al",
	"azul",
	"amigos",
	"abeja",
	"árbol",
	"arbolito",
	"adentro",
	"Adrían",
	"Adriana",
	"Ariana",
	"Arco iris",
	"arriba",

	"con",
	"columpios",
	"carritos",
	"contigo",
	"camello",
	"cebra",
	"chocolate",
	"café",
	"caer",
	"cuatro",
	"círculo",
	"cuadrado",

	"dos",
	"del",
	"debajo",

	"escondidas",
	"en",
	"el",
	"es",
	"escoba",
	"elefante",
	"escalera",
	"estrellas",
	"escuela",
	"enfermera",
	"este",
	"Exjani",
	"Emma",

	"gorilla",
	"gato",
	"gusta",
	"que",

	"hiena",
	"hacer",
	"hipopótamo",
	"hoja",
	"hasta",

	"iguana",
	"iguanas",
	"iglú",
	"insectos",
	"isla",
	"idea",
	"igual",
	"instrumento",

	"jugar",

	"la",
	"las",
	"los",
	"libro",
	"lo",

	"mi",
	"mí",
	"mis",
	"me",
	"más",
	"murciélago",
	"muchos",
	"mapa",
	"manzana",
	"mano",
	"manos",
	"mariposa",
	"mango",
	"muñeca",
	"mono",
	"mamá",
	"Mireya",
	"Marissa",
	"Maren",
	"Melina",

	"negro",
	"nariz",

	"oso",
	"ocho",
	"oveja",
	"ojo",
	"once",
	"oído",
	"óvalo",
	"oruga",
	"ola",
	"otoño",

	"puedo",
	"pelota",
	"perro",
	"pero",
	"pelo",
	"puedes",
	"patín",
	"pavo",
	"pato",
	"papá",
	"papa",
	"puma",
	"pomo",
	"payaso",
	"pavo",
	"pluma",

	"qué",

	"rana",
	"roja",
	"rojo",
	"regalos",
	"réctangulo",

	"silla",
	"suelo",
	"sé",
	"sombrero",

	"tobogán",
	"tiburón",
	"tres",
	"triángulo",
	"tengo",
	"traje",

	"una",
	"uña",
	"uno",
	"un",
	"uvas",
	"último",
	"unicornio",
	"universo",

	"ver",
	"verde",

	"yo",

	"zapatos",
}

var sentences = []string{
	"Me gusta ver una rana.",
	"Me gusta ver un perro.",
	"Me gusta ver un gato.",
	"Me gusta ver un oso.",
	"Me gusta ver una silla.",
	"Pero lo que más me gusta ver es un libro.",
	"A mí me gusta el chocolate.",

	"Que puedes hacer?",
	"Yo puedo jugar en los columpios.",
	"Yo puedo jugar con la pelota.",
	"Yo puedo jugar con mis carritos.",
	"Yo puedo jugar a las escondidas.",
	"Yo puedo jugar con mi patín.",
	"Yo puedo jugar en el tobogán.",
	"Yo puedo jugar contigo.",

	"Me gusta el pato amarillo.",
	"Me gusta la pelota roja.",
	"Me gusa el pavo café.",
	"Me gusta el perro negro.",

	"Qué puedes ver?",
	"Yo puedo ver una cebra.",
	"Yo puedo ver un camello.",
	"Yo puedo ver un tiburón.",
	"Yo puedo ver un hipopótamo.",
	"Yo puedo ver una hiena.",
	"Yo puedo ver un gorila.",
	"Yo puedo ver muchos animales.",

	"Hojas de otoño",
	"Una hoja verde.",
	"Una hoja roja.",
	"Una hoja amarilla.",
	"Una hoja anaranjada.",
	"Una hoja café.",
	"Hasta caer al suelo.",

	"Tengo cuatro regalos.",
	"Debajo del arbolito.",
	"Uno es un círculo.",
	"Qué está adentro?",
	"Yo no sé.",
	"Uno es un cuadrado.",
	"Uno es un retángulo.",
	"Uno es un triángulo.",
	"Tengo cuatro regalos debajo del arbolito.",
	"Todos son para mí!",

	"Es una manzana roja.",
	"Es una manzana verde.",
	"Es una manzana blanca.",
	"Es una manzana café.",

	"Este payaso tiene el sombrero morado.",
	"Este payaso tiene pelo verde.",
	"Este payaso tiene la nariz roja.",
	"Este payaso tiene la manos cafés.",
	"Este payaso tiene el traje amarillo.",
	"Este payaso tiene los zapatos azules.",
	"Este es un payaso arco iris!",

	"El pavo tiene una pluma roja.",
	"El pavo tiene una pluma verde.",
	"El pavo tiene una pluma anaranjada.",
	"El pavo tiene una pluma azul.",
	"El pavo tiene una pluma café.",
	"El pavo tiene una pluma amarilla.",
	"El pavo tiene una pluma morada.",
	"Qué pavo más rico!",

	"Yo veo la mariposa.",
	"Me gusta la mariposa.",
	"Yo veo el mango.",
	"Me gusta el mango.",
	"Yo veo el mono.",
	"Me gusta el mono.",
	"Yo veo la muñeca.",
	"Me gusta la muñeca.",
}

var silabas = []string{
	"pa", "pe", "pi", "po", "pu", "ma", "me", "mi", "mo", "mu",
}
