package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"pg-migrate/internal"
)

type Config struct {
	CurrentVersion int    `json:"current_version"`
	ConnURL        string `json:"conn_url"`
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("invalid command")
		os.Exit(1)
	}

	switch os.Args[1] {
	case "migrate":
		handleMigrate(os.Args[2:])
	case "init":
		handleInit(os.Args[2:])
	default:
		fmt.Println("invalid command")
		os.Exit(1)
	}
}

func handleInit(args []string) {
	fs := flag.NewFlagSet("init", flag.ExitOnError)
	sourcePtr := fs.String("source", "./", "source directory")
	fs.Parse(args)

    migrationsDirectory := *sourcePtr + "/migrations"

	err := os.Mkdir(migrationsDirectory, 0750)
	if err != nil && !os.IsExist(err) {
        fmt.Println("error creating migration directory:", err)
        os.Exit(1)
	}

    var config Config
    json, err := json.MarshalIndent(config, "", "\t")
    if err != nil {
        fmt.Println("error encoding migration config to json:", err)
        os.Exit(1)
    }

    err = os.WriteFile(migrationsDirectory + "/migrate.config", json, 0660)
    if err != nil {
        fmt.Println("error creating migration config file:", err)
        os.Exit(1)
    }

    fmt.Printf("successfully created migraitons directory: %s\n", migrationsDirectory)
    fmt.Println(`To get Started, edit the migrate.config file in the migrations directory. If using env varaible, then pass "$ENV_VARIABLE"`)
}

func handleMigrate(args []string) {
	fs := flag.NewFlagSet("migrate", flag.ExitOnError)
	sourcePtr := fs.String("source", "./", "source directory")
	databasePtr := fs.String("database", "", "database connection url")
	fs.Parse(args)

	if *sourcePtr == "" {
		fmt.Println("Invalid source directory:", *sourcePtr)
		return
	}

	if *databasePtr == "" {
		fmt.Println("Invalid database connection url:", *databasePtr)
		return
	}

	migrater, err := internal.NewMigrater(*sourcePtr, *databasePtr)
	if err != nil {
		fmt.Println("error creating migrater")
		return
	}

	migrater.RunMigrations(internal.UP)
}
