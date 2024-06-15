package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/link2618/bookings/pkg/config"
	"github.com/link2618/bookings/pkg/driver"
	"github.com/link2618/bookings/pkg/handlers"
	"github.com/link2618/bookings/pkg/helpers"
	"github.com/link2618/bookings/pkg/models"
	"github.com/link2618/bookings/pkg/render"

	"github.com/alexedwards/scs/v2"
)

const portNumber = ":3000"

var app config.AppConfig
var session *scs.SessionManager
var infoLog *log.Logger
var errorLog *log.Logger

// main is the main function
func main() {
	error := godotenv.Load()
	if error != nil {
		panic(error)
	}

	db, err := run()
	if err != nil {
		log.Fatal(err)
	}
	defer db.SQL.Close()
	defer close(app.MailChan)

	fmt.Println("Staring mail listener...")
	listenForMail()

	fmt.Println(fmt.Sprintf("Starting application on port %s", portNumber))

	srv := &http.Server{
		Addr:    portNumber,
		Handler: routes(&app),
	}

	err = srv.ListenAndServe()
	log.Fatal(err)
}

func run() (*driver.DB, error) {
	// what am I going to put in the session
	gob.Register(models.Reservation{})
	gob.Register(models.User{})
	gob.Register(models.Room{})
	gob.Register(models.Restriction{})
	gob.Register(map[string]int{})

	// read flags
	inProduction := flag.Bool("production", true, "Application is in production")
	useCache := flag.Bool("cache", true, "Use template cache")
	dbHost := flag.String("dbhost", os.Getenv("DB_HOST"), "Database host")
	dbName := flag.String("dbname", os.Getenv("DB_NAME"), "Database name")
	dbUser := flag.String("dbuser", os.Getenv("DB_USER"), "Database user")
	dbPass := flag.String("dbpass", os.Getenv("DB_PASS"), "Database password")
	dbPort := flag.String("dbport", os.Getenv("DB_PORT"), "Database port")
	dbSSL := flag.String("dbssl", os.Getenv("DB_SSL"), "Database ssl settings (disable, prefer, require)")

	flag.Parse()

	if *dbName == "" || *dbUser == "" {
		fmt.Println("Missing required flags")
		os.Exit(1)
	}

	mailChan := make(chan models.MailData)
	app.MailChan = mailChan

	// change this to true when in production
	app.InProduction = *inProduction
	app.UseCache = *useCache

	infoLog = log.New(os.Stdout, "INFO\t", log.Ldate|log.Ltime)
	app.InfoLog = infoLog

	errorLog = log.New(os.Stdout, "ERROR\t", log.Ldate|log.Ltime|log.Lshortfile)
	app.ErrorLog = errorLog

	session = scs.New()
	session.Lifetime = 24 * time.Hour
	session.Cookie.Persist = true
	session.Cookie.SameSite = http.SameSiteLaxMode
	session.Cookie.Secure = app.InProduction

	app.Session = session

	// connect to database
	log.Println("Connecting to database...")
	connectionString := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s", *dbHost, *dbPort, *dbName, *dbUser, *dbPass, *dbSSL)
	// log.Printf("Connection string: %s", connectionString)
	db, err := driver.ConnectSQL(connectionString)
	if err != nil {
		log.Fatal("Cannot connect to database! Dying...")
	}
	log.Println("Connected to database!")

	tc, err := render.CreateTemplateCache()
	if err != nil {
		log.Fatal("cannot create template cache")
		return nil, err
	}

	app.TemplateCache = tc

	repo := handlers.NewRepo(&app, db)
	handlers.NewHandlers(repo)
	render.NewRenderer(&app)
	helpers.NewHelpers(&app)

	return db, nil
}

// go mod tidy -> Borra librerias no usadas
