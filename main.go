package main

import (
	"flag"
	"fmt"
	"os"
	"pg-migrate/internal"
)

func main() {
    if len(os.Args) < 2 {
        fmt.Println("invalid command")
        os.Exit(1)
    }

    switch os.Args[1] {
    case "migrate":
        handleMigrate(os.Args[2:])
    default:
        fmt.Println("invalid command")
        os.Exit(1)
    }
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
