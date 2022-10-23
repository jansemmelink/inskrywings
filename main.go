package main

import (
	ussd "github.com/jansemmelink/ussd2"
	_ "github.com/jansemmelink/ussd2/ms/console"
	sessions "github.com/jansemmelink/ussd2/rest-sessions/client"
	"github.com/jansemmelink/utils2/errors"
	"github.com/jansemmelink/utils2/logger"
	_ "github.com/jansemmelink/utils2/ms/nats"
	_ "github.com/jansemmelink/utils2/ms/rest"
)

var mainMenu ussd.Item

var log = logger.New() //.WithLevel(logger.LevelDebug)

func main() {
	//logger.SetGlobalLevel(logger.LevelDebug)

	// ...custom types sent to / retrieved from rest-session does not maintain their data types!
	// Need to get and partse into correct type

	//register data types for validators (before loading the menu)

	if err := ussd.LoadItems("./menu.json"); err != nil {
		panic(errors.Errorf("failed to load menu.json: %+v", err))
	}

	//get main menu defined in the JSON file:
	var ok bool
	mainMenu, ok = ussd.ItemByID("main", nil)
	if !ok {
		panic("missing main")
	}

	//register functions implemented in go
	// start := ussd.Func("start", start)

	//todo: before menu is displayed, ensure we got msisdn, needed to send SMS...
	//and possible load some user account details...

	//use external HTTP REST service for sessions
	s := sessions.New("http://localhost:8100")
	ussd.SetSessions(s)

	//create and run the USSD service:
	svc := ussd.NewService(mainMenu)
	if err := svc.Run(); err != nil {
		panic(errors.Errorf("failed to run: %+v", err))
	}
}
